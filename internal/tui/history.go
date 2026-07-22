package tui

import "github.com/lgzzzz/gocode/internal/tui/compoent"

// History stores the ordered list of chat message components and tracks
// whether the rendered view is stale (dirty). All mutations go through
// History so the dirty invariant is never broken.
type History struct {
	items []compoent.Component
	dirty bool
}

// Append adds a component and marks the history as dirty.
func (h *History) Append(c compoent.Component) {
	h.items = append(h.items, c)
	h.dirty = true
}

// Upsert searches backwards for a component with the same MsgID and Type,
// updates its content in-place if found, otherwise appends a new entry.
// Components with an empty MsgID are always appended (never matched).
// Returns true when an existing component was updated, false when appended.
func (h *History) Upsert(c compoent.Component) bool {
	if c.MsgID() == "" {
		h.Append(c)
		return false
	}
	for i := len(h.items) - 1; i >= 0; i-- {
		if h.items[i].MsgID() == c.MsgID() && h.items[i].Type() == c.Type() {
			h.items[i].SetContent(c.Content())
			h.dirty = true
			return true
		}
	}
	h.Append(c)
	return false
}

// UpdateToolResult searches backwards for a tool component identified
// by id, sets the result and optionally marks it as errored, and marks
// the history dirty. Returns false when no matching tool is found.
func (h *History) UpdateToolResult(id, result string, hasErr bool) bool {
	for i := len(h.items) - 1; i >= 0; i-- {
		if h.items[i].MsgID() == id && h.items[i].Type() == "tool" {
			if tm, ok := h.items[i].(*compoent.ToolMessage); ok {
				tm.SetResult(result)
				if hasErr {
					tm.SetError()
				}
			}
			h.dirty = true
			return true
		}
	}
	return false
}

// Clear removes all items from the history and marks it dirty.
func (h *History) Clear() {
	h.items = nil
	h.dirty = true
}

// MarkDirty forces the history to be considered stale (e.g. after a
// terminal resize that changes the available width).
func (h *History) MarkDirty() {
	h.dirty = true
}

// Render returns the rendered lines of every component for the given
// width. If the history is not dirty it returns (nil, false) so the
// caller can skip the expensive output update.
func (h *History) Render(width int) (lines []string, ok bool) {
	if !h.dirty {
		return nil, false
	}
	h.dirty = false

	for i, comp := range h.items {
		rendered := comp.Render(width)
		if rendered != "" {
			lines = append(lines, rendered)
			if i != len(h.items)-1 {
				lines = append(lines, "") // spacing between cards
			}
		}
	}
	return lines, true
}
