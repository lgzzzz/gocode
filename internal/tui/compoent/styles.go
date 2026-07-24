package compoent

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func renderTrim(style lipgloss.Style, width int, content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	return strings.TrimSpace(style.Width(width).Render(content))
}


func leftBar(hexColor string) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color(hexColor)).
		PaddingLeft(1)
}

var (
	userStyle = leftBar("15").
			Foreground(lipgloss.Color("15"))

	assistantStyle = leftBar("12").
			Foreground(lipgloss.Color("15"))

	thinkingStyle = leftBar("13").
			Foreground(lipgloss.Color("13")).
			Italic(true)

	toolStyle = leftBar("10").
			Foreground(lipgloss.Color("10"))

	toolBoldStyle = leftBar("10").
			Foreground(lipgloss.Color("10")).
			Bold(true)

	toolErrorStyle = leftBar("1").
			Foreground(lipgloss.Color("1"))

	toolErrorBoldStyle = leftBar("1").
			Foreground(lipgloss.Color("1")).
			Bold(true)

	errorStyle = leftBar("1").
			Foreground(lipgloss.Color("1")).
			Bold(true)

	systemStyle = leftBar("11").
			Foreground(lipgloss.Color("11"))
)
