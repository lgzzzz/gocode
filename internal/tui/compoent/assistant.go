package compoent

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/tui/markdown"
)

// shared markdown config (can be customized later)
var mdConfig = markdown.DefaultStyles()

// Bar prefix styles — we render these once and prepend to every line,
// which avoids lipgloss BorderLeft edge cases with ANSI-markup content.
var (
	assistantBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Render("┃ ")

	thinkingBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("13")).
			Render("┃ ")
)

type AssistantMessage struct {
	id          string
	content     string
	renderCache string
	renderWidth int
	dirty       bool
}

func NewAssistantMessage(id, content string) *AssistantMessage {
	m := &AssistantMessage{id: id}
	m.SetContent(content)
	return m
}

func (m *AssistantMessage) Type() string    { return string(agent.MsgAssistant) }
func (m *AssistantMessage) MsgID() string   { return m.id }
func (m *AssistantMessage) Content() string { return m.content }

func (m *AssistantMessage) SetContent(content string) {
	if m.content == content {
		return
	}
	m.content = content
	m.dirty = true
}

func (m *AssistantMessage) Render(width int) string {
	if !m.dirty && width == m.renderWidth {
		return m.renderCache
	}
	m.renderWidth = width
	m.renderCache = m.renderMarkdown(width)
	m.dirty = false
	return m.renderCache
}

// renderMarkdown converts the message content from markdown to styled
// terminal output using glamour, then prepends the assistant bar prefix
// to each line.
func (m *AssistantMessage) renderMarkdown(width int) string {
	if strings.TrimSpace(m.content) == "" {
		return ""
	}

	// glamour handles word wrap; subtract 2 for the bar prefix "▌ ".
	renderWidth := width - 2

	renderer := markdown.MarkdownRenderer(mdConfig, renderWidth)
	out, err := renderer.Render(m.content)
	if err != nil {
		return renderTrimWithPrefix(assistantBar, width, m.content)
	}

	out = strings.TrimSuffix(out, "\n")
	return prefixLines(assistantBar, out)
}

// prefixLines prepends the given prefix to every line in s.
func prefixLines(prefix, s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

// renderTrimWithPrefix is a fallback that renders plain text with the bar prefix.
func renderTrimWithPrefix(prefix string, width int, content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	// For plain-text fallback, wrap to fit width-2 (bar takes 2 chars).
	contentWidth := width - 2
	if contentWidth < 40 {
		contentWidth = 40
	}
	wrapped := lipgloss.NewStyle().
		Width(contentWidth).
		Foreground(lipgloss.Color("15")).
		Render(content)
	return prefixLines(prefix, strings.TrimSpace(wrapped))
}
