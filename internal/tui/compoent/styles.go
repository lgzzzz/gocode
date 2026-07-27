package compoent

import (
	"charm.land/lipgloss/v2"
	"github.com/lgzzzz/gocode/internal/tui/util"
)

func Render(style lipgloss.Style, width int, content string) string {
	out := style.Width(width).Render(content)
	return util.TrimEmptyLine(out)
}

func leftBar(hexColor string) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color(hexColor)).
		PaddingLeft(1)
}

var (
	UserStyle = leftBar("15").
			Foreground(lipgloss.Color("15"))

	AssistantStyle = leftBar("12")

	ThinkingStyle = leftBar("13").
			Italic(true)

	ToolStyle = leftBar("10").
			Foreground(lipgloss.Color("10"))

	ToolBoldStyle = leftBar("10").
			Foreground(lipgloss.Color("10")).
			Bold(true)

	ToolErrorStyle = leftBar("1").
			Foreground(lipgloss.Color("1"))

	ToolErrorBoldStyle = leftBar("1").
				Foreground(lipgloss.Color("1")).
				Bold(true)

	ErrorStyle = leftBar("1").
			Foreground(lipgloss.Color("1")).
			Bold(true)

	SystemStyle = leftBar("11").
			Foreground(lipgloss.Color("11"))
)
