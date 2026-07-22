package command

import "context"

// SessionsCommand implements the /sessions command: lists recent sessions.
type SessionsCommand struct{}

func (c *SessionsCommand) Name() string        { return "sessions" }
func (c *SessionsCommand) Description() string { return "列出最近的会话记录" }

func (c *SessionsCommand) Execute(ctx context.Context, args string, env *Env) (*Result, error) {
	content := env.Model.ListSessions()
	if content == "" {
		return &Result{Message: "📭 暂无历史会话。"}, nil
	}
	return &Result{Message: content}, nil
}
