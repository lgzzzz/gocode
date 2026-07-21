package compoent

// ErrorMessage renders an error message.
type ErrorMessage struct {
	Content string
}

func (m ErrorMessage) Type() string  { return "error" }
func (m ErrorMessage) MsgID() string { return "" }

func (m ErrorMessage) Render(width int) string {
	return renderTrim(errorStyle, width-1, m.Content)
}
