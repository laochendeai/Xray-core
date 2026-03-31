//go:build unix

package webpanel

import (
	"os/exec"
	"syscall"
)

func detachFromControllingTTY(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
