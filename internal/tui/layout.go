package tui

import (
	"strings"

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

func (m *model) adjustLayout() {
	m.input.SetWidth(m.width - 2)
	m.viewport.SetWidth(m.width - 2)
	m.viewport.SetHeight(max(0, m.height-m.input.Height()-1))
}

func (m *model) updateViewport() {
	// Skip expensive re-render if nothing changed (e.g. during rapid typing/deletion).
	if !m.dirty {
		return
	}
	m.dirty = false

	atBottom := m.viewport.AtBottom()
	var parts []string
	for _, comp := range m.log {
		rendered := comp.Render(m.viewport.Width())
		if rendered != "" {
			parts = append(parts, rendered)
			parts = append(parts, "") // spacing between cards
		}
	}
	content := strings.TrimSpace(strings.Join(parts, "\n"))
	m.viewport.SetContent(content)

	if content != m.lastContent {
		if atBottom {
			m.viewport.GotoBottom()
		}
		m.lastContent = content
	}
}
