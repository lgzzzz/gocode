package compoent

import "strings"

// ErrorMessage renders an error message.
type ErrorMessage struct {
	Content string
}

func (m ErrorMessage) Type() string  { return "error" }
func (m ErrorMessage) MsgID() string { return "" }

func (m ErrorMessage) Render(width int) string {
	if strings.TrimSpace(m.Content) == "" {
		return ""
	}
	return errorStyle.Width(width - 1).Render(m.Content)
}
