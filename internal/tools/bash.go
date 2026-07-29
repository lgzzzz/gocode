package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// BashTool executes shell commands on Linux/Unix systems via bash (or sh fallback).
type BashTool struct {
	tracker *RollbackTracker
}

func (t *BashTool) Name() string                        { return "bash" }
func (t *BashTool) SetTracker(tracker *RollbackTracker) { t.tracker = tracker }

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
	result, err := runShellCommand(args.Command, args.Timeout, "bash", buildLinuxShellCmd)
	t.tracker.RecordShellCommand(args.Command, result)
	return result, err
}

func buildLinuxShellCmd(command string) *exec.Cmd {
	var cmd *exec.Cmd
	if _, err := exec.LookPath("bash"); err == nil {
		cmd = exec.Command("bash", "-c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	setDetachTerminal(cmd)
	return cmd
}
