package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lgzzzz/gocode/internal/store"
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)

// ---- ModelAccess interface implementation ----

// ClearHistory clears the TUI message history.
func (m *model) ClearHistory() { m.history.Clear() }

// NewSession swaps to a new session ID and clears TUI history.
// The actual DB session row is created lazily when the first message is sent.
func (m *model) NewSession() {
	m.history.Clear()
	m.sessionID = store.NewSessionID()
	m.cwd, _ = os.Getwd()
}

// AppendSystemMessage appends a system message to the chat history.
func (m *model) AppendSystemMessage(content string) {
	m.history.Append(compoent.NewSystemMessage(content))
}

// ListSessions returns a formatted string listing recent sessions.
func (m *model) ListSessions() string {
	if m.store == nil {
		return ""
	}
	sessions, err := m.store.ListSessions(20)
	if err != nil || len(sessions) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("📋 最近会话:\n\n")

	for _, s := range sessions {
		t, _ := time.Parse(time.RFC3339, s.CreatedAt)
		timeStr := t.Local().Format("2006-01-02 15:04")
		marker := ""
		if s.ID == m.sessionID {
			marker = " ◀ 当前"
		}
		sb.WriteString(fmt.Sprintf("  %s  %s  %d 条消息  %s%s\n",
			timeStr, s.Model, s.MessageCount, s.CWD, marker))
	}
	return sb.String()
}
