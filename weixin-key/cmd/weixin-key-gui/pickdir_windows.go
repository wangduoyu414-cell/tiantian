//go:build windows

package main

import (
	"os/exec"
	"strings"
	"syscall"
)

// pickDirectory opens the native folder browser via PowerShell WinForms
// (hidden console, no new dependencies). Cancel returns ("", nil).
func pickDirectory(initial string) (string, error) {
	esc := strings.ReplaceAll(initial, "'", "''")
	ps := `Add-Type -AssemblyName System.Windows.Forms; ` +
		`$d = New-Object System.Windows.Forms.FolderBrowserDialog; ` +
		`$d.Description = '选择导出目录'; ` +
		`if ('` + esc + `' -and (Test-Path '` + esc + `')) { $d.SelectedPath = '` + esc + `' }; ` +
		`if ($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { $d.SelectedPath }`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-STA", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
