//go:build darwin

package main

import (
	"os/exec"
	"strings"
)

// pickDirectory uses macOS's native folder picker through osascript.
func pickDirectory(initial string) (string, error) {
	out, err := exec.Command("osascript", "-e", `POSIX path of (choose folder with prompt "选择导出目录")`).Output()
	if err != nil {
		return "", nil // user cancelled
	}
	return strings.TrimSpace(string(out)), nil
}
