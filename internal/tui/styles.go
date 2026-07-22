package tui

import (
	"charm.land/lipgloss/v2"
)

// ---- styles (input box only; message styles live in compoent package) ----

var (
	// inputBarDimStyle — gray left bar for disabled / processing state.
	inputBarDimStyle = lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("8")).
		PaddingLeft(1).
		Foreground(lipgloss.Color("8"))
)
