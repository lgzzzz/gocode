package tui

import (
	"charm.land/lipgloss/v2"
)

// ---- styles (input box only; message styles live in compoent package) ----

var (
	// inputBarStyle — cyan left bar for the input area.
	inputBarStyle = lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("14")).
			PaddingLeft(1).
			Foreground(lipgloss.Color("15"))

	// inputBarDimStyle — gray left bar for disabled / processing state.
	inputBarDimStyle = lipgloss.NewStyle().
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color("8")).
				PaddingLeft(1).
				Foreground(lipgloss.Color("8"))

	// moreLinesStyle — subtle indicator for hidden lines above the input area.
	moreLinesStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Italic(true)
)
