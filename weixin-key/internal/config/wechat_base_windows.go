//go:build windows

package config

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

func DefaultWeChatBases() ([]string, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	var bases []string
	// WeChat 4.x default: <Documents>\xwechat_files\<account>\db_storage
	// WeChat 3.x legacy:  <Documents>\WeChat Files\<account>\...
	if profile := os.Getenv("USERPROFILE"); profile != "" {
		for _, root := range []string{
			filepath.Join(profile, "Documents", "xwechat_files"),
			filepath.Join(profile, "xwechat_files"),
			filepath.Join(profile, "Documents", "WeChat Files"),
			filepath.Join(profile, "WeChat Files"),
			filepath.Join(profile, "AppData", "Roaming", "Tencent", "WeChat", "WeChat Files"),
		} {
			bases = append(bases, withXWeChatFilesBase(root)...)
		}
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		bases = append(bases, withXWeChatFilesBase(filepath.Join(appData, "Tencent", "WeChat", "WeChat Files"))...)
	}
	for _, root := range []string{
		filepath.Join(h, "Documents", "xwechat_files"),
		filepath.Join(h, "xwechat_files"),
		filepath.Join(h, "Documents", "WeChat Files"),
		filepath.Join(h, "WeChat Files"),
	} {
		bases = append(bases, withXWeChatFilesBase(root)...)
	}
	// Custom install/data drives: xwechat_files may live at a drive root or
	// one directory level below it (e.g. E:\xwechat_files, E:\软件\xwechat_files).
	bases = append(bases, FixedDriveWeChatBases(listFixedDrives(), readSubdirNames)...)
	return uniquePaths(bases), nil
}

// FixedDriveWeChatBases returns xwechat_files candidates under each fixed
// drive root and one directory level down. Bounded: drive roots + their
// immediate subdirectories only.
func FixedDriveWeChatBases(drives []string, readSubdirs func(string) []string) []string {
	var out []string
	for _, d := range drives {
		d = strings.TrimRight(d, `\/`)
		if d == "" {
			continue
		}
		out = append(out, d+`\xwechat_files`)
		for _, sub := range readSubdirs(d) {
			if strings.EqualFold(sub, "xwechat_files") {
				continue // already covered at the root
			}
			out = append(out, d+`\`+sub+`\xwechat_files`)
		}
	}
	return out
}

// listFixedDrives returns fixed local drive roots (C:\, D:\, ...).
func listFixedDrives() []string {
	var out []string
	for _, letter := range "CDEFGHIJKLMNOPQRSTUVWXYZ" {
		root := string(letter) + `:\`
		driveType := getDriveType(root)
		if driveType == 3 { // DRIVE_FIXED
			out = append(out, root)
		}
	}
	return out
}

// readSubdirNames returns immediate subdirectory names of dir (empty on error).
func readSubdirNames(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

func uniquePaths(paths []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p == "" {
			continue
		}
		clean := filepath.Clean(p)
		key := filepath.ToSlash(clean)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, clean)
	}
	return out
}

var (
	kernel32DLLForDrives = syscall.NewLazyDLL("kernel32.dll")
	procGetDriveTypeW    = kernel32DLLForDrives.NewProc("GetDriveTypeW")
)

// getDriveType wraps GetDriveTypeW (3 = DRIVE_FIXED).
func getDriveType(root string) uint32 {
	p, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return 0
	}
	r, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(p)))
	return uint32(r)
}
