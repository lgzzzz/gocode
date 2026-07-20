package compoent

import "github.com/charmbracelet/lipgloss"

// ---- Modern Left Accent Bar Styles ----
//
// Design: each message type gets ONLY a left-side border using the
// Unicode half-block character "▌" for a smooth, filled accent bar.
// Different colors distinguish message sources at a glance.
//
// lipgloss capabilities used:
//   - BorderLeft(true)            → only render left edge
//   - BorderStyle(OuterHalfBlock) → left character = "▌" (filled half-width bar)
//   - BorderForeground(color)     → tint the bar (only left edge drawn, so it's
//                                    effectively BorderLeftForeground)
//   - PaddingLeft(1)              → gap between bar and text

// leftBar returns a base style with a colored left accent bar.
func leftBar(hexColor string) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.OuterHalfBlockBorder()). // left = "▌"
		BorderForeground(lipgloss.Color(hexColor)).
		PaddingLeft(1)
}

var (
	// userStyle — white "▌" bar, pushed slightly right via left margin
	userStyle = leftBar("15"). // white
			Foreground(lipgloss.Color("15")).
			MarginLeft(4)

	// assistantStyle — blue "▌" bar
	assistantStyle = leftBar("12"). // bright blue
			Foreground(lipgloss.Color("15"))

	// thinkingStyle — purple "▌" bar + italic
	thinkingStyle = leftBar("13"). // purple
			Foreground(lipgloss.Color("13")).
			Italic(true)

	// toolStyle — green "▌" bar
	toolStyle = leftBar("10"). // green
			Foreground(lipgloss.Color("10"))

	// errorStyle — red "▌" bar + bold
	errorStyle = leftBar("9"). // red
			Foreground(lipgloss.Color("9")).
			Bold(true)

	// systemStyle — yellow "▌" bar
	systemStyle = leftBar("11"). // yellow
			Foreground(lipgloss.Color("11"))
)
