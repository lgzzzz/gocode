//go:build !windows

package tools

import (
	"os/exec"
	"syscall"
)

func setDetachTerminal(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
