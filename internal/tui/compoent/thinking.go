package compoent

import "github.com/lgzzzz/gocode/internal/agent"

type ThinkingMessage struct {
	id          string
	content     string
	renderCache string
	renderWidth int
	dirty       bool
}

func NewThinkingMessage(id, content string) *ThinkingMessage {
	m := &ThinkingMessage{id: id}
	m.SetContent(content)
	return m
}

func (m *ThinkingMessage) Type() string    { return string(agent.MsgThinking) }
func (m *ThinkingMessage) MsgID() string   { return m.id }
func (m *ThinkingMessage) Content() string { return m.content }

func (m *ThinkingMessage) SetContent(content string) {
	if m.content == content {
		return
	}
	m.content = content
	if m.renderWidth > 0 {
		m.renderCache = renderTrim(thinkingStyle, m.renderWidth-1, content)
	} else {
		m.dirty = true
	}
}

func (m *ThinkingMessage) Render(width int) string {
	if !m.dirty && width == m.renderWidth {
		return m.renderCache
	}
	m.renderWidth = width
	m.renderCache = renderTrim(thinkingStyle, width-1, m.content)
	m.dirty = false
	return m.renderCache
}
