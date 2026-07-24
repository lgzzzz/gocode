package tui

import (
	"os"

	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/store"
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)


func (m *model) NewSession() {
	m.agent.ClearContextMessage()
	m.history.Clear()
	m.sessionID = store.NewSessionID()
	m.cwd, _ = os.Getwd()
}

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

func (m *model) CloseSessionBrowser() {
	m.sessionBrowser.SetActive(false)
}

func (m *model) LoadSession(sessionID string) {
	msgs, err := m.sessionBrowser.GetMessages(sessionID)
	if err != nil {
		m.history.Append(compoent.NewErrorMessage(err.Error()))
		return
	}

	m.history.Clear()
	for i := range msgs {
		msg := msgs[i]
		switch msg.MsgType {
		case string(agent.MsgUser):
			m.history.Append(compoent.NewUserMessage(msg.Content))
		case string(agent.MsgAssistant):
			m.history.Append(compoent.NewAssistantMessage(msg.MsgID, msg.Content))
		case string(agent.MsgThinking):
			m.history.Append(compoent.NewThinkingMessage(msg.MsgID, msg.Content))
		case string(agent.MsgToolCall):
			m.history.Append(compoent.NewToolMessage(msg.MsgID, msg.ToolName, msg.ToolArgs))
		case string(agent.MsgToolResult):
			m.history.UpdateToolResult(msg.MsgID, msg.Content, msg.HasError)
		}
	}

	agentMsgs := make([]agent.HistoryMessage, len(msgs))
	for i, m := range msgs {
		agentMsgs[i] = agent.HistoryMessage{
			MsgType:    m.MsgType,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
			ToolName:   m.ToolName,
			ToolArgs:   m.ToolArgs,
		}
	}
	openaiHistory := agent.ReconstructHistory(agentMsgs, m.agent.SystemPrompt())
	m.agent.SetContextMessage(openaiHistory)

	m.sessionID = sessionID
	m.sessionBrowser.SetActive(false)
	m.output.GotoBottom()
}
