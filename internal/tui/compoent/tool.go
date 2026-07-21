package compoent

import (
	"fmt"
	"strings"
)

// ---- Tool states ----

type ToolState int

const (
	ToolStateExecuting ToolState = iota
	ToolStateSuccess
	ToolStateError
)

// ToolMessage merges a tool call and its result into a single block.
// Use NewToolMessage to create one in "executing" state, then call
// SetResult when the tool result arrives to transition to success/error.
type ToolMessage struct {
	ID     string
	Name   string
	Args   string
	Result string
	State  ToolState
}

func NewToolMessage(id, name, args string) *ToolMessage {
	return &ToolMessage{
		ID:    id,
		Name:  name,
		Args:  args,
		State: ToolStateExecuting,
	}
}

// SetResult updates the tool message with the execution result.
// It falls back to pattern-matching on the result string to detect errors,
// but callers should prefer using SetError() via CallbackMsg.Err for accuracy.
func (m *ToolMessage) SetResult(result string) {
	m.Result = result
	if strings.HasPrefix(result, "Error:") ||
		strings.HasPrefix(result, "exit ") ||
		strings.HasPrefix(result, "(timed out") {
		m.State = ToolStateError
	} else {
		m.State = ToolStateSuccess
	}
}

// SetError explicitly marks the tool as having errored.
// Call this after SetResult when CallbackMsg.Err is non-nil.
func (m *ToolMessage) SetError() {
	m.State = ToolStateError
}

func (m *ToolMessage) Type() string  { return "tool" }
func (m *ToolMessage) MsgID() string { return m.ID }

const maxToolResultLines = 6

func (m *ToolMessage) Render(width int) string {
	argsPreview := truncateStr(m.Args, 150)
	content := m.Name + "(" + argsPreview + ")"

	if m.State != ToolStateExecuting && m.Result != "" {
		// Truncate result to avoid taking too much vertical space
		lines := strings.Split(m.Result, "\n")
		if len(lines) > maxToolResultLines {
			skipped := len(lines) - maxToolResultLines
			lines = append(lines[:maxToolResultLines],
				fmt.Sprintf("...%d more lines...", skipped))
		}
		content += "\n" + strings.Join(lines, "\n")
	}

	style := toolStyle
	if m.State == ToolStateError {
		style = toolErrorStyle
	}
	return strings.TrimRight(style.Width(width-1).Render(content), "\n")
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
