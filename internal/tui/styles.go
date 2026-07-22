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

	// commandPaletteStyle — style for the command palette popup.
	commandPaletteStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("8")).
				Padding(0, 1)

	// commandHighlightStyle — style for the selected command in the palette.
	commandHighlightStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("4")). // blue background
				Foreground(lipgloss.Color("15")) // white text

	// commandDimStyle — style for non-selected commands.
	commandDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))
)
