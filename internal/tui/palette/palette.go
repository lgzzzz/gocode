package palette

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/lgzzzz/gocode/internal/command"
)

var (
	lineBase = lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("7")).
			Padding(0, 1)

	highlightStyle = lineBase.
			Background(lipgloss.Color("4")).
			Foreground(lipgloss.Color("15"))

	dimStyle = lineBase.
			Foreground(lipgloss.Color("15"))
)


type KeyResult struct {
	Dismiss      bool
	Execute      command.Executor
	CompleteText string
}

type Palette struct {
	active   bool
	filter   string
	index    int
	matches  []command.Executor
	registry *command.Registry
	width    int
}

func (p *Palette) SetWidth(w int) {
	p.width = w
}

func New(reg *command.Registry) *Palette {
	return &Palette{registry: reg}
}

func (p *Palette) UpdateFilter(editorText string) {
	if strings.HasPrefix(editorText, "/") {
		p.active = true
		p.filter = editorText[1:]

		filterWord := strings.SplitN(p.filter, " ", 2)[0]
		p.matches = p.registry.Filter(filterWord)

		if len(p.matches) == 0 {
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

func (p *Palette) Active() bool { return p.active }

func (p *Palette) Dismiss() {
	p.active = false
	p.index = 0
	p.matches = nil
}

func (p *Palette) Args(cmdName string) string {
	if len(p.filter) >= len(cmdName) &&
		strings.EqualFold(p.filter[:len(cmdName)], cmdName) {
		return strings.TrimSpace(p.filter[len(cmdName):])
	}
	return ""
}


const maxPaletteRows = 7

func (p *Palette) Height() int {
	if !p.active {
		return 0
	}
	n := len(p.matches)
	if n == 0 {
		return 0
	}
	if n > maxPaletteRows {
		n = maxPaletteRows
	}
	return n
}

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
