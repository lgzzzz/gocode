package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/store"
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)

// ---- agent actions ----

// startAgent launches the agent with the given input string.
// The caller is responsible for input validation, UI cleanup
// (editor reset, history append), and persistence.
func (m *model) startAgent(input string) tea.Cmd {
	m.running = true

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	ch := make(chan progressMsg, 64)
	m.ch = ch

	go func(ag *agent.Agent, input string) {
		defer func() {
			if r := recover(); r != nil {
				ch <- progressMsg{err: fmt.Errorf("panic: %v", r)}
				ch <- progressMsg{done: true}
				close(ch)
			}
		}()
		ag.Run(ctx, input, func(msg agent.CallbackMsg) {
			ch <- progressMsg{
				typ:      msg.Type,
				id:       msg.ID,
				content:  msg.Content,
				toolName: msg.ToolName,
				toolArgs: msg.ToolArgs,
				toolErr:  msg.Err,
			}
		})
		ch <- progressMsg{done: true}
		close(ch)
	}(m.agent, input)

	return waitCmd(ch)
}

// ---- ModelAccess interface implementation ----

// Running returns whether the agent is currently executing.
func (m *model) Running() bool { return m.running }

// CancelAgent cancels the running agent context.
func (m *model) CancelAgent() { m.cancelAgent() }

// cancelAgent cancels the running agent context, stopping the ReAct loop.
func (m *model) cancelAgent() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	// The goroutine will receive context.Canceled, send the error + done
	// messages, and handleProgressMsg will transition out of running state.
}

// ---- progress handling ----

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
		SessionID:  m.sessionID,
		ToolCallID: msg.id,
		ToolName:   msg.toolName,
		ToolArgs:   msg.toolArgs,
	}

	switch msg.typ {
	// ---- complete thinking ----
	case agent.MsgThinking:
		sm.MsgType = string(agent.MsgThinking)
		sm.Content = msg.content

	// ---- complete assistant reply ----
	case agent.MsgAssistant:
		sm.MsgType = string(agent.MsgAssistant)
		sm.Content = msg.content

	// ---- tool call ----
	case agent.MsgToolCall:
		sm.MsgType = string(agent.MsgToolCall)
		sm.Content = msg.content

	// ---- tool result ----
	case agent.MsgToolResult:
		sm.MsgType = string(agent.MsgToolResult)
		sm.Content = msg.content
		sm.HasError = msg.toolErr != nil
	default:
		return
	}

	m.store.AppendMessage(sm)
}
