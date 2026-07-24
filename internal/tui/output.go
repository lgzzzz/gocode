package tui

import tea "charm.land/bubbletea/v2"

func (m *model) updateOutput(msg tea.Msg) []tea.Cmd {
	newOutput, cmd := m.output.Update(msg)
	m.output = newOutput
	if cmd != nil {
		return []tea.Cmd{cmd}
	}
	return nil
}

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
