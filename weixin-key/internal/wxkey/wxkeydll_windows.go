//go:build windows

package wxkey

// wxkeydll_windows.go — Route C: delegate key extraction to wx_key.dll
//
// WeFlow (github.com/hicccc77/WeFlow) ships a pre-compiled wx_key.dll that
// hooks into the WeChat process and extracts the SQLCipher raw key. The DLL
// exports a simple C API:
//
//   InitializeHook(pid uint32) int32   -- attach hook to process
//   PollKeyData(*byte, int32) int32    -- copy key hex into caller buffer
//   CleanupHook() int32               -- detach hook
//   GetLastErrorMsg() *byte           -- last error as C string (optional)
//
// This file looks for wx_key.dll in several standard locations and, if found,
// calls those three functions to obtain the raw key hex string for the target
// WeChat process. The result is then verified against the local .db files and
// written into the wechat-cli key config exactly like the other routes.
//
// To use:
//   1. Install WeFlow on Windows (or copy wx_key.dll next to wechat-cli.exe).
//   2. Run: wechat-cli cache refresh --force
//
// wx_key.dll is looked up in this order:
//   a. WECHAT_CLI_WX_KEY_DLL env var (explicit path)
//   b. Alongside wechat-cli.exe
//   c. WeFlow's default install locations under %LOCALAPPDATA%

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// windowsWxKeyDLL holds a lazily-loaded wx_key.dll and its function pointers.
type windowsWxKeyDLL struct {
	lib            *syscall.DLL
	initializeHook *syscall.Proc
	pollKeyData    *syscall.Proc
	cleanupHook    *syscall.Proc
	getLastError   *syscall.Proc
}

func (d *windowsWxKeyDLL) release() {
	if d != nil && d.lib != nil {
		d.lib.Release()
	}
}

// getKey installs the wx_key hook on pid via InitializeHook, then polls
// PollKeyData repeatedly until the hook captures the key or the deadline
// expires. The hook only fires when WeChat actually opens a database, so the
// user must trigger a DB open (open/switch a chat, or re-login) during this
// window. CleanupHook is always called on return.
func (d *windowsWxKeyDLL) getKey(pid uint32, deadline time.Time, scan *setupScan) (string, error) {
	if err := windowsActiveCapturePreflight("wxkey-dll-call"); err != nil {
		return "", err
	}
	rc, _, _ := d.initializeHook.Call(uintptr(pid))
	if int32(rc) == 0 {
		msg := d.lastError()
		if msg == "" {
			msg = fmt.Sprintf("code %d", int32(rc))
		}
		return "", fmt.Errorf("wx_key InitializeHook: %s", msg)
	}
	defer d.cleanupHook.Call()

	buf := make([]byte, 256)
	polls := 0
	for {
		if scan.stopped(deadline) {
			return "", fmt.Errorf("wx_key: hook installed but no key captured after %d polls (open/switch a chat or re-login in WeChat while scanning)", polls)
		}
		for i := range buf {
			buf[i] = 0
		}
		rc, _, _ = d.pollKeyData.Call(
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(len(buf)),
		)
		polls++
		if int32(rc) != 0 {
			end := 0
			for end < len(buf) && buf[end] != 0 {
				end++
			}
			key := strings.TrimSpace(string(buf[:end]))
			if key != "" {
				return strings.ToLower(key), nil
			}
		}
		if err := scan.wait(300 * time.Millisecond); err != nil {
			return "", err
		}
	}
}

func (d *windowsWxKeyDLL) lastError() string {
	// GetLastErrorMsg returns a pointer into DLL-owned memory. We skip
	// dereferencing it here to stay within Go's unsafe.Pointer rules; callers
	// receive the numeric error code in the error message instead.
	return ""
}

// windowsWxKeyDLLSearchPaths returns every location Route C looks in for
// wx_key.dll, in priority order. Used both for lookup and for diagnostics.
func windowsWxKeyDLLSearchPaths() []string {
	var candidates []string
	// 1. Explicit override.
	if p := strings.TrimSpace(os.Getenv("WECHAT_CLI_WX_KEY_DLL")); p != "" {
		candidates = append(candidates, p)
	}
	// 2. Alongside wechat-cli.exe.
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "wx_key.dll"))
	}
	// 3. Current working directory.
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "wx_key.dll"))
	}
	// 4. WeFlow default install paths under %LOCALAPPDATA%.
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		candidates = append(candidates,
			filepath.Join(local, "WeFlow", "resources", "runtime", "key", "win32", "x64", "wx_key.dll"),
			filepath.Join(local, "Programs", "WeFlow", "resources", "runtime", "key", "win32", "x64", "wx_key.dll"),
		)
	}
	// 5. Home dir fallback.
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".config", "wxcli", "lib", "wx_key.dll"),
		)
	}
	return candidates
}

// windowsFindWxKeyDLL searches standard locations for wx_key.dll.
func windowsFindWxKeyDLL() string {
	for _, p := range windowsWxKeyDLLSearchPaths() {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// loadWxKeyDLLAt loads wx_key.dll from a specific path and resolves its
// exported functions. Returns (nil, err) on any failure.
func loadWxKeyDLLAt(path string) (*windowsWxKeyDLL, error) {
	lib, err := syscall.LoadDLL(path)
	if err != nil {
		return nil, fmt.Errorf("LoadDLL: %w", err)
	}
	initFn, err := lib.FindProc("InitializeHook")
	if err != nil {
		lib.Release()
		return nil, fmt.Errorf("FindProc InitializeHook: %w", err)
	}
	pollFn, err := lib.FindProc("PollKeyData")
	if err != nil {
		lib.Release()
		return nil, fmt.Errorf("FindProc PollKeyData: %w", err)
	}
	cleanFn, err := lib.FindProc("CleanupHook")
	if err != nil {
		lib.Release()
		return nil, fmt.Errorf("FindProc CleanupHook: %w", err)
	}
	d := &windowsWxKeyDLL{
		lib:            lib,
		initializeHook: initFn,
		pollKeyData:    pollFn,
		cleanupHook:    cleanFn,
	}
	if errFn, e := lib.FindProc("GetLastErrorMsg"); e == nil {
		d.getLastError = errFn
	}
	return d, nil
}

// windowsDLLScan is Route C: load wx_key.dll (if available) and use it to
// extract the key for every target WeChat process.
func windowsDLLScan(procs []windowsProcess, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) (uint32, error) {
	// Stop before LoadLibrary: even loading an unreviewed helper executes code.
	if err := windowsActiveCapturePreflight("wxkey-dll"); err != nil {
		return 0, err
	}
	path := windowsFindWxKeyDLL()
	if path == "" {
		scan.addDiag("wx_key.dll not found (searched: %s)", strings.Join(windowsWxKeyDLLSearchPaths(), "; "))
		return 0, nil
	}
	scan.addDiag("wx_key.dll found at %s", path)

	dll, loadErr := loadWxKeyDLLAt(path)
	if dll == nil {
		scan.addDiag("wx_key.dll load/FindProc failed: %v", loadErr)
		return 0, nil
	}
	defer dll.release()
	scan.addDiag("wx_key.dll loaded OK, exports resolved")

	var firstHitPID uint32

	for _, p := range procs {
		if scan.lenFound() == len(salts) {
			break
		}
		if scan.stopped(deadline) {
			return firstHitPID, errWindowsKeyScanDeadline
		}
		keyHex, err := dll.getKey(p.pid, deadline, scan)
		if err != nil {
			scan.addDiag("pid %d: %v", p.pid, err)
			continue
		}
		if keyHex == "" {
			scan.addDiag("pid %d: empty key", p.pid)
			continue
		}
		scan.addDiag("pid %d: got %d-char key, verifying against %d db(s)", p.pid, len(keyHex), len(dbs))
		before := scan.lenFound()
		for _, db := range dbs {
			if scan.hasSalt(db.salt) {
				continue
			}
			// Typed verification: the DLL may hand back either an enc_key or
			// a passphrase; persist the correct derived value either way.
			v, verr := VerifyKeyCandidate(db.path, keyHex, db.salt)
			if verr != nil {
				scan.addDiag("pid %d: verify against %s failed: %v", p.pid, db.rel, verr)
				continue
			}
			v.Source = "routeC:wx_key.dll"
			scan.noteVerified(v)
		}
		if scan.lenFound() > before {
			scan.addDiag("pid %d: key verified, matched %d salt(s)", p.pid, scan.lenFound()-before)
			if firstHitPID == 0 {
				firstHitPID = p.pid
			}
		} else {
			scan.addDiag("pid %d: key did NOT verify against any db (wrong account or stale key?)", p.pid)
		}
	}
	return firstHitPID, nil
}
