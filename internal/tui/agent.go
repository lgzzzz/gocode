package tui

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/store"
	"github.com/lgzzzz/gocode/internal/tools"
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)

func (m *model) StartAgent(input string) []tea.Cmd {
	// Clear rollback tracker for the new agent interaction
	m.rollbackTracker.Clear()

	m.setRunning(true)

	ctx, cancel := context.WithCancel(context.Background())
	m.setCancel(cancel)

	ch := make(chan progressMsg, 64)

	// Agent goroutine: runs the LLM call and feeds progress messages into ch.
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

	// doneCh signals the ticker to stop and emit a final agentDoneMsg.
	doneCh := make(chan struct{})

	// Background processor: drains ch and updates history in real time.
	go func() {
		for msg := range ch {
			m.handleProgressMsg(msg)
			if msg.done {
				m.setRunning(false)
				m.setCancel(nil)
				close(doneCh)
				return
			}
		}

	}()

	return []tea.Cmd{timerCmd(25 * time.Millisecond), agentDoneCmd(doneCh)}
}

func (m *model) Running() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.running
}

func (m *model) setRunning(val bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.running = val
}

func (m *model) getCancel() context.CancelFunc {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.cancel
}

func (m *model) setCancel(c context.CancelFunc) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.cancel = c
}

func (m *model) SystemPrompt() string { return m.agent.SystemPrompt() }

func (m *model) CancelAgent() { m.cancelAgent() }

func (m *model) SetSkipAgentsMD(skip bool) { m.agent.SetSkipAgentsMD(skip) }

func (m *model) RollbackTracker() *tools.RollbackTracker { return m.rollbackTracker }

// RollbackConversation truncates the agent context, TUI history, and persisted store messages
// to remove the last user interaction and everything after it.
func (m *model) RollbackConversation() (rollBacked bool) {
	ctxRollBacked := m.agent.TruncateContextFromLastUser()
	historyRollBacked := m.history.TruncateFromLastUser()
	var storeRollBacked int
	if m.store != nil {
		storeRollBacked = m.store.TruncateMessagesFromLastUser(m.sessionID)
	}
	m.rollbackTracker.Clear()
	if ctxRollBacked != 0 || historyRollBacked != 0 || storeRollBacked != 0 {
		return true
	}
	m.renderOutput()
	m.output.GotoBottom()
	return false
}

func (m *model) cancelAgent() {
	c := m.getCancel()
	if c != nil {
		c()
	}
}

// handleProgressMsg is called from the background processor goroutine.
// It updates the history and persists messages. State cleanup on completion
// (running=false, etc.) is handled in the Update loop upon receiving agentDoneMsg.
func (m *model) handleProgressMsg(msg progressMsg) {
	if msg.err != nil {
		m.history.Append(compoent.NewErrorMessage(msg.err.Error()))
		return
	}

	if msg.done {
		return
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
}

func (m *model) persistMessage(msg progressMsg) {
	if m.store == nil {
		return
	}
	if msg.Type != agent.MsgAssistant && msg.Type != agent.MsgThinking && msg.Type != agent.MsgToolCall && msg.Type != agent.MsgToolResult {
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
