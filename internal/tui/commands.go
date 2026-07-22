package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/lgzzzz/gocode/internal/command"
	"github.com/lgzzzz/gocode/internal/tui/compoent"
)

// ---- command execution ----

// executeCommand runs the given command with the provided arguments,
// appends the result to the chat history.  The caller is responsible
// for dismissing the palette and resetting the editor beforehand.
func (m *model) executeCommand(cmd command.Executor, args string) tea.Cmd {
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
