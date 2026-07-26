package command

import "context"

// initPrompt is the preset prompt sent to the LLM to generate AGENTS.md.
const initPrompt = `Please analyze the current project and generate an AGENTS.md file at .gocode/AGENTS.md in the current working directory.

The purpose of the AGENTS.md file is: when an AI assistant works in this project, this file will be automatically loaded as part of the system prompt, helping the AI better understand the project.

Please first use tools to explore the project structure and code (read key files, view directory structure, etc.), then generate the AGENTS.md file.

AGENTS.md should include the following:
1. Project overview (project name, purpose, tech stack)
2. Project directory structure description
3. Build and run instructions (how to compile, run, test)
4. Coding standards and conventions (code style, naming conventions, etc.)
5. Key module descriptions
6. Notes or special conventions

Please write based on the actual project content, do not fabricate non-existent information. Write AGENTS.md in English.`

type InitCommand struct{}

func (c *InitCommand) Name() string        { return "init" }
func (c *InitCommand) Description() string { return "Analyze the project and generate AGENTS.md" }

func (c *InitCommand) Execute(ctx context.Context, args string, env *Env) (*Result, error) {
	if env.TUI.Running() {
		env.TUI.CancelAgent()
	}
	env.TUI.NewSession()
	return &Result{
		Message:    "Analyzing project and generating AGENTS.md ...",
		AgentInput: initPrompt,
	}, nil
}
