//go:build windows

package tools

import (
	"os/exec"
)

func setDetachTerminal(cmd *exec.Cmd) {
	// On Windows there is no /dev/tty concept; no-op.
}
