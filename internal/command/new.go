package command

import "context"

type NewCommand struct{}

func (c *NewCommand) Name() string        { return "new" }
func (c *NewCommand) Description() string { return "Start a new conversation" }

func (c *NewCommand) Execute(ctx context.Context, args string, env *Env) (*Result, error) {
	if env.TUI.Running() {
		env.TUI.CancelAgent()
	}
	env.TUI.NewSession()
	return &Result{Message: "New conversation started, context cleared."}, nil
}
