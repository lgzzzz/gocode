package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/store"
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)

// ---- message handlers ----

// handleKeyPress processes keyboard events.
func (m *model) handleKeyPress(msg tea.KeyPressMsg) []tea.Cmd {
	var cmds []tea.Cmd

	k := msg.Key()

	// ---- command mode: intercept special keys ----
	if m.palette.Active() {
		result := m.palette.HandleKey(msg.String())

		if result.Dismiss {
			m.editor.Reset()
			return nil
		}

		if result.Execute != nil {
			args := m.palette.Args(result.Execute.Name())
			m.palette.Dismiss()
			m.editor.Reset()
			return []tea.Cmd{m.executeCommand(result.Execute, args)}
		}

		if result.CompleteText != "" {
			m.editor.SetValue(result.CompleteText)
			m.editor.CursorEnd()
			m.palette.UpdateFilter(m.editor.Value())
			return nil
		}

		// For up/down (already handled inside HandleKey) and unrecognized
		// keys, don't forward to the editor — navigation keys are consumed.
		if msg.String() == "up" || msg.String() == "down" {
			return nil
		}

		// For other keys (letters, backspace, etc.), fall through and let the
		// editor process them normally.
	}

	// Always forward to editor (except for special keys that we handle first).
	switch {
	case k.Code == tea.KeyUp || k.Code == tea.KeyDown:
		cmds = append(cmds, m.updateEditor(msg)...)
	default:
		cmds = append(cmds, m.updateEditor(msg)...)
	}

	// After editor update, refresh command palette state
	m.palette.UpdateFilter(m.editor.Value())

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
			input := strings.TrimSpace(m.editor.Value())
			if input == "" {
				return cmds
			}
			m.editor.Reset()
			m.history.Append(compoent.NewUserMessage(input))

			// Persist user message.
			if m.store != nil {
				m.store.AppendMessage(store.Message{
					SessionID: m.sessionID,
					Role:      "user",
					MsgType:   "user_message",
					Content:   input,
				})
			}

			cmd := m.startAgent(input)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			m.output.GotoBottom()
		}
	}

	return cmds
}

// handleProgressMsg processes agent callback messages (streaming, tool calls, etc.).
func (m *model) handleProgressMsg(msg progressMsg) []tea.Cmd {
	if msg.err != nil {
		m.history.Append(compoent.NewErrorMessage(msg.err.Error()))
		return nil
	}

	if msg.done {
		m.running = false
		m.cancel = nil
		m.ch = nil
		return nil
	}

	// Persist complete messages (streaming messages are skipped inside).
	m.persistMessage(msg)

	switch msg.typ {
	case agent.MsgAssistantStream, agent.MsgThinkingStream:
		m.applyStreamUpdate(msg)

	case agent.MsgToolResult:
		m.applyToolResult(msg)

	case agent.MsgToolCall:
		m.history.Append(compoent.NewToolMessage(msg.id, msg.toolName, msg.toolArgs))

	case agent.MsgError, agent.MsgRetryWait:
		// Show error and retry messages in the history
		m.history.Append(compoent.NewErrorMessage(msg.content))

	default:
		m.history.Append(compoent.NewAssistantMessage(msg.id, msg.content))
	}

	if m.ch != nil {
		return []tea.Cmd{waitCmd(m.ch)}
	}
	return nil
}

// persistMessage persists only "complete" message types to the store.
// Streaming messages (thinking_stream, assistant_stream) are skipped.
func (m *model) persistMessage(msg progressMsg) {
	if m.store == nil {
		return
	}

	sm := store.Message{
		SessionID: m.sessionID,
		ToolName:  msg.toolName,
		ToolArgs:  msg.toolArgs,
	}

	switch msg.typ {

	// ---- streaming messages: not persisted ----
	case agent.MsgThinkingStream, agent.MsgAssistantStream:
		return

	// ---- complete thinking ----
	case agent.MsgThinking:
		sm.Role = "assistant"
		sm.MsgType = "thinking"
		sm.Content = msg.content

	// ---- complete assistant reply ----
	case agent.MsgAssistant:
		sm.Role = "assistant"
		sm.MsgType = "assistant"
		sm.Content = msg.content

	// ---- tool call ----
	case agent.MsgToolCall:
		sm.Role = "assistant"
		sm.MsgType = "tool_call"
		sm.ToolCallID = msg.id
		sm.ToolName = msg.toolName
		sm.ToolArgs = msg.toolArgs
		sm.Content = msg.content

	// ---- tool result ----
	case agent.MsgToolResult:
		sm.Role = "tool"
		sm.MsgType = "tool_result"
		sm.ToolCallID = msg.id
		sm.ToolName = msg.toolName
		sm.Content = msg.content
		sm.HasError = msg.toolErr != nil

	// ---- error ----
	case agent.MsgError:
		sm.Role = "system"
		sm.MsgType = "error"
		sm.Content = msg.content
		sm.HasError = true

	// ---- retry wait ----
	case agent.MsgRetryWait:
		sm.Role = "system"
		sm.MsgType = "retry_wait"
		sm.Content = msg.content

	default:
		return
	}

	m.store.AppendMessage(sm)
}
