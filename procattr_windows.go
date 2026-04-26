//go:build windows

package main

import "os/exec"

func setProcAttrDetached(cmd *exec.Cmd) {}
