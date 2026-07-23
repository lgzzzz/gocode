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
				ID:         msg.ID,
				Type:       msg.Type,
				Content:    msg.Content,
				ToolCallID: msg.ToolCallID,
				ToolName:   msg.ToolName,
				ToolArgs:   msg.ToolArgs,
				ToolErr:    msg.ToolErr,
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
	m.persistMessage(msg)
	switch msg.Type {
	case agent.MsgAssistantStream:
		m.history.Upsert(compoent.NewAssistantMessage(msg.ID, msg.Content))
	case agent.MsgThinkingStream:
		m.history.Upsert(compoent.NewThinkingMessage(msg.ID, msg.Content))
	case agent.MsgToolCall:
		m.history.Append(compoent.NewToolMessage(msg.ID, msg.ToolName, msg.ToolArgs))
	case agent.MsgToolResult:
		hasErr := msg.ToolErr != nil
		m.history.UpdateToolResult(msg.ID, msg.Content, hasErr)
	case agent.MsgError, agent.MsgRetryWait:
		m.history.Append(compoent.NewErrorMessage(msg.Content))
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
		MsgID:      msg.ID,
		MsgType:    string(msg.Type),
		Content:    msg.Content,
		ToolCallID: msg.ToolCallID,
		ToolName:   msg.ToolName,
		ToolArgs:   msg.ToolArgs,
		HasError:   msg.ToolErr != nil,
	}
	m.store.AppendMessage(sm)
}
