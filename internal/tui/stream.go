package tui

import (
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)

// ---- streaming helpers ----

// applyStreamUpdate finds or creates a streaming component (assistant / thinking)
// and updates its content in-place.
func (m *model) applyStreamUpdate(msg progressMsg) {
	kind := componentTypeStr(msg.typ)
	for i := len(m.log) - 1; i >= 0; i-- {
		if m.log[i].MsgID() == msg.id && m.log[i].Type() == kind {
			m.log[i].SetContent(msg.content)
			m.dirty = true
			return
		}
	}
	// Not found — append new streaming component.
	switch kind {
	case "assistant":
		m.appendLog(compoent.NewAssistantMessage(msg.id, msg.content))
	case "thinking":
		m.appendLog(compoent.NewThinkingMessage(msg.id, msg.content))
	}
}

// applyToolResult finds the matching tool-call component and sets its result,
// or creates a new one if the call was somehow missed (orphan result).
func (m *model) applyToolResult(msg progressMsg) {
	for i := len(m.log) - 1; i >= 0; i-- {
		if m.log[i].MsgID() == msg.id && m.log[i].Type() == "tool" {
			if tm, ok := m.log[i].(*compoent.ToolMessage); ok {
				tm.SetResult(msg.content)
				if msg.toolErr != nil {
					tm.SetError()
				}
			}
			m.dirty = true
			return
		}
	}
	// Orphan result — create a tool message with the result already set.
	tm := compoent.NewToolMessage(msg.id, msg.toolName, msg.toolArgs)
	tm.SetResult(msg.content)
	if msg.toolErr != nil {
		tm.SetError()
	}
	m.appendLog(tm)
}
