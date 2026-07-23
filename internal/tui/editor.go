package tui

import tea "charm.land/bubbletea/v2"

// updateEditor forwards a message to the editor textarea and returns any command.
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
