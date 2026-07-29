package tools

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// runShellCommand executes a shell command with the given timeout, using the provided
// buildCmd factory to construct the platform-appropriate exec.Cmd.
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
			return result, err
		}
		if result == "" {
			result = "(no output)"
		}
		return result, nil
	case <-time.After(time.Duration(timeout) * time.Second):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return out.String(), fmt.Errorf("timed out after %ds", timeout)
	}
}
