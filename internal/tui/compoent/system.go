package compoent

// SystemMessage renders a system/welcome banner.
type SystemMessage struct {
	Content string
}

func (m SystemMessage) Type() string  { return "system" }
func (m SystemMessage) MsgID() string { return "" }

func (m SystemMessage) Render(width int) string {
	return renderTrim(systemStyle, width-1, m.Content)
}
