package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lgzzzz/gocode/internal/command"
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)

// ---- command mode management ----

// updateCommandMode checks whether the editor content starts with "/" and
// updates the command palette state accordingly. Call this after every
// editor update.
func (m *model) updateCommandMode() {
	val := m.editor.Value()
	if strings.HasPrefix(val, "/") {
		m.commandMode = true
		m.commandFilter = val[1:] // strip "/" prefix

		// Extract the first word from the filter for command matching.
		// This allows the user to type arguments after the command name
		// (e.g. "/new some args") without breaking the filter.
		filterWord := strings.SplitN(m.commandFilter, " ", 2)[0]
		m.commandMatches = m.commandRegistry.Filter(filterWord)

		// Clamp commandIndex to valid range
		if len(m.commandMatches) == 0 {
			m.commandIndex = -1
		} else {
			if m.commandIndex < 0 || m.commandIndex >= len(m.commandMatches) {
				m.commandIndex = 0
			}
		}
	} else {
		m.commandMode = false
		m.commandIndex = 0
		m.commandMatches = nil
	}
}

// ---- command palette rendering ----

const maxPaletteRows = 8

// renderCommandPalette renders the command palette popup. Returns an empty
// string when there is nothing to show (no matches and no filter).
func (m *model) renderCommandPalette() string {
	paletteWidth := m.width - 2
	if paletteWidth < 20 {
		paletteWidth = 20
	}

	if len(m.commandMatches) == 0 {
		// No matches — show a hint
		msg := "无匹配命令"
		if m.commandFilter == "" {
			msg = "输入命令名称进行搜索..."
		}
		style := commandPaletteStyle.Width(paletteWidth)
		return style.Render(commandDimStyle.Render(msg))
	}

	// Build the list of command lines
	lines := make([]string, 0, len(m.commandMatches))
	for i, cmd := range m.commandMatches {
		line := fmt.Sprintf("/%-12s %s", cmd.Name(), cmd.Description())
		if i == m.commandIndex {
			line = commandHighlightStyle.Render(line)
		} else {
			line = commandDimStyle.Render(line)
		}
		lines = append(lines, line)
	}

	// Scroll so the highlighted item stays visible
	start := 0
	if m.commandIndex >= maxPaletteRows {
		start = m.commandIndex - maxPaletteRows + 1
	}
	end := min(start+maxPaletteRows, len(lines))
	visible := lines[start:end]

	style := commandPaletteStyle.Width(paletteWidth)
	return style.Render(lipgloss.JoinVertical(lipgloss.Left, visible...))
}

// ---- command execution ----

// executeCommand runs the given command, resets the editor and palette state,
// and appends the result to the chat history.
func (m *model) executeCommand(cmd command.Executor) tea.Cmd {
	// Capture args before resetting the editor
	args := ""
	if m.commandFilter != "" {
		// Remove the command name from the filter to get the args.
		// e.g. filter "new some args" → args "some args"
		// Use case-insensitive prefix removal.
		cmdName := cmd.Name()
		if len(m.commandFilter) >= len(cmdName) &&
			strings.EqualFold(m.commandFilter[:len(cmdName)], cmdName) {
			args = strings.TrimSpace(m.commandFilter[len(cmdName):])
		}
	}

	m.editor.Reset()
	m.commandMode = false
	m.commandIndex = 0
	m.commandMatches = nil

	// Build the environment for command execution
	env := &command.Env{
		Agent: m.agent,
		Model: m, // *model implements ModelAccess directly
	}

	ctx := context.Background()
	result, err := cmd.Execute(ctx, args, env)
	if err != nil {
		m.history.Append(compoent.NewErrorMessage(err.Error()))
	} else if result != nil {
		m.history.Append(compoent.NewSystemMessage(result.Message))
	}
	return nil
}
