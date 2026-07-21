package compoent

// ThinkingMessage renders an assistant's reasoning/thinking block.
type ThinkingMessage struct {
	ID      string
	Content string
}

func (m ThinkingMessage) Type() string  { return "thinking" }
func (m ThinkingMessage) MsgID() string { return m.ID }

func (m ThinkingMessage) Render(width int) string {
	return renderTrim(thinkingStyle, width-1, m.Content)
}
