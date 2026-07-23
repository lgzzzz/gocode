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

// StartAgent launches the agent with the given input string.
func (m *model) StartAgent(input string) tea.Cmd {
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
				msg: msg,
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

func (m *model) cancelAgent() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

// ---- progress handling ----

// handleProgressMsg processes agent callback messages.
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
	agentMsg := msg.msg
	m.persistMessage(agentMsg)
	switch agentMsg.Type {
	case agent.MsgAssistantStream:
		m.history.Upsert(compoent.NewAssistantMessage(agentMsg.ID, agentMsg.Content))
	case agent.MsgThinkingStream:
		m.history.Upsert(compoent.NewThinkingMessage(agentMsg.ID, agentMsg.Reasoning))
	case agent.MsgToolCall:
		m.history.Append(compoent.NewToolMessage(agentMsg.ID, agentMsg.ToolName, agentMsg.ToolArgs))
	case agent.MsgToolResult:
		hasErr := agentMsg.ToolErr != nil
		m.history.UpdateToolResult(agentMsg.ID, agentMsg.Content, hasErr)
	case agent.MsgError, agent.MsgRetryWait:
		m.history.Append(compoent.NewErrorMessage(agentMsg.Content))
	}

	if m.ch != nil {
		return []tea.Cmd{waitCmd(m.ch)}
	}
	return nil
}

// persistMessage persists only "complete" message types to the store.
// Streaming messages (thinking_stream, assistant_stream) are skipped.
func (m *model) persistMessage(msg agent.CallbackMsg) {
	if m.store == nil {
		return
	}

	sm := store.Message{
		SessionID: m.sessionID,
		MsgID:     msg.ID,
	}

	switch msg.Type {
	case agent.MsgThinking:
		sm.MsgType = string(agent.MsgThinking)
		sm.Reasoning = msg.Reasoning

	case agent.MsgAssistant:
		sm.MsgType = string(agent.MsgAssistant)
		sm.Content = msg.Content

	case agent.MsgToolCall:
		sm.MsgType = string(agent.MsgToolCall)
		sm.ToolCallID = msg.ToolCallID
		sm.ToolName = msg.ToolName
		sm.ToolArgs = msg.ToolArgs

	case agent.MsgToolResult:
		sm.MsgType = string(agent.MsgToolResult)
		sm.ToolCallID = msg.ToolCallID
		sm.Content = msg.Content
		sm.HasError = msg.ToolErr != nil

	default:
		return
	}

	m.store.AppendMessage(sm)
}
