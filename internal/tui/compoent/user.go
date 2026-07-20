package compoent

import "strings"

// UserMessage renders a user chat message.
type UserMessage struct {
	Content string
}

func (m UserMessage) Type() string  { return "user" }
func (m UserMessage) MsgID() string { return "" }

func (m UserMessage) Render(width int) string {
	if strings.TrimSpace(m.Content) == "" {
		return ""
	}
	return userStyle.Width(width - 1).Render(m.Content)
}
