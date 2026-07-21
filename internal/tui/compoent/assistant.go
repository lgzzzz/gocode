package compoent

// AssistantMessage renders an assistant chat message.
type AssistantMessage struct {
	ID      string
	Content string
}

func (m AssistantMessage) Type() string  { return "assistant" }
func (m AssistantMessage) MsgID() string { return m.ID }

func (m AssistantMessage) Render(width int) string {
	return renderTrim(assistantStyle, width-1, m.Content)
}
