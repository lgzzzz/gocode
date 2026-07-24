package compoent

type SystemMessage struct {
	content     string
	renderCache string
	renderWidth int
	dirty       bool
}

func NewSystemMessage(content string) *SystemMessage {
	m := &SystemMessage{}
	m.SetContent(content)
	return m
}

func (m *SystemMessage) Type() string    { return "system" }
func (m *SystemMessage) MsgID() string   { return "" }
func (m *SystemMessage) Content() string { return m.content }

func (m *SystemMessage) SetContent(content string) {
	if m.content == content {
		return
	}
	m.content = content
	if m.renderWidth > 0 {
		m.renderCache = renderTrim(systemStyle, m.renderWidth-1, content)
	} else {
		m.dirty = true
	}
}

func (m *SystemMessage) Render(width int) string {
	if !m.dirty && width == m.renderWidth {
		return m.renderCache
	}
	m.renderWidth = width
	m.renderCache = renderTrim(systemStyle, width-1, m.content)
	m.dirty = false
	return m.renderCache
}
