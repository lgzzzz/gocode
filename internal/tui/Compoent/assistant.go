package compoent

import "strings"

// AssistantMessage renders an assistant chat message.
type AssistantMessage struct {
	ID      string
	Content string
}

func (m AssistantMessage) Type() string  { return "assistant" }
func (m AssistantMessage) MsgID() string { return m.ID }

func (m AssistantMessage) Render(width int) string {
	if strings.TrimSpace(m.Content) == "" {
		return ""
	}
	return assistantStyle.Width(width - 1).Render(m.Content)
}
