package terminal

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// probeTmuxHyperlinks checks whether tmux forwards OSC 8 hyperlinks.
// Mirrors pi's probeTmuxHyperlinks() in terminal-image.ts.
func probeTmuxHyperlinks() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "tmux", "display-message", "-p", "#{client_termfeatures}")
	cmd.Stdin = nil
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	features := strings.Split(strings.TrimSpace(string(out)), ",")
	for _, f := range features {
		if strings.TrimSpace(f) == "hyperlinks" {
			return true
		}
	}
	return false
}
