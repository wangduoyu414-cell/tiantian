//go:build !windows && !darwin

package main

import "os/exec"

func openFolderImpl(dir string) {
	if dir == "" {
		return
	}
	_ = exec.Command("xdg-open", dir).Start()
}
