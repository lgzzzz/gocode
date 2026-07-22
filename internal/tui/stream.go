package tui

import (
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)

// ---- streaming helpers ----

// applyStreamUpdate finds or creates a streaming component (assistant / thinking)
// and updates its content in-place.
func (m *model) applyStreamUpdate(msg progressMsg) {
	kind := componentTypeStr(msg.typ)
	if m.history.UpdateByID(msg.id, kind, msg.content) {
		return
	}
	// Not found — append new streaming component.
	switch kind {
	case "assistant":
		m.history.Append(compoent.NewAssistantMessage(msg.id, msg.content))
	case "thinking":
		m.history.Append(compoent.NewThinkingMessage(msg.id, msg.content))
	}
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
