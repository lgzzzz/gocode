package tui

import (
	"charm.land/lipgloss/v2"
)


var (
	inputBarDimStyle = lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("7")).
		PaddingLeft(1).
		Foreground(lipgloss.Color("8"))
)
