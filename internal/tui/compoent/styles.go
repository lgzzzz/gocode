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

// prefixLines prepends the given prefix to every line in s.
func prefixLines(prefix, s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

// renderTrimWithPrefix is a fallback that renders plain text with the bar prefix.
func renderTrimWithPrefix(prefix string, width int, content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	// For plain-text fallback, wrap to fit width-2 (bar takes 2 chars).
	contentWidth := width - 2
	if contentWidth < 40 {
		contentWidth = 40
	}
	wrapped := lipgloss.NewStyle().
		Width(contentWidth).
		Foreground(lipgloss.Color("15")).
		Render(content)
	return prefixLines(prefix, strings.TrimSpace(wrapped))
}

var (
	assistantBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Render("┃ ")

	thinkingBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("13")).
			Render("┃ ")
)

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
