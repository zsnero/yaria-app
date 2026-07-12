//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

func setProcAttrDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func hideConsole(cmd *exec.Cmd) {}
