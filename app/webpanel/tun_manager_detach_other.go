//go:build !unix

package webpanel

import "os/exec"

func detachFromControllingTTY(cmd *exec.Cmd) {}
