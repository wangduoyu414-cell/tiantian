//go:build windows

package wxkey

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const debugOnlyThisProcess = 0x00000002

// windowsRestartCaptureEnabled allows the caller to opt into restarting
// WeChat under the debugger so the first database-key call cannot be missed.
func windowsRestartCaptureEnabled() bool {
	switch strings.ToLower(firstEnv("WECHAT_CLI_KEY_RESTART", "WX_MCP_KEY_RESTART")) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func windowsLaunchCapture(exePath string, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) (pid uint32, err error) {
	if err := windowsActiveCapturePreflight("restart-launch"); err != nil {
		return 0, err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ready, err := prepareDebuggerOwner(scan.recovery)
	if err != nil {
		return 0, err
	}
	return windowsLaunchCaptureOnOwner(ready, exePath, dbs, salts, scan, deadline)
}

func windowsLaunchCaptureOnOwner(ready *debugOwnerReady, exePath string, dbs []windowsSourceDB,
	salts map[string]bool, scan *setupScan, deadline time.Time) (pid uint32, err error) {
	if err := windowsActiveCapturePreflight("prepared-restart-launch"); err != nil {
		return 0, err
	}
	if err := ready.check(); err != nil {
		return 0, err
	}
	if scan.stopped(deadline) {
		return 0, errors.Join(scan.ctx.Err(), errWindowsKeyScanDeadline)
	}
	app, err := windows.UTF16PtrFromString(exePath)
	if err != nil {
		return 0, err
	}
	args, err := windows.UTF16PtrFromString(syscall.EscapeArg(exePath) + " --scene=desktop")
	if err != nil {
		return 0, err
	}
	dir, err := windows.UTF16PtrFromString(filepath.Dir(exePath))
	if err != nil {
		return 0, err
	}
	si := windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{}))}
	var pi windows.ProcessInformation
	if err := windows.CreateProcess(app, args, nil, nil, false, debugOnlyThisProcess|windows.CREATE_NO_WINDOW, nil, dir, &si, &pi); err != nil {
		return 0, fmt.Errorf("restart capture: launch WeChat: %w", err)
	}
	pid = pi.ProcessId
	rh := uintptr(pi.Process)
	// PROCESS_INFORMATION handles are ours, unlike DEBUG_EVENT handles.
	// Keep the exact process object alive across failed detach/recovery.
	defer func() { err = errors.Join(err, windows.CloseHandle(pi.Process), windows.CloseHandle(pi.Thread)) }()
	session := newDebugSession(nativeDebugAPI{process: rh}, pid, 0)
	if scan.recovery != nil {
		session.recovery = scan.recovery
	}
	defer func() {
		if !session.cleaned {
			err = errors.Join(err, session.finishCleanup())
		}
	}()
	scan.addDiag("RouteB-restart: launched pid %d under debugger", pid)
	return pid, windowsCaptureDebugLoop(session, rh, dbs, salts, scan, deadline, func() (uintptr, error) {
		// Legacy discovery only; NOT a verified instruction/material profile.
		if start, end := windowsFindWeChatCoreRange(pid); start != 0 && end != 0 {
			strAddr := windowsFindBytesInRange(rh, start, end, []byte("x'%s'\x00"))
			if strAddr != 0 {
				return windowsFindCodeRefTo(rh, start, end, strAddr), nil
			}
		}
		return 0, nil
	})
}
func windowsProcessExecutablePath(pid uint32) string {
	snap, _, _ := procCreateToolhelp32.Call(th32csSnapModule, uintptr(pid))
	if snap == uintptr(syscall.InvalidHandle) || snap == 0 {
		return ""
	}
	defer procCloseHandle.Call(snap)
	var entry windowsModuleEntry32W
	entry.Size = uint32(unsafe.Sizeof(entry))
	r, _, _ := procModule32FirstW.Call(snap, uintptr(unsafe.Pointer(&entry)))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(entry.ExePath[:])
}

func windowsAnyWeChatProcessExists() bool {
	procs, err := windowsEnumerateProcesses()
	if err != nil {
		return false
	}
	for _, p := range procs {
		name := strings.ToLower(p.exe)
		if name == "weixin.exe" || name == "wechat.exe" {
			return true
		}
	}
	return false
}
