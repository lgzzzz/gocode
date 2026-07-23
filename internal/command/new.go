package command

import "context"

// NewCommand implements the /new command: starts a fresh conversation
// by clearing all context history on both the agent and TUI sides.
type NewCommand struct{}

func (c *NewCommand) Name() string        { return "new" }
func (c *NewCommand) Description() string { return "开启一轮新的对话" }

func (c *NewCommand) Execute(ctx context.Context, args string, env *Env) (*Result, error) {
	// Cancel the running agent if any
	if env.TUI.Running() {
		env.TUI.CancelAgent()
	}
	// Create a new session in the store and clear TUI message history
	env.TUI.NewSession()

	return &Result{Message: "✨ 已开启新对话，上下文已清除。"}, nil
}
