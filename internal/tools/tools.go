package tools

import (
	"runtime"
	"strings"
)

// ToolExecutor is the interface that all tools must implement.
type ToolExecutor interface {
	Execute(argsJSON string) (string, error)
	Name() string
	SetTracker(tracker *RollbackTracker)
}

// ToolDef describes a tool for the LLM system prompt and API function definitions.
type ToolDef struct {
	Name             string
	Description      string
	Parameters       any
	PromptSnippet    string
	PromptGuidelines []string
}

// dirOf returns the parent directory of a file path.
func dirOf(path string) string {
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[:idx]
	}
	return "."
}

// AllTools returns the map of tool executors and their definitions.
func AllTools() (map[string]ToolExecutor, []ToolDef) {
	tools := map[string]ToolExecutor{
		"read":  &ReadTool{},
		"write": &WriteTool{},
		"edit":  &EditTool{},
	}

	if runtime.GOOS == "windows" {
		tools["powershell"] = &PowershellTool{}
	} else {
		tools["bash"] = &BashTool{}
	}

	defs := []ToolDef{
		{
			Name:             "read",
			Description:      "Read contents of a text file. Returns file content as text.",
			PromptSnippet:    "Read file contents.",
			PromptGuidelines: []string{"Use read to examine files instead of cat or sed."},
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":   map[string]any{"type": "string", "description": "Path to the file to read (relative or absolute)"},
					"offset": map[string]any{"type": "integer", "description": "Line number to start reading from (1-indexed)"},
					"limit":  map[string]any{"type": "integer", "description": "Maximum number of lines to read"},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:             "write",
			Description:      "Create or overwrite a file with the given content. Creates parent directories as needed.",
			PromptSnippet:    "Create or overwrite files.",
			PromptGuidelines: []string{"Use write only for new files or complete rewrites."},
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":    map[string]any{"type": "string", "description": "Path to the file to write (relative or absolute)"},
					"content": map[string]any{"type": "string", "description": "Content to write to the file"},
				},
				"required": []string{"path", "content"},
			},
		},
		{
			Name:          "edit",
			Description:   "Edit a file by replacing an exact text match with new text. oldText must be unique in the file.",
			PromptSnippet: "Make precise text replacements in files.",
			PromptGuidelines: []string{
				"Use edit for precise, small changes; use write only for new files or complete rewrites.",
				"When edit fails because oldText is not unique, read the file around the target area and try again with more context.",
			},
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":    map[string]any{"type": "string", "description": "Path to the file to edit (relative or absolute)"},
					"oldText": map[string]any{"type": "string", "description": "Exact text to find and replace (must be unique in the file)"},
					"newText": map[string]any{"type": "string", "description": "Replacement text"},
				},
				"required": []string{"path", "oldText", "newText"},
			},
		},
	}

	if runtime.GOOS == "windows" {
		defs = append(defs, ToolDef{
			Name:          "powershell",
			Description:   "Execute a shell command on Windows systems. Runs via PowerShell. Returns stdout and stderr combined.",
			PromptSnippet: "Execute shell commands on Windows systems.",
			PromptGuidelines: []string{
				"PowerShell uses `;` instead of `&&`.",
			},
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{"type": "string", "description": "Shell command to execute"},
					"timeout": map[string]any{"type": "integer", "description": "Timeout in seconds (default 30)"},
				},
				"required": []string{"command"},
			},
		})
	} else {
		defs = append(defs, ToolDef{
			Name:          "bash",
			Description:   "Execute a shell command on Linux/Unix systems. Runs via bash. Returns stdout and stderr combined.",
			PromptSnippet: "Execute shell commands on Linux/Unix systems.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{"type": "string", "description": "Shell command to execute"},
					"timeout": map[string]any{"type": "integer", "description": "Timeout in seconds (default 30)"},
				},
				"required": []string{"command"},
			},
		})
	}

	return tools, defs
}
