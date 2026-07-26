package command

import "context"

const updatePrompt = `Please update the existing AGENTS.md file at .gocode/AGENTS.md in the current working directory.

First, use the read tool to read the current .gocode/AGENTS.md to understand the existing content.
Then, explore the project to identify any changes since the AGENTS.md was last written — for example:
- New or removed files and directories
- New or changed modules, packages, or commands
- Updated build instructions, dependencies, or configuration
- Changes to coding conventions or project structure

After identifying the changes, update .gocode/AGENTS.md to accurately reflect the current state of the project. Preserve the existing structure and writing style of AGENTS.md. Only modify sections that are outdated; do not rewrite the entire file from scratch unless necessary.

Important:
- Keep all information that is still accurate — do not remove or change correct content
- Add new sections only if the project has gained new capabilities or modules
- Write in English
- Use the edit tool for precise, minimal changes to the file`

type UpdateCommand struct{}

func (c *UpdateCommand) Name() string { return "update" }
func (c *UpdateCommand) Description() string {
	return "Update AGENTS.md based on recent project changes"
}

func (c *UpdateCommand) Execute(ctx context.Context, args string, env *Env) (*Result, error) {
	if env.TUI.Running() {
		env.TUI.CancelAgent()
	}
	env.TUI.SetSkipAgentsMD(true)
	env.TUI.NewSession()
	return &Result{
		Message:    "Analyzing project changes and updating AGENTS.md ...",
		AgentInput: updatePrompt,
	}, nil
}
