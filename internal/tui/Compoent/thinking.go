package compoent

import "strings"

// ThinkingMessage renders an assistant's reasoning/thinking block.
type ThinkingMessage struct {
	ID      string
	Content string
}

func (m ThinkingMessage) Type() string  { return "thinking" }
func (m ThinkingMessage) MsgID() string { return m.ID }

func (m ThinkingMessage) Render(width int) string {
	if strings.TrimSpace(m.Content) == "" {
		return ""
	}
	return thinkingStyle.Width(width - 1).Render(m.Content)
}
