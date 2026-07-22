package tui

import (
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)

// ---- streaming helpers ----

// applyStreamUpdate creates or updates a streaming component (assistant / thinking)
// in-place via the history's Upsert method.
func (m *model) applyStreamUpdate(msg progressMsg) {
	kind := componentTypeStr(msg.typ)
	var c compoent.Component
	switch kind {
	case "assistant":
		c = compoent.NewAssistantMessage(msg.id, msg.content)
	case "thinking":
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
