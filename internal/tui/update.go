package tui

import (
	tea "charm.land/bubbletea/v2"
)

// ---- layout helpers ----

// updateInput forwards a message to the input textarea and returns any command.
func (m *model) updateInput(msg tea.Msg) []tea.Cmd {
	if m.running {
		return nil
	}
	newInput, cmd := m.input.Update(msg)
	m.input = newInput
	if cmd != nil {
		return []tea.Cmd{cmd}
	}
	return nil
}

// updateViewportModel forwards a message to the viewport and returns any command.
func (m *model) updateViewportModel(msg tea.Msg) []tea.Cmd {
	newVP, cmd := m.viewport.Update(msg)
	m.viewport = newVP
	if cmd != nil {
		return []tea.Cmd{cmd}
	}
	return nil
}

func (m *model) updateViewport() {
	// Skip expensive re-render if nothing changed (e.g. during rapid typing/deletion).
	if !m.dirty {
		return
	}
	m.dirty = false

	atBottom := m.viewport.AtBottom()
	var parts []string
	for i, comp := range m.log {
		rendered := comp.Render(m.viewport.Width())
		if rendered != "" {
			parts = append(parts, rendered)
			if i != len(m.log)-1 {
				parts = append(parts, "") // spacing between cards
			}
		}
	}
	m.viewport.SetContentLines(parts)
	if atBottom {
		m.viewport.GotoBottom()
	}
}
