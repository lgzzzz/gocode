package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ToolExecutor knows how to execute a tool given JSON arguments.
type ToolExecutor interface {
	Execute(argsJSON string) (string, error)
	Name() string
}

// ToolDef describes a tool for OpenAI function calling and system prompt generation.
type ToolDef struct {
	Name             string
	Description      string
	Parameters       any
	PromptSnippet    string   // one-line description for the system prompt tool list
	PromptGuidelines []string // guidelines added to system prompt when tool is available
}

// ---- read tool ----

type ReadTool struct{}

func (t *ReadTool) Name() string { return "read" }

type readArgs struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

func (t *ReadTool) Execute(argsJSON string) (string, error) {
	var args readArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("read: bad arguments: %w", err)
	}
	if args.Path == "" {
		return "", fmt.Errorf("read: path is required")
	}
	info, err := os.Stat(args.Path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", args.Path, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("read: %s is a directory, not a file", args.Path)
	}
	data, err := os.ReadFile(args.Path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", args.Path, err)
	}
	lines := strings.Split(string(data), "\n")
	if args.Offset > 0 {
		if args.Offset > len(lines) {
			return "", fmt.Errorf("read: offset %d exceeds file length %d lines", args.Offset, len(lines))
		}
		lines = lines[args.Offset-1:]
	}
	if args.Limit > 0 && args.Limit < len(lines) {
		lines = lines[:args.Limit]
	}
	result := strings.Join(lines, "\n")
	if len(result) > 50000 {
		result = result[:50000] + "\n... [truncated]"
	}
	return result, nil
}

// ---- write tool ----

type WriteTool struct{}

func (t *WriteTool) Name() string { return "write" }

type writeArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (t *WriteTool) Execute(argsJSON string) (string, error) {
	var args writeArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("write: bad arguments: %w", err)
	}
	if args.Path == "" {
		return "", fmt.Errorf("write: path is required")
	}
	if err := os.MkdirAll(dirOf(args.Path), 0755); err != nil {
		return "", fmt.Errorf("write: %w", err)
	}
	if err := os.WriteFile(args.Path, []byte(args.Content), 0644); err != nil {
		return "", fmt.Errorf("write %s: %w", args.Path, err)
	}
	return fmt.Sprintf("Wrote %d bytes to %s", len(args.Content), args.Path), nil
}

// ---- edit tool ----

type EditTool struct{}

func (t *EditTool) Name() string { return "edit" }

type editArgs struct {
	Path    string `json:"path"`
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
}

func (t *EditTool) Execute(argsJSON string) (string, error) {
	var args editArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("edit: bad arguments: %w", err)
	}
	if args.Path == "" {
		return "", fmt.Errorf("edit: path is required")
	}
	if args.OldText == "" {
		return "", fmt.Errorf("edit: oldText is required")
	}
	info, err := os.Stat(args.Path)
	if err != nil {
		return "", fmt.Errorf("edit: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("edit: %s is a directory, not a file", args.Path)
	}
	data, err := os.ReadFile(args.Path)
	if err != nil {
		return "", fmt.Errorf("edit: %w", err)
	}
	content := string(data)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	count := strings.Count(content, args.OldText)
	if count == 0 {
		return "", fmt.Errorf("edit: oldText not found in %s", args.Path)
	}
	if count > 1 {
		return "", fmt.Errorf("edit: oldText matches %d times in %s — must be unique", count, args.Path)
	}
	newContent := strings.Replace(content, args.OldText, args.NewText, 1)
	if err := os.WriteFile(args.Path, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("edit %s: %w", args.Path, err)
	}
	return fmt.Sprintf("Edited %s: replaced 1 occurrence", args.Path), nil
}

// ---- bash tool (Linux/Unix) ----

type BashTool struct{}

func (t *BashTool) Name() string { return "bash" }

type bashArgs struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

func (t *BashTool) Execute(argsJSON string) (string, error) {
	var args bashArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("bash: bad arguments: %w", err)
	}
	if args.Command == "" {
		return "", fmt.Errorf("bash: command is required")
	}
	return runShellCommand(args.Command, args.Timeout, "bash", buildLinuxShellCmd)
}

// buildLinuxShellCmd returns a shell command using bash (fallback sh).
func buildLinuxShellCmd(command string) *exec.Cmd {
	if _, err := exec.LookPath("bash"); err == nil {
		return exec.Command("bash", "-c", command)
	}
	return exec.Command("sh", "-c", command)
}

// ---- powershell tool (Windows) ----

type PowershellTool struct{}

func (t *PowershellTool) Name() string { return "powershell" }

type powershellArgs struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

func (t *PowershellTool) Execute(argsJSON string) (string, error) {
	var args powershellArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("powershell: bad arguments: %w", err)
	}
	if args.Command == "" {
		return "", fmt.Errorf("powershell: command is required")
	}
	return runShellCommand(args.Command, args.Timeout, "powershell", buildWindowsShellCmd)
}

// buildWindowsShellCmd returns a shell command using PowerShell (fallback cmd).
// The command is wrapped to ensure output encoding is UTF-8, so Chinese and other
// Unicode characters are preserved correctly.
func buildWindowsShellCmd(command string) *exec.Cmd {
	if _, err := exec.LookPath("powershell.exe"); err == nil {
		// Prepend UTF-8 output encoding setup before the user's command.
		// This ensures that tools like echo/Write-Output emit UTF-8 text.
		wrappedCmd := "[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; " + command
		return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", wrappedCmd)
	}
	return exec.Command("cmd", "/c", command)
}

// runShellCommand executes a shell command with a timeout and returns combined stdout/stderr.
func runShellCommand(command string, timeoutSec int, toolName string, buildCmd func(string) *exec.Cmd) (string, error) {
	timeout := 30
	if timeoutSec > 0 {
		timeout = timeoutSec
	}
	cmd := buildCmd(command)
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out

	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	select {
	case err := <-done:
		result := out.String()
		if len(result) > 10000 {
			result = result[:10000] + "\n... [truncated]"
		}
		if err != nil {
			return fmt.Sprintf("exit %v\n%s", err, result), nil
		}
		if result == "" {
			result = "(no output)"
		}
		return result, nil
	case <-time.After(time.Duration(timeout) * time.Second):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return fmt.Sprintf("(timed out after %ds)\n%s", timeout, out.String()), nil
	}
}

// ---- helpers ----

func dirOf(path string) string {
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[:idx]
	}
	return "."
}

// AllTools returns a map of tool executors and their OpenAI function definitions.
// Shell tools are selected based on the operating system:
//   - Windows: powershell (with cmd fallback)
//   - Linux/macOS/Unix: bash (with sh fallback)
func AllTools() (map[string]ToolExecutor, []ToolDef) {
	tools := map[string]ToolExecutor{
		"read":  &ReadTool{},
		"write": &WriteTool{},
		"edit":  &EditTool{},
	}

	// Select the appropriate shell tool based on OS
	if runtime.GOOS == "windows" {
		tools["powershell"] = &PowershellTool{}
	} else {
		tools["bash"] = &BashTool{}
	}

	defs := []ToolDef{
		{
			Name:             "read",
			Description:      "Read contents of a text file. Returns file content as text.",
			PromptSnippet:    "Read file contents",
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
			PromptSnippet:    "Create or overwrite files",
			PromptGuidelines: []string{"Use write only for new files or complete rewrites"},
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
			PromptSnippet: "Make precise text replacements in files",
			PromptGuidelines: []string{
				"Use edit for precise, small changes; use write only for new files or complete rewrites",
				"When edit fails because oldText is not unique, read the file around the target area and try again with more context",
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
			PromptSnippet: "Execute shell commands on Windows systems",
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
			PromptSnippet: "Execute shell commands on Linux/Unix systems",
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
