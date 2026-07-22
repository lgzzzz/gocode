package tui

import (
	tea "charm.land/bubbletea/v2"
)

// ---- layout helpers ----

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

// updateOutput forwards a message to the output viewport and returns any command.
func (m *model) updateOutput(msg tea.Msg) []tea.Cmd {
	newOutput, cmd := m.output.Update(msg)
	m.output = newOutput
	if cmd != nil {
		return []tea.Cmd{cmd}
	}
	return nil
}

// renderOutput re-renders the history into the output viewport when dirty.
func (m *model) renderOutput() {
	parts, ok := m.history.Render(m.output.Width())
	if !ok {
		return
	}

	atBottom := m.output.AtBottom()
	m.output.SetContentLines(parts)
	if atBottom {
		m.output.GotoBottom()
	}
}
