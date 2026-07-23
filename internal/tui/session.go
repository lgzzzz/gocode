package tui

import (
	"os"

	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/store"
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)

// ---- ModelAccess interface implementation ----

// NewSession swaps to a new session ID and clears TUI history.
func (m *model) NewSession() {
	m.agent.ClearContextMessage()
	m.history.Clear()
	m.sessionID = store.NewSessionID()
	m.cwd, _ = os.Getwd()
}

// OpenSessionBrowser loads sessions from the store and activates
// the interactive session browser.
func (m *model) OpenSessionBrowser() {
	m.sessionBrowser.SetSize(m.width, m.height)
	if err := m.sessionBrowser.Reload(); err != nil {
		m.history.Append(compoent.NewErrorMessage(err.Error()))
		return
	}
	if m.sessionBrowser.IsEmpty() {
		m.history.Append(compoent.NewSystemMessage("📭 暂无历史会话"))
		return
	}
	m.sessionBrowser.SetActive(true)
}

// CloseSessionBrowser deactivates the session browser.
func (m *model) CloseSessionBrowser() {
	m.sessionBrowser.SetActive(false)
}

// LoadSession loads all messages from the given session and rebuilds
// both the TUI history and the agent's conversation history.
func (m *model) LoadSession(sessionID string) {
	msgs, err := m.sessionBrowser.GetMessages(sessionID)
	if err != nil {
		m.history.Append(compoent.NewErrorMessage(err.Error()))
		return
	}

	// 1. Rebuild TUI history: iterate and pair tool_call/tool_result.
	m.history.Clear()
	for i := 0; i < len(msgs); i++ {
		msg := msgs[i]
		switch msg.MsgType {
		case string(agent.MsgUser):
			m.history.Append(compoent.NewUserMessage(msg.Content))
		case string(agent.MsgAssistant):
			m.history.Append(compoent.NewAssistantMessage(msg.MsgID, msg.Content))
		case string(agent.MsgThinking):
			m.history.Append(compoent.NewThinkingMessage(msg.MsgID, msg.Reasoning))
		case string(agent.MsgToolCall):
			m.history.Append(compoent.NewToolMessage(msg.MsgID, msg.ToolName, msg.ToolArgs))
		case string(agent.MsgToolResult):
			m.history.UpdateToolResult(msg.MsgID, msg.Content, msg.HasError)
		}
	}

	// 2. Rebuild Agent history (with reasoning_content).
	agentMsgs := make([]agent.HistoryMessage, len(msgs))
	for i, m := range msgs {
		agentMsgs[i] = agent.HistoryMessage{
			MsgType:    m.MsgType,
			Content:    m.Content,
			Reasoning:  m.Reasoning,
			ToolCallID: m.ToolCallID,
			ToolName:   m.ToolName,
			ToolArgs:   m.ToolArgs,
		}
	}
	openaiHistory := agent.ReconstructHistory(agentMsgs, m.agent.SystemPrompt())
	m.agent.SetContextMessage(openaiHistory)

	// 3. Switch to the loaded session.
	m.sessionID = sessionID
	m.sessionBrowser.SetActive(false)
	m.output.GotoBottom()
}
