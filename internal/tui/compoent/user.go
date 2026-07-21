package compoent

// UserMessage renders a user chat message.
type UserMessage struct {
	Content string
}

func (m UserMessage) Type() string  { return "user" }
func (m UserMessage) MsgID() string { return "" }

func (m UserMessage) Render(width int) string {
	return renderTrim(userStyle, width-1, m.Content)
}
