package compoent

// UserMessage renders a user chat message.
type UserMessage struct {
	content     string
	renderCache string
	renderWidth int
	dirty       bool
}

// NewUserMessage creates a new user message component.
func NewUserMessage(content string) *UserMessage {
	m := &UserMessage{}
	m.SetContent(content)
	return m
}

func (m *UserMessage) Type() string  { return "user" }
func (m *UserMessage) MsgID() string { return "" }

func (m *UserMessage) SetContent(content string) {
	if m.content == content {
		return
	}
	m.content = content
	if m.renderWidth > 0 {
		m.renderCache = renderTrim(userStyle, m.renderWidth-1, content)
	} else {
		m.dirty = true
	}
}

func (m *UserMessage) Render(width int) string {
	if !m.dirty && width == m.renderWidth {
		return m.renderCache
	}
	m.renderWidth = width
	m.renderCache = renderTrim(userStyle, width-1, m.content)
	m.dirty = false
	return m.renderCache
}
