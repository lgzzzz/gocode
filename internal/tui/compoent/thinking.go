package compoent

import (
	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/tui/markdown"
)

type ThinkingMessage struct {
	id          string
	content     string
	renderCache string
	renderWidth int
	dirty       bool
	md          *markdown.Renderer
}

func NewThinkingMessage(id, content string) *ThinkingMessage {
	m := &ThinkingMessage{id: id, md: markdown.NewRenderer(ThinkingStyle)}
	m.SetContent(content)
	return m
}

func (m *ThinkingMessage) SetFullRender(fullRender bool) {
	m.md.SetFullRender(fullRender)
}

func (m *ThinkingMessage) SetFullStyleRender(fullRender bool) {
	m.md.SetFullStyleRender(fullRender)
}

func (m *ThinkingMessage) Type() string    { return string(agent.MsgThinking) }
func (m *ThinkingMessage) MsgID() string   { return m.id }
func (m *ThinkingMessage) Content() string { return m.content }

func (m *ThinkingMessage) SetContent(content string) {
	if m.content == content {
		return
	}
	m.content = content
	m.dirty = true
}

func (m *ThinkingMessage) Render(width int) string {
	if !m.dirty && width == m.renderWidth {
		return m.renderCache
	}
	m.renderWidth = width
	m.renderCache = m.renderMarkdown(width)
	m.dirty = false
	return m.renderCache
}

func (m *ThinkingMessage) renderMarkdown(width int) string {
	if m.content == "" {
		return ""
	}
	return m.md.Render(m.content, width)
}
