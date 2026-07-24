package compoent


type Component interface {
	Type() string
	MsgID() string
	Content() string
	Render(width int) string
	SetContent(content string)
}
