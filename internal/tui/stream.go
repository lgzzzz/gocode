package tui

import (
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

func waitCmd(ch chan progressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}
