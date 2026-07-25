package command

import (
	"context"
	"fmt"
	"strings"
)

// RollbackCommand implements the /rollback slash command.
// It restores files modified during the last agent interaction and displays
// shell commands that were executed (which cannot be automatically rolled back).
type RollbackCommand struct{}

func (c *RollbackCommand) Name() string { return "rollback" }
func (c *RollbackCommand) Description() string {
	return "Rollback file changes from the last agent interaction"
}

func (c *RollbackCommand) Execute(ctx context.Context, args string, env *Env) (*Result, error) {
	tracker := env.TUI.RollbackTracker()
	if tracker == nil {
		return &Result{Message: "Rollback tracker is not available."}, nil
	}

	var sb strings.Builder
	if tracker.HasChanges() {
		restoredFiles, shellCommands, errors := tracker.Rollback()
		// Report restored files
		if len(restoredFiles) > 0 {
			sb.WriteString("Restored files:\n")
			for _, f := range restoredFiles {
				sb.WriteString(fmt.Sprintf("  • %s\n", f))
			}
		}

		// Report shell commands that were executed
		if len(shellCommands) > 0 {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString("Shell commands executed (side effects may need manual reversal):\n")
			for _, sc := range shellCommands {
				sb.WriteString(fmt.Sprintf("  • %s\n", sc.Command))
			}
		}

		// Report any errors
		if len(errors) > 0 {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString("Errors during rollback:\n")
			for _, e := range errors {
				sb.WriteString(fmt.Sprintf("  • %v\n", e))
			}
		}
	}

	rollBacked := env.TUI.RollbackConversation()
	if rollBacked {
		// Report conversation rollback
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("Conversation context rolled back to before the last interaction.")
	}

	if sb.Len() == 0 {
		return &Result{Message: "Rollback complete, nothing to report."}, nil
	}

	// Remove trailing newline
	msg := strings.TrimSpace(sb.String())
	return &Result{Message: msg}, nil
}
