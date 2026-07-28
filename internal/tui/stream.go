package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/lgzzzz/gocode/internal/agent"
)

type progressMsg struct {
	ID         string
	Type       agent.MsgType
	Content    string
	ToolCallID string
	ToolName   string
	ToolArgs   string
	ToolErr    error
	done       bool
	err        error
}

// tickMsg is sent every 25ms to trigger a UI render during agent processing.
type tickMsg struct{}

// agentDoneMsg is sent when the agent finishes processing and the ticker stops.
type agentDoneMsg struct{}

func timerCmd(interval time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(interval)
		return tickMsg{}
	}
}

func agentDoneCmd(done <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		<-done
		return agentDoneMsg{}
	}
}
