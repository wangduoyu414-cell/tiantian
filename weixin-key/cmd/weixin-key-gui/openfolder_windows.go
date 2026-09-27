//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

func openFolderImpl(dir string) {
	if dir == "" {
		return
	}
	cmd := exec.Command("explorer.exe", dir)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()
}
