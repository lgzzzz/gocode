package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/store"
	"github.com/lgzzzz/gocode/internal/tui/compoent"
	"github.com/lgzzzz/gocode/internal/tui/sessionbrowser"
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

// ---- session browser ----

// EnterSessionBrowser loads sessions from the store and activates
// the interactive session browser, replacing the output viewport.
func (m *model) EnterSessionBrowser() {
	if m.store == nil {
		m.history.Append(compoent.NewSystemMessage("📭 会话存储不可用"))
		return
	}
	sessions, err := m.store.ListSessions(50)
	if err != nil {
		m.history.Append(compoent.NewErrorMessage(err.Error()))
		return
	}
	if len(sessions) == 0 {
		m.history.Append(compoent.NewSystemMessage("📭 暂无历史会话"))
		return
	}
	m.sessionBrowser = sessionbrowser.New(m.width, m.output.Height())
	m.sessionBrowser.SetSessions(sessions)
}

// ExitSessionBrowser deactivates the session browser and restores
// the normal output viewport.
func (m *model) ExitSessionBrowser() {
	m.sessionBrowser.SetActive(false)
}

// LoadSession loads all messages from the given session and rebuilds
// both the TUI history and the agent's conversation history.
func (m *model) LoadSession(sessionID string) {
	if m.store == nil {
		return
	}

	msgs, err := m.store.GetSessionMessages(sessionID)
	if err != nil {
		m.history.Append(compoent.NewErrorMessage(err.Error()))
		return
	}

	// 1. Rebuild TUI history: iterate and pair tool_call/tool_result.
	m.history.Clear()
	for i := 0; i < len(msgs); i++ {
		msg := msgs[i]
		switch msg.MsgType {
		case "user":
			m.history.Append(compoent.NewUserMessage(msg.Content))
		case "assistant":
			if msg.Content != "" {
				m.history.Append(compoent.NewAssistantMessage(msg.ToolCallID, msg.Content))
			}
		case "thinking":
			m.history.Append(compoent.NewThinkingMessage(msg.ToolCallID, msg.Content))
		case "tool_call":
			tm := compoent.NewToolMessage(msg.ToolCallID, msg.ToolName, msg.ToolArgs)
			m.history.Append(tm)
		case "tool_result":
			hasErr := msg.HasError
			m.history.UpdateToolResult(msg.ToolCallID, msg.Content, hasErr)
		}
	}

	// 2. Rebuild Agent history.
	agentMsgs := make([]agent.HistoryMessage, len(msgs))
	for i, m := range msgs {
		agentMsgs[i] = agent.HistoryMessage{
			MsgType:    m.MsgType,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
			ToolName:   m.ToolName,
			ToolArgs:   m.ToolArgs,
		}
	}
	openaiHistory := agent.ReconstructHistory(agentMsgs, m.agent.SystemPrompt())
	m.agent.SetHistory(openaiHistory)

	// 3. Switch to the loaded session.
	m.sessionID = sessionID
	m.sessionBrowser.SetActive(false)
	m.output.GotoBottom()
}
