package compoent

import "strings"

// SystemMessage renders a system/welcome banner.
type SystemMessage struct {
	Content string
}

func (m SystemMessage) Type() string  { return "system" }
func (m SystemMessage) MsgID() string { return "" }

func (m SystemMessage) Render(width int) string {
	if strings.TrimSpace(m.Content) == "" {
		return ""
	}
	return systemStyle.Width(width - 1).Render(m.Content)
}
