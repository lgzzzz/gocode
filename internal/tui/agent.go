package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/store"
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)

// ---- agent actions ----

// startAgent sends the current editor content to the agent for processing.
func (m *model) startAgent() tea.Cmd {
	input := strings.TrimSpace(m.editor.Value())
	if input == "" {
		return nil
	}
	m.editor.Reset()
	m.history.Append(compoent.NewUserMessage(input))
	m.running = true

	// Persist user message.
	if m.store != nil {
		m.store.AppendMessage(store.Message{
			SessionID: m.sessionID,
			Role:      "user",
			MsgType:   "user_message",
			Content:   input,
		})
	}

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
