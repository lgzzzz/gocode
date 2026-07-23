package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/lgzzzz/gocode/internal/agent"
)

// ---- progress message ----

type progressMsg struct {
	ID         string        // message ID
	Type       agent.MsgType // event type
	Content    string
	ToolCallID string // tool call ID (set for tool_call and tool_result)
	ToolName   string // tool name (set for tool_call)
	ToolArgs   string // tool arguments JSON (set for tool_call)
	ToolErr    error  // tool execution error (set for tool_result)
	done       bool
	err        error // fatal / panic error
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
