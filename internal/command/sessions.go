package command

import "context"

type SessionsCommand struct{}

func (c *SessionsCommand) Name() string        { return "sessions" }
func (c *SessionsCommand) Description() string { return "浏览并继续历史会话" }

func (c *SessionsCommand) Execute(ctx context.Context, args string, env *Env) (*Result, error) {
	env.TUI.OpenSessionBrowser()
	return nil, nil
}
