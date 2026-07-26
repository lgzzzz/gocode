package compoent

import (
	"strings"

	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/tui/markdown"
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
	content := strings.TrimSpace(m.content)
	if content == "" {
		return ""
	}

	renderer := markdown.MarkdownRenderer(width)
	defer renderer.Close()
	out, err := renderer.Render(content)
	if err != nil {
		return renderTrim(assistantStyle, width, content)
	}

	return renderTrim(assistantStyle, width, out)
}
