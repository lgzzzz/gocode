package command

import "context"

type NewCommand struct{}

func (c *NewCommand) Name() string        { return "new" }
func (c *NewCommand) Description() string { return "开启一轮新的对话" }

func (c *NewCommand) Execute(ctx context.Context, args string, env *Env) (*Result, error) {
	if env.TUI.Running() {
		env.TUI.CancelAgent()
	}
	env.TUI.NewSession()

	return &Result{Message: "✨ 已开启新对话，上下文已清除。"}, nil
}
