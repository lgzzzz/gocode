package command

import "context"

// NewCommand implements the /new command: starts a fresh conversation
// by clearing all context history on both the agent and TUI sides.
type NewCommand struct{}

func (c *NewCommand) Name() string        { return "new" }
func (c *NewCommand) Description() string { return "开启一轮新的对话" }

func (c *NewCommand) Execute(ctx context.Context, args string, env *Env) (*Result, error) {
	// Cancel the running agent if any
	if env.Model.Running() {
		env.Model.CancelAgent()
	}

	// Clear agent conversation history
	env.Agent.ClearHistory()

	// Clear TUI message history
	env.Model.ClearHistory()

	return &Result{Message: "✨ 已开启新对话，上下文已清除。"}, nil
}
