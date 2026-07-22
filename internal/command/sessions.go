package command

import "context"

// SessionsCommand implements the /sessions command: lists recent sessions.
type SessionsCommand struct{}

func (c *SessionsCommand) Name() string        { return "sessions" }
func (c *SessionsCommand) Description() string { return "浏览并继续历史会话" }

func (c *SessionsCommand) Execute(ctx context.Context, args string, env *Env) (*Result, error) {
	env.Model.EnterSessionBrowser()
	return nil, nil // browser mode activated, no output message
}
