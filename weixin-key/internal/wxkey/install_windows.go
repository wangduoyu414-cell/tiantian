//go:build windows

package wxkey

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

type installDiscovery struct {
	processes func() ([]WeChatProcess, error)
	roots     func() ([]string, error)
}

func defaultInstallDiscovery() installDiscovery {
	return installDiscovery{processes: FindWeChatProcesses, roots: windowsInstallRoots}
}

func queryProcessExecutable(pid uint32) (string, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", processAccessError(pid, windows.PROCESS_QUERY_LIMITED_INFORMATION, err)
	}
	defer windows.CloseHandle(h)
	var buf [32768]uint16
	n := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buf[:n]), nil
}

// No disk-wide search, registry writes, process-memory reads or execution.
func windowsInstallRoots() ([]string, error) {
	var roots []string
	for _, entry := range []struct {
		h           registry.Key
		path, value string
	}{
		{registry.CURRENT_USER, `Software\Tencent\Weixin`, "InstallPath"},
		{registry.LOCAL_MACHINE, `SOFTWARE\Tencent\Weixin`, "InstallPath"},
		{registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Tencent\Weixin`, "InstallPath"},
		{registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\App Paths\Weixin.exe`, ""},
		{registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\Weixin.exe`, ""},
	} {
		key, err := registry.OpenKey(entry.h, entry.path, registry.QUERY_VALUE)
		if errors.Is(err, registry.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read WeChat installation registry: %w", err)
		}
		value, _, err := key.GetStringValue(entry.value)
		key.Close()
		if errors.Is(err, registry.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		if strings.EqualFold(filepath.Ext(value), ".exe") {
			value = filepath.Dir(value)
		}
		if value != "" {
			roots = append(roots, value)
		}
	}
	for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
		if base != "" {
			roots = append(roots, filepath.Join(base, "Tencent", "Weixin"))
		}
	}
	return roots, nil
}

func existingInstallFile(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("installation file is not regular: %s", abs)
	}
	return filepath.EvalSymlinks(abs)
}

// oneInstallFile accepts a flat install, otherwise the newest numeric version
// directory in ONE install root. It never recursively traverses data folders.
func oneInstallFile(root, name string) (string, error) {
	flat, err := existingInstallFile(filepath.Join(root, name))
	if err == nil {
		return flat, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	f, err := os.Open(root)
	if err != nil {
		return "", err
	}
	defer f.Close()
	entries, err := f.ReadDir(257)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if len(entries) > 256 {
		return "", errors.New("installation root has too many entries; supply an explicit executable/module")
	}
	sort.Slice(entries, func(i, j int) bool {
		a, _ := parseDottedVersion(entries[i].Name())
		b, _ := parseDottedVersion(entries[j].Name())
		return compareVersions(a, b) > 0
	})
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, ok := parseDottedVersion(e.Name()); !ok {
			continue
		}
		path, err := existingInstallFile(filepath.Join(root, e.Name(), name))
		if err == nil {
			return path, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", os.ErrNotExist
}

func uniqueInstall(paths []string) (string, error) {
	var found []string
	for _, p := range paths {
		duplicate := false
		for _, existing := range found {
			a, ea := os.Stat(p)
			b, eb := os.Stat(existing)
			if strings.EqualFold(existing, p) || (ea == nil && eb == nil && os.SameFile(a, b)) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			found = append(found, p)
		}
	}
	if len(found) == 1 {
		return found[0], nil
	}
	if len(found) > 1 {
		return "", errors.New("multiple WeChat installations; select the intended executable explicitly")
	}
	return "", errors.New("no usable WeChat installation found; supply the actual executable path")
}

func discoverWeixinExe(explicit string, deps installDiscovery) (string, error) {
	if explicit != "" {
		return existingInstallFile(explicit)
	}
	procs, err := deps.processes()
	if err != nil {
		return "", err
	}
	if len(procs) > 0 {
		var paths []string
		for _, p := range procs {
			if p.ExePath == "" {
				return "", errors.New("running WeChat executable identity unavailable; refusing to guess")
			}
			path, err := existingInstallFile(p.ExePath)
			if err != nil {
				return "", err
			}
			paths = append(paths, path)
		}
		return uniqueInstall(paths)
	}
	roots, err := deps.roots()
	if err != nil {
		return "", err
	}
	var paths []string
	for _, root := range roots {
		path, err := oneInstallFile(root, "Weixin.exe")
		if err == nil {
			paths = append(paths, path)
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return uniqueInstall(paths)
}

func DefaultWeixinExePath() (string, error) {
	return discoverWeixinExe(firstEnv("WECHAT_CLI_WECHAT_EXE", "WX_MCP_WECHAT_EXE"), defaultInstallDiscovery())
}

func defaultWeixinDLLPath() (string, error) {
	exe, err := DefaultWeixinExePath()
	if err != nil {
		return "", err
	}
	return oneInstallFile(filepath.Dir(exe), "Weixin.dll")
}
