//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

func setProcAttrDetached(cmd *exec.Cmd) {
	// Detach is a no-op on Windows; still hide console for child tools.
	hideConsole(cmd)
}

// hideConsole prevents console windows from flashing for child .exe tools
// (yt-dlp, ffmpeg, powershell, etc.) when launched from a GUI app.
func hideConsole(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags |= createNoWindow
}
