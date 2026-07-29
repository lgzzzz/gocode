package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// PowershellTool executes shell commands on Windows systems via PowerShell (or cmd fallback).
type PowershellTool struct {
	tracker *RollbackTracker
}

func (t *PowershellTool) Name() string                        { return "powershell" }
func (t *PowershellTool) SetTracker(tracker *RollbackTracker) { t.tracker = tracker }

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
	result, err := runShellCommand(args.Command, args.Timeout, "powershell", buildWindowsShellCmd)
	t.tracker.RecordShellCommand(args.Command, result)
	return result, err
}

func buildWindowsShellCmd(command string) *exec.Cmd {
	if _, err := exec.LookPath("powershell.exe"); err == nil {
		wrappedCmd := "[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; " + command
		return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", wrappedCmd)
	}
	return exec.Command("cmd", "/c", command)
}
