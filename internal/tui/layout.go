package tui

import (
	tea "charm.land/bubbletea/v2"
)

// ---- layout helpers ----

// adjustLayout recalculates editor and output dimensions based on the
// current terminal size, command palette visibility, and editor height.
func (m *model) adjustLayout() {
	m.editor.SetWidth(m.width - 2)
	m.output.SetWidth(m.width - 2)

	paletteHeight := m.palette.Height()
	editorHeight := m.editor.Height()
	totalBottom := editorHeight + paletteHeight + 1 // +1 for spacing
	m.output.SetHeight(max(0, m.height-totalBottom))
}

// handleWindowSizeMsg updates dimensions on terminal resize.
func (m *model) handleWindowSizeMsg(msg tea.WindowSizeMsg) []tea.Cmd {
	m.width = msg.Width
	m.height = msg.Height
	m.history.MarkDirty() // width changed, need re-render
	// WindowSizeMsg still needs to reach editor and output so they
	// can adjust their own internal sizes.
	return append(m.updateEditor(msg), m.updateOutput(msg)...)
}
