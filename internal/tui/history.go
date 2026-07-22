package tui

import "github.com/lgzzzz/gocode/internal/tui/compoent"

// ---- ModelAccess interface implementation ----

// ClearHistory clears the TUI message history.
func (m *model) ClearHistory() { m.history.Clear() }

// AppendSystemMessage appends a system message to the chat history.
func (m *model) AppendSystemMessage(content string) {
	m.history.Append(compoent.NewSystemMessage(content))
}
