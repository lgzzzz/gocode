package history

import "github.com/lgzzzz/gocode/internal/tui/compoent"

type History struct {
	items []compoent.Component
	dirty bool
}

func (h *History) Append(c compoent.Component) {
	h.items = append(h.items, c)
	h.dirty = true
}

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

func (h *History) Len() int {
	return len(h.items)
}

// TruncateFromLastUser removes the last user message and everything after it.
// Returns the number of items removed (0 if no user message found).
func (h *History) TruncateFromLastUser() int {
	// Find the index of the last user message
	lastUserIdx := -1
	for i := len(h.items) - 1; i >= 0; i-- {
		if h.items[i].Type() == "user" {
			lastUserIdx = i
			break
		}
	}
	if lastUserIdx == -1 {
		return 0
	}
	removed := len(h.items) - lastUserIdx
	h.items = h.items[:lastUserIdx]
	h.dirty = true
	return removed
}

func (h *History) Clear() {
	h.items = nil
	h.dirty = true
}

func (h *History) MarkDirty() {
	h.dirty = true
}

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
				lines = append(lines, "")
			}
		}
	}
	return lines, true
}
