//go:build darwin

package main

import "os/exec"

func openFolderImpl(dir string) {
	if dir == "" {
		return
	}
	_ = exec.Command("open", dir).Start()
}
