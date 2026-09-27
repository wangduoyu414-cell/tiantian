//go:build windows

package wxkey

import (
	"context"
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

// processAccessError retains the actual Win32 error, rather than inferring
// protection/permissions from a generic "OpenProcess failed" log message.
func processAccessError(pid uint32, access uint32, err error) error {
	var code syscall.Errno
	errors.As(err, &code)
	return fmt.Errorf("OpenProcess pid=%d requested_access=0x%x win32=%d: %w", pid, access, uint32(code), err)
}

func isProcessAccessDenied(err error) bool {
	return errors.Is(err, syscall.ERROR_ACCESS_DENIED)
}

type accountProcess struct {
	pid   uint32
	start windows.Filetime
}

type rmProcessInfo struct {
	Process     accountProcess
	AppName     [256]uint16
	ServiceName [64]uint16
	AppType     uint32
	AppStatus   uint32
	SessionID   uint32
	Restartable int32
}

// accountFileOwners asks Windows Restart Manager for processes holding this
// account's DB files. It never reads process memory or sends RmShutdown.
// Creation time is part of the identity, so a recycled PID cannot be closed.
func accountFileOwners(dbs []windowsSourceDB) ([]accountProcess, error) {
	dll := windows.NewLazySystemDLL("rstrtmgr.dll")
	start := dll.NewProc("RmStartSession")
	register := dll.NewProc("RmRegisterResources")
	getList := dll.NewProc("RmGetList")
	end := dll.NewProc("RmEndSession")
	var session uint32
	var key [33]uint16
	rc, _, _ := start.Call(uintptr(unsafe.Pointer(&session)), 0, uintptr(unsafe.Pointer(&key[0])))
	if rc != 0 {
		return nil, fmt.Errorf("RmStartSession: win32=%d", rc)
	}
	defer end.Call(uintptr(session))
	files := make([]*uint16, 0, len(dbs))
	for _, db := range dbs {
		p, err := windows.UTF16PtrFromString(db.path)
		if err != nil {
			return nil, err
		}
		files = append(files, p)
	}
	if len(files) == 0 {
		return nil, errors.New("no selected-account files to associate")
	}
	rc, _, _ = register.Call(uintptr(session), uintptr(len(files)), uintptr(unsafe.Pointer(&files[0])), 0, 0, 0, 0)
	runtime.KeepAlive(files)
	if rc != 0 {
		return nil, fmt.Errorf("RmRegisterResources: win32=%d", rc)
	}
	for attempts := 0; attempts < 3; attempts++ {
		var needed, count, reasons uint32
		rc, _, _ = getList.Call(uintptr(session), uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&count)), 0, uintptr(unsafe.Pointer(&reasons)))
		if rc == 0 {
			return nil, nil
		}
		if rc != uintptr(windows.ERROR_MORE_DATA) || needed > 4096 {
			return nil, fmt.Errorf("RmGetList: win32=%d count=%d", rc, needed)
		}
		infos := make([]rmProcessInfo, needed)
		count = needed
		rc, _, _ = getList.Call(uintptr(session), uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&infos[0])), uintptr(unsafe.Pointer(&reasons)))
		if rc == uintptr(windows.ERROR_MORE_DATA) {
			continue
		}
		if rc != 0 {
			return nil, fmt.Errorf("RmGetList: win32=%d", rc)
		}
		var out []accountProcess
		for _, info := range infos[:count] {
			out = append(out, info.Process)
		}
		return out, nil
	}
	return nil, errors.New("account file owners changed repeatedly; retry after activity settles")
}

func pinAccountProcess(p accountProcess) (windows.Handle, string, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, p.pid)
	if err != nil {
		return 0, "", processAccessError(p.pid, windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, err)
	}
	var created, exited, kernel, user windows.Filetime
	if err = windows.GetProcessTimes(h, &created, &exited, &kernel, &user); err != nil || created != p.start {
		windows.CloseHandle(h)
		return 0, "", errors.New("account process exited or PID identity changed")
	}
	var buf [32768]uint16
	n := uint32(len(buf))
	if err = windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		windows.CloseHandle(h)
		return 0, "", err
	}
	return h, windows.UTF16ToString(buf[:n]), nil
}

func waitPinnedProcess(ctx context.Context, h windows.Handle) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		event, err := windows.WaitForSingleObject(h, 200)
		if err != nil {
			return err
		}
		if event == windows.WAIT_OBJECT_0 {
			return nil
		}
		if event != uint32(windows.WAIT_TIMEOUT) {
			return fmt.Errorf("process wait returned %d", event)
		}
	}
}

// requestWindowClose sends WM_CLOSE only to top-level windows of one pinned
// process. No process-name kill, /T child-tree termination, or force-kill.
func requestWindowClose(pid uint32) error {
	user32 := windows.NewLazySystemDLL("user32.dll")
	enum := user32.NewProc("EnumWindows")
	getPID := user32.NewProc("GetWindowThreadProcessId")
	post := user32.NewProc("PostMessageW")
	count := 0
	var closeErr error
	cb := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		var windowPID uint32
		getPID.Call(hwnd, uintptr(unsafe.Pointer(&windowPID)))
		if windowPID == pid {
			if ok, _, err := post.Call(hwnd, 0x0010, 0, 0); ok == 0 {
				closeErr = fmt.Errorf("WM_CLOSE pid=%d: %w", pid, err)
			} else {
				count++
			}
		}
		return 1
	})
	if ok, _, err := enum.Call(cb, 0); ok == 0 {
		return fmt.Errorf("EnumWindows: %w", err)
	}
	if closeErr != nil {
		return closeErr
	}
	if count == 0 {
		return errors.New("target account has no closable window; close it manually")
	}
	return nil
}

// A retained handle prevents PID recycling. All controls use this identity;
// filenames alone never authorize a target, including for non-restart routes.
type accountControlTarget struct {
	identity accountProcess
	exe      string
	alive    func() error
	close    func() error
	wait     func(context.Context) error
	release  func()
}

func pinControlTarget(p accountProcess) (*accountControlTarget, error) {
	h, path, err := pinAccountProcess(p)
	if err != nil {
		return nil, err
	}
	t := &accountControlTarget{identity: p, exe: path}
	t.alive = func() error {
		state, err := windows.WaitForSingleObject(h, 0)
		if err != nil {
			return err
		}
		if state != uint32(windows.WAIT_TIMEOUT) {
			return errors.New("selected account process exited; discard its capture session")
		}
		return nil
	}
	t.close = func() error {
		if err := t.alive(); err != nil {
			return err
		}
		return requestWindowClose(p.pid)
	}
	t.wait = func(ctx context.Context) error { return waitPinnedProcess(ctx, h) }
	t.release = func() { windows.CloseHandle(h) }
	return t, nil
}

func releaseAccountTargets(targets []*accountControlTarget) {
	for _, t := range targets {
		t.release()
	}
}

func selectAccountTargets(ctx context.Context, procs []windowsProcess, dbs []windowsSourceDB, requireAll bool,
	ownersFn func([]windowsSourceDB) ([]accountProcess, error),
	pin func(accountProcess) (*accountControlTarget, error)) (targets []*accountControlTarget, err error) {
	defer func() {
		if err != nil {
			releaseAccountTargets(targets)
			targets = nil
		}
	}()
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	if len(procs) == 0 {
		return nil, nil
	}
	owners, err := ownersFn(dbs)
	if err != nil {
		return nil, fmt.Errorf("cannot safely associate account with process: %w", err)
	}
	allowed := map[uint32]bool{}
	for _, p := range procs {
		allowed[p.pid] = true
	}
	seen := map[uint32]accountProcess{}
	for _, owner := range owners {
		if !allowed[owner.pid] {
			continue // unrelated file owners are never opened or controlled
		}
		if prev, ok := seen[owner.pid]; ok {
			if prev != owner {
				return targets, errors.New("conflicting account process identity")
			}
			continue
		}
		if err = ctx.Err(); err != nil {
			return targets, err
		}
		target, pinErr := pin(owner)
		if pinErr != nil {
			return targets, pinErr
		}
		targets = append(targets, target)
		seen[owner.pid] = owner
	}
	if len(targets) == 0 {
		return targets, errors.New("no running WeChat process is associated with the selected account; no process will be scanned or controlled")
	}
	if requireAll && len(seen) != len(allowed) {
		return targets, errors.New("other or unassociated WeChat processes remain; refusing restart before closing any window")
	}
	for _, target := range targets {
		if err = target.alive(); err != nil {
			return targets, err
		}
	}
	return targets, nil
}

type accountRestartDeps struct {
	processes func() ([]windowsProcess, error)
	owners    func([]windowsSourceDB) ([]accountProcess, error)
	pin       func(accountProcess) (*accountControlTarget, error)
	exe       func() (string, error)
	prepare   func(string) error
	launch    func(string, []windowsSourceDB, map[string]bool, *setupScan, time.Time) (uint32, error)
}

func windowsRestartAccount(ctx context.Context, opts SetupOptions, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan) (uint32, error) {
	// Must be before even the first WM_CLOSE, not merely in windowsLaunchCapture.
	if err := windowsActiveCapturePreflight("account-restart"); err != nil {
		return 0, err
	}
	// prepare → every WM_CLOSE → launch all share the same debugger owner.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	return restartAccountWithDeps(ctx, opts, dbs, salts, scan, defaultAccountRestartDeps(windowsEnumerateProcesses, opts.Recovery))
}

func defaultAccountRestartDeps(enumerate func() ([]windowsProcess, error), recovery *CaptureRecovery) accountRestartDeps {
	var ready *debugOwnerReady
	return accountRestartDeps{
		processes: func() ([]windowsProcess, error) { return allWeChatProcesses(enumerate) },
		owners:    accountFileOwners, pin: pinControlTarget,
		exe: DefaultWeixinExePath,
		// The gate still precedes helper creation and any account side effects.
		prepare: func(exe string) error {
			if err := windowsActiveCapturePreflight(exe); err != nil {
				return err
			}
			var err error
			ready, err = prepareDebuggerOwner(recovery)
			return err
		},
		launch: func(exe string, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) (uint32, error) {
			return windowsLaunchCaptureOnOwner(ready, exe, dbs, salts, scan, deadline)
		},
	}
}

// Phase one gathers every identity and capture prerequisite before any side
// effect. Phase two can still fail part-way through normal user-app shutdown;
// this is not an atomic multi-process close and never uses force-kill.
func restartAccountWithDeps(ctx context.Context, opts SetupOptions, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deps accountRestartDeps) (uint32, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	procs, err := deps.processes()
	if err != nil {
		return 0, err
	}
	targets, err := selectAccountTargets(ctx, procs, dbs, true, deps.owners, deps.pin)
	if err != nil {
		return 0, err
	}
	defer releaseAccountTargets(targets)
	exe := opts.ExePath
	for _, target := range targets {
		if exe != "" && !strings.EqualFold(filepath.Clean(exe), filepath.Clean(target.exe)) {
			return 0, errors.New("selected executable differs from an account process; no windows closed")
		}
		exe = target.exe
	}
	if exe == "" {
		exe, err = deps.exe()
		if err != nil {
			return 0, err
		}
	}
	if err := deps.prepare(exe); err != nil {
		return 0, err
	}
	// Repeat only the cheap metadata check after potentially expensive static
	// preparation, before closing anything. Retained handles prevent PID reuse.
	current, err := deps.processes()
	if err != nil {
		return 0, err
	}
	pinned := map[uint32]bool{}
	for _, target := range targets {
		pinned[target.identity.pid] = true
		if err := target.alive(); err != nil {
			return 0, err
		}
	}
	if len(current) != len(pinned) {
		return 0, errors.New("WeChat process set changed before shutdown; no windows closed")
	}
	for _, p := range current {
		if !pinned[p.pid] {
			return 0, errors.New("unassociated WeChat appeared before shutdown; no windows closed")
		}
	}
	for _, target := range targets {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if opts.Progress != nil {
			opts.Progress(fmt.Sprintf("请求所选账号的微信正常关闭：pid=%d，%s", target.identity.pid, target.exe))
		}
		if err := target.close(); err != nil {
			return 0, fmt.Errorf("normal close failed; no force-kill performed: %w", err)
		}
		closeCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		err = target.wait(closeCtx)
		cancel()
		if err != nil {
			return 0, fmt.Errorf("normal exit not confirmed; no force-kill performed: %w", err)
		}
	}
	remaining, err := deps.processes()
	if err != nil {
		return 0, err
	}
	if len(remaining) > 0 {
		return 0, errors.New("WeChat processes remain after normal close; not launching into an unknown session")
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if opts.Progress != nil {
		opts.Progress("准备启动期观察；请在微信窗口中登录所选账号（当前并未确认登录成功）")
	}
	deadline, _ := ctx.Deadline()
	return deps.launch(exe, dbs, salts, scan, deadline)
}
