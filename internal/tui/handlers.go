package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)

// ---- message handlers ----

// handleKeyPress processes keyboard events.
func (m *model) handleKeyPress(msg tea.KeyPressMsg) []tea.Cmd {
	var cmds []tea.Cmd

	k := msg.Key()

	// Always forward to input (except for special keys that we handle first).
	switch {
	case k.Code == tea.KeyUp || k.Code == tea.KeyDown:
		cmds = append(cmds, m.updateInput(msg)...)
	default:
		cmds = append(cmds, m.updateInput(msg)...)
	}

	// Special key bindings (quit, submit, etc.)
	switch msg.String() {
	case "ctrl+c":
		cmds = append(cmds, tea.Quit)
		return cmds

	case "esc":
		if m.running {
			m.cancelAgent()
		}
		return cmds

	case "enter":
		if !m.running {
			cmd := m.submitTask()
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return cmds
}

// handleWindowSizeMsg updates dimensions on terminal resize.
func (m *model) handleWindowSizeMsg(msg tea.WindowSizeMsg) []tea.Cmd {
	m.width = msg.Width
	m.height = msg.Height
	m.dirty = true // width changed, need re-render
	// WindowSizeMsg still needs to reach input and viewport so they
	// can adjust their own internal sizes.
	return append(m.updateInput(msg), m.updateViewportModel(msg)...)
}

// handleProgressMsg processes agent callback messages (streaming, tool calls, etc.).
func (m *model) handleProgressMsg(msg progressMsg) []tea.Cmd {
	if msg.err != nil {
		m.log = append(m.log,
			compoent.ErrorMessage{Content: msg.err.Error()},
		)
		m.dirty = true
		return nil
	}

	if msg.done {
		m.running = false
		m.cancel = nil
		m.ch = nil
		return nil
	}

	switch msg.typ {
	case agent.MsgAssistantStream, agent.MsgThinkingStream:
		m.applyStreamUpdate(msg)

	case agent.MsgToolCall:
		m.log = append(m.log, compoent.NewToolMessage(msg.id, msg.toolName, msg.toolArgs))
		m.dirty = true

	case agent.MsgToolResult:
		m.applyToolResult(msg)
		m.dirty = true

	default:
		m.log = append(m.log, &compoent.AssistantMessage{ID: msg.id, Content: msg.content})
		m.dirty = true
	}

	if m.ch != nil {
		return []tea.Cmd{waitCmd(m.ch)}
	}
	return nil
}
