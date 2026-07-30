package command

import (
	"context"
	"fmt"

	"github.com/atotto/clipboard"
)

type CopyCommand struct{}

func (c *CopyCommand) Name() string        { return "copy" }
func (c *CopyCommand) Description() string { return "Copy the last assistant message to clipboard" }

func (c *CopyCommand) Execute(ctx context.Context, args string, env *Env) (*Result, error) {
	content := env.TUI.LastAssistantContent()
	if content == "" {
		return &Result{Message: "No assistant message to copy."}, nil
	}

	if err := clipboard.WriteAll(content); err != nil {
		return nil, fmt.Errorf("failed to copy to clipboard: %w", err)
	}
	return &Result{Message: "Copied to clipboard."}, nil
}
