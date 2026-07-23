package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/lgzzzz/gocode/internal/agent"
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)

// ---- progress message ----

type progressMsg struct {
	typ       agent.MsgType // callback message type
	id        string        // message ID (for streaming updates)
	content   string
	reasoning string // reasoning_content for assistant messages
	toolName  string // tool name (set for tool_call)
	toolArgs  string // tool arguments JSON (set for tool_call)
	toolErr   error  // tool execution error (set for tool_result)
	done      bool
	err       error // fatal / panic error
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
	case agent.MsgThinkingStream, agent.MsgThinking:
		return string(agent.MsgThinking)
	case agent.MsgAssistantStream, agent.MsgAssistant:
		return string(agent.MsgAssistant)
	default:
		return string(agent.MsgAssistant)
	}
}

// ---- streaming helpers ----

// applyStreamUpdate creates or updates a streaming component (assistant / thinking)
// in-place via the history's Upsert method.
func (m *model) applyStreamUpdate(msg progressMsg) {
	kind := componentTypeStr(msg.typ)
	var c compoent.Component
	switch kind {
	case string(agent.MsgAssistant):
		c = compoent.NewAssistantMessage(msg.id, msg.content)
	case string(agent.MsgThinking):
		c = compoent.NewThinkingMessage(msg.id, msg.content)
	default:
		return
	}
	m.history.Upsert(c)
}

// applyToolResult finds the matching tool-call component and sets its result,
// or creates a new one if the call was somehow missed (orphan result).
func (m *model) applyToolResult(msg progressMsg) {
	hasErr := msg.toolErr != nil
	if m.history.UpdateToolResult(msg.id, msg.content, hasErr) {
		return
	}
	// Orphan result — create a tool message with the result already set.
	tm := compoent.NewToolMessage(msg.id, msg.toolName, msg.toolArgs)
	tm.SetResult(msg.content)
	if msg.toolErr != nil {
		tm.SetError()
	}
	m.history.Append(tm)
}
