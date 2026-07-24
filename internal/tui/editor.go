package tui

import tea "charm.land/bubbletea/v2"

func (m *model) updateEditor(msg tea.Msg) []tea.Cmd {
	if m.running {
		return nil
	}
	newEditor, cmd := m.editor.Update(msg)
	m.editor = newEditor
	if cmd != nil {
		return []tea.Cmd{cmd}
	}
	return nil
}
