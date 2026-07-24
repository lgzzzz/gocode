package compoent

import (
	"encoding/json"
	"fmt"
	"strings"
)


type ToolState int

const (
	ToolStateExecuting ToolState = iota
	ToolStateSuccess
	ToolStateError
)

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

func (m *ToolMessage) SetError() {
	if m.state == ToolStateError {
		return
	}
	m.state = ToolStateError
	m.invalidateCache()
}

func (m *ToolMessage) Type() string    { return "tool" }
func (m *ToolMessage) MsgID() string   { return m.id }
func (m *ToolMessage) Content() string { return m.result }

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

func (m *ToolMessage) renderLocked(width int) string {
	firstLine := m.formatFirstLine()

	var body string
	if m.state != ToolStateExecuting && m.result != "" {
		lines := strings.Split(m.result, "\n")
		if len(lines) > maxToolResultLines {
			skipped := len(lines) - maxToolResultLines
			lines = append(lines[:maxToolResultLines],
				fmt.Sprintf("...%d more lines...", skipped))
		}
		body = "\n" + strings.Join(lines, "\n")
	}

	style := toolStyle
	boldStyle := toolBoldStyle
	if m.state == ToolStateError {
		style = toolErrorStyle
		boldStyle = toolErrorBoldStyle
	}

	rendered := strings.TrimSpace(boldStyle.Width(width - 1).Render(firstLine))
	if body != "" {
		rendered += "\n" + strings.TrimSpace(style.Width(width - 1).Render(strings.TrimSpace(body)))
	}
	return rendered
}

func (m *ToolMessage) formatFirstLine() string {
	switch m.name {
	case "edit":
		path := parseArgPath(m.args)
		return "Edit " + path
	case "read":
		path := parseArgPath(m.args)
		return "Read " + path
	case "write":
		path := parseArgPath(m.args)
		return "Write " + path
	case "bash", "powershell":
		cmd := parseArgCommand(m.args)
		return cmd
	default:
		return m.name + "(" + truncateStr(m.args, 150) + ")"
	}
}

func parseArgPath(argsJSON string) string {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.Path == "" {
		return "(unknown path)"
	}
	return truncateStr(args.Path, 200)
}

func parseArgCommand(argsJSON string) string {
	var args struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args.Command == "" {
		return "(unknown command)"
	}
	return truncateStr(args.Command, 200)
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
