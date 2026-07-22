package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/lgzzzz/gocode/internal/agent"
)

// ---- progress message ----

type progressMsg struct {
	typ      agent.MsgType // callback message type
	id       string        // message ID (for streaming updates)
	content  string
	toolName string // tool name (set for tool_call)
	toolArgs string // tool arguments JSON (set for tool_call)
	toolErr  error  // tool execution error (set for tool_result)
	done     bool
	err      error // fatal / panic error
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

// componentTypeStr maps agent callback types to component type strings,
// used for finding the right component during streaming updates.
func componentTypeStr(t agent.MsgType) string {
	switch t {
	case agent.MsgThinkingStream:
		return "thinking"
	case agent.MsgAssistantStream:
		return "assistant"
	default:
		return "assistant"
	}
}
