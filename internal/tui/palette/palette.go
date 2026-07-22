package palette

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/lgzzzz/gocode/internal/command"
)

// ---- styles ----
var (
	// lineBase — shared left border + padding for every palette row.
	lineBase = lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("7")).
			Padding(0, 1)

	// highlightStyle — selected command row (inherits lineBase).
	highlightStyle = lineBase.
			Background(lipgloss.Color("4")). // blue background
			Foreground(lipgloss.Color("15")) // white text

	// dimStyle — normal (non-selected) command rows (inherits lineBase).
	dimStyle = lineBase.
			Foreground(lipgloss.Color("15"))
)

// ---- types ----

// KeyResult describes the outcome of processing a key press while the
// palette is active.  The caller (model) inspects this to decide what
// side effects to perform (reset editor, execute command, set text).
type KeyResult struct {
	Dismiss      bool             // esc — caller should reset the editor
	Execute      command.Executor // enter on a valid match
	CompleteText string           // tab — caller should set editor to this value
}

// Palette manages the state of the command palette popup: detection
// (text starting with "/"), filtering, keyboard navigation, rendering,
// and command selection.
type Palette struct {
	active   bool
	filter   string
	index    int
	matches  []command.Executor
	registry *command.Registry
	width    int
}

// SetWidth stores the terminal width used for rendering.
func (p *Palette) SetWidth(w int) {
	p.width = w
}

// New creates a new Palette backed by the given command registry.
func New(reg *command.Registry) *Palette {
	return &Palette{registry: reg}
}

// UpdateFilter reads the current editor text and updates the palette
// state: active when text starts with "/", filtering commands by the
// first word after "/".
func (p *Palette) UpdateFilter(editorText string) {
	if strings.HasPrefix(editorText, "/") {
		p.active = true
		p.filter = editorText[1:] // strip "/" prefix

		// Extract the first word for command matching.  This allows
		// the user to type arguments after the command name (e.g.
		// "/new some args") without breaking the filter.
		filterWord := strings.SplitN(p.filter, " ", 2)[0]
		p.matches = p.registry.Filter(filterWord)

		if len(p.matches) == 0 {
			// No matching commands: hide the palette entirely.
			p.active = false
			p.index = -1
		} else if p.index < 0 || p.index >= len(p.matches) {
			p.index = 0
		}
	} else {
		p.active = false
		p.index = 0
		p.matches = nil
	}
}

// HandleKey processes a single key press when the palette is active.  It
// mutates internal state for navigation keys (up/down) and returns a
// KeyResult for actions that require the caller to perform editor
// changes or command execution.
func (p *Palette) HandleKey(key string) KeyResult {
	switch key {
	case "esc":
		p.active = false
		p.index = 0
		p.matches = nil
		return KeyResult{Dismiss: true}

	case "enter":
		if len(p.matches) > 0 && p.index >= 0 {
			return KeyResult{Execute: p.matches[p.index]}
		}
		return KeyResult{}

	case "up":
		if p.index > 0 {
			p.index--
		}
		return KeyResult{}

	case "down":
		if p.index < len(p.matches)-1 {
			p.index++
		}
		return KeyResult{}

	case "tab":
		if len(p.matches) > 0 && p.index >= 0 {
			return KeyResult{
				CompleteText: "/" + p.matches[p.index].Name() + " ",
			}
		}
		return KeyResult{}

	default:
		return KeyResult{}
	}
}

// Active returns whether the command palette is currently visible.
func (p *Palette) Active() bool { return p.active }

// Dismiss resets the palette to inactive state without performing any
// editor side effects.
func (p *Palette) Dismiss() {
	p.active = false
	p.index = 0
	p.matches = nil
}

// Args extracts the arguments portion of the filter after the given
// command name.  For example, with filter "new my project", Args("new")
// returns "my project".
func (p *Palette) Args(cmdName string) string {
	if len(p.filter) >= len(cmdName) &&
		strings.EqualFold(p.filter[:len(cmdName)], cmdName) {
		return strings.TrimSpace(p.filter[len(cmdName):])
	}
	return ""
}

// ---- rendering ----

const maxPaletteRows = 7

// Height returns the number of rows the palette occupies (0 when
// inactive), used for layout calculations.
func (p *Palette) Height() int {
	if !p.active {
		return 0
	}
	n := len(p.matches)
	if n == 0 {
		return 0 // minimum (empty tip + border)
	}
	if n > maxPaletteRows {
		n = maxPaletteRows
	}
	return n // content + border
}

// Render returns the styled command palette content, or an empty string
// when there is nothing to show.
func (p *Palette) Render() string {
	paletteWidth := p.width
	if len(p.matches) == 0 {
		return ""
	}

	lines := make([]string, 0, len(p.matches))
	for i, cmd := range p.matches {
		line := fmt.Sprintf("/%-12s %s", cmd.Name(), cmd.Description())
		if i == p.index {
			line = highlightStyle.Width(paletteWidth).Render(line)
		} else {
			line = dimStyle.Width(paletteWidth).Render(line)
		}
		lines = append(lines, line)
	}

	start := 0
	if p.index >= maxPaletteRows {
		start = p.index - maxPaletteRows + 1
	}
	end := min(start+maxPaletteRows, len(lines))
	visible := lines[start:end]

	return lipgloss.JoinVertical(lipgloss.Left, visible...)
}
