package command

import "context"

type PromptCommand struct{}

func (c *PromptCommand) Name() string        { return "prompt" }
func (c *PromptCommand) Description() string { return "Display the current system prompt" }

func (c *PromptCommand) Execute(ctx context.Context, args string, env *Env) (*Result, error) {
	prompt := env.TUI.SystemPrompt()
	return &Result{Message: prompt}, nil
}
