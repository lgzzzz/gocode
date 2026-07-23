package compoent

import "github.com/lgzzzz/gocode/internal/agent"

// AssistantMessage renders an assistant chat message.
type AssistantMessage struct {
	id          string
	content     string
	renderCache string
	renderWidth int
	dirty       bool
}

// NewAssistantMessage creates a new assistant message component.
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
	if m.renderWidth > 0 {
		m.renderCache = renderTrim(assistantStyle, m.renderWidth-1, content)
	} else {
		m.dirty = true
	}
}

func (m *AssistantMessage) Render(width int) string {
	if !m.dirty && width == m.renderWidth {
		return m.renderCache
	}
	m.renderWidth = width
	m.renderCache = renderTrim(assistantStyle, width-1, m.content)
	m.dirty = false
	return m.renderCache
}
