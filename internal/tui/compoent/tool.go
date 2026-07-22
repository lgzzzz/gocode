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
	id          string
	name        string
	args        string
	result      string
	state       ToolState
	renderCache string
	renderWidth int
	dirty       bool
}

func NewToolMessage(id, name, args string) *ToolMessage {
	return &ToolMessage{
		id:    id,
		name:  name,
		args:  args,
		dirty: true,
	}
}

// SetResult updates the tool message with the execution result.
// It falls back to pattern-matching on the result string to detect errors,
// but callers should prefer using SetError() via CallbackMsg.Err for accuracy.
func (m *ToolMessage) SetResult(result string) {
	if m.result == result && m.state != ToolStateExecuting {
		return
	}
	m.result = result
	if strings.HasPrefix(result, "Error:") ||
		strings.HasPrefix(result, "exit ") ||
		strings.HasPrefix(result, "(timed out") {
		m.state = ToolStateError
	} else {
		m.state = ToolStateSuccess
	}
	m.invalidateCache()
}

// SetError explicitly marks the tool as having errored.
// Call this after SetResult when CallbackMsg.Err is non-nil.
func (m *ToolMessage) SetError() {
	if m.state == ToolStateError {
		return
	}
	m.state = ToolStateError
	m.invalidateCache()
}

func (m *ToolMessage) Type() string  { return "tool" }
func (m *ToolMessage) MsgID() string { return m.id }

func (m *ToolMessage) SetContent(content string) {
	if m.result == content {
		return
	}
	m.result = content
	m.invalidateCache()
}

func (m *ToolMessage) invalidateCache() {
	if m.renderWidth > 0 {
		m.renderCache = m.renderLocked(m.renderWidth)
	} else {
		m.dirty = true
	}
}

func (m *ToolMessage) Render(width int) string {
	if !m.dirty && width == m.renderWidth {
		return m.renderCache
	}
	m.renderWidth = width
	m.renderCache = m.renderLocked(width)
	m.dirty = false
	return m.renderCache
}

const maxToolResultLines = 6

// renderLocked builds the rendered string for the given width.
func (m *ToolMessage) renderLocked(width int) string {
	argsPreview := truncateStr(m.args, 150)
	content := m.name + "(" + argsPreview + ")"

	if m.state != ToolStateExecuting && m.result != "" {
		lines := strings.Split(m.result, "\n")
		if len(lines) > maxToolResultLines {
			skipped := len(lines) - maxToolResultLines
			lines = append(lines[:maxToolResultLines],
				fmt.Sprintf("...%d more lines...", skipped))
		}
		content += "\n" + strings.Join(lines, "\n")
	}

	style := toolStyle
	if m.state == ToolStateError {
		style = toolErrorStyle
	}
	return strings.TrimSpace(style.Width(width - 1).Render(strings.TrimSpace(content)))
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
