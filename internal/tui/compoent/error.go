package compoent

// ErrorMessage renders an error message.
type ErrorMessage struct {
	content     string
	renderCache string
	renderWidth int
	dirty       bool
}

// NewErrorMessage creates a new error message component.
func NewErrorMessage(content string) *ErrorMessage {
	m := &ErrorMessage{}
	m.SetContent(content)
	return m
}

func (m *ErrorMessage) Type() string  { return "error" }
func (m *ErrorMessage) MsgID() string { return "" }

func (m *ErrorMessage) SetContent(content string) {
	if m.content == content {
		return
	}
	m.content = content
	if m.renderWidth > 0 {
		m.renderCache = renderTrim(errorStyle, m.renderWidth-1, content)
	} else {
		m.dirty = true
	}
}

func (m *ErrorMessage) Render(width int) string {
	if !m.dirty && width == m.renderWidth {
		return m.renderCache
	}
	m.renderWidth = width
	m.renderCache = renderTrim(errorStyle, width-1, m.content)
	m.dirty = false
	return m.renderCache
}
