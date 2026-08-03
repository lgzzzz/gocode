package command

import "context"

// commitPrompt is submitted to the conversation history and triggers the tool
// loop (git status / add / commit via the shell tool). Unlike /init and /update,
// it does not clear the conversation history.
const commitPrompt = `Commit all code, just commit, don't push.`

type CommitCommand struct{}

func (c *CommitCommand) Name() string { return "commit" }
func (c *CommitCommand) Description() string {
	return "Commit all code changes (no push)"
}

func (c *CommitCommand) Execute(ctx context.Context, args string, env *Env) (*Result, error) {
	if env.TUI.Running() {
		env.TUI.CancelAgent()
	}
	return &Result{
		Message:    "Committing all code ...",
		AgentInput: commitPrompt,
	}, nil
}
