package compoent

import (
	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/tui/markdown"
)

type AssistantMessage struct {
	id          string
	content     string
	renderCache string
	renderWidth int
	dirty       bool
	md          *markdown.Renderer
}

func NewAssistantMessage(id, content string) *AssistantMessage {
	m := &AssistantMessage{id: id, md: markdown.NewRenderer()}
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

func (m *AssistantMessage) renderMarkdown(width int) string {
	out := m.md.Render(m.content, width-2)
	return Render(AssistantStyle, width, out)
}
