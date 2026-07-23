package compoent

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// renderTrim renders content with the given style and trims trailing blank lines.
func renderTrim(style lipgloss.Style, width int, content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	return strings.TrimSpace(style.Width(width).Render(content))
}

// ---- Modern Left Accent Bar Styles ----
//
// Design: each message type gets ONLY a left-side border using the
// Unicode heavy vertical bar "┃" (U+2503) for a solid, connected accent bar
// that renders consistently across all terminals.
// Different colors distinguish message sources at a glance.
//
// lipgloss capabilities used:
//   - BorderLeft(true)         → only render left edge
//   - BorderStyle(ThickBorder) → left character = "┃" (full-cell heavy bar)
//   - BorderForeground(color)  → tint the bar
//   - PaddingLeft(1)           → gap between bar and text

// leftBar returns a base style with a colored left accent bar.
func leftBar(hexColor string) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()). // left = "┃"
		BorderForeground(lipgloss.Color(hexColor)).
		PaddingLeft(1)
}

var (
	// userStyle — white "┃" bar
	userStyle = leftBar("15"). // white
			Foreground(lipgloss.Color("15"))

	// assistantStyle — blue "┃" bar
	assistantStyle = leftBar("12"). // bright blue
			Foreground(lipgloss.Color("15"))

	// thinkingStyle — purple "┃" bar + italic
	thinkingStyle = leftBar("13"). // purple
			Foreground(lipgloss.Color("13")).
			Italic(true)

	// toolStyle — green "┃" bar
	toolStyle = leftBar("10"). // green
			Foreground(lipgloss.Color("10"))

	// toolBoldStyle — green "┃" bar + bold (first line)
	toolBoldStyle = leftBar("10"). // green
			Foreground(lipgloss.Color("10")).
			Bold(true)

	// toolErrorStyle — dark red "┃" bar for tool execution errors
	toolErrorStyle = leftBar("1"). // dark red
			Foreground(lipgloss.Color("1"))

	// toolErrorBoldStyle — dark red "┃" bar + bold (first line)
	toolErrorBoldStyle = leftBar("1"). // dark red
			Foreground(lipgloss.Color("1")).
			Bold(true)

	// errorStyle — dark red "┃" bar + bold
	errorStyle = leftBar("1"). // dark red
			Foreground(lipgloss.Color("1")).
			Bold(true)

	// systemStyle — yellow "┃" bar
	systemStyle = leftBar("11"). // yellow
			Foreground(lipgloss.Color("11"))
)
