//go:build windows

package wxkey

import (
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// The readiness probe must find a needle that really is in this process's
// writable heap (self-scan via OpenProcess on our own PID).
func TestWindowsProcessContainsAnyFindsNeedleInSelf(t *testing.T) {
	needle := []byte("wxkey-watch-needle-7f3a9c")
	keep := make([]byte, 0, 4096)
	keep = append(keep, []byte("padding-")...)
	keep = append(keep, needle...)
	keep = append(keep, []byte("-suffix")...)
	runtime.KeepAlive(keep)

	ok := windowsProcessContainsAny(uint32(os.Getpid()), [][]byte{needle}, 30*time.Second)
	runtime.KeepAlive(keep)
	if !ok {
		t.Fatal("needle present in self heap was not found")
	}
}

// An absent needle must not match. We scan a CHILD process (never our own:
// the test needle itself would sit in our heap). The per-process budget is
// honored even for a running child.
func TestWindowsProcessContainsAnyMissAndBudget(t *testing.T) {
	absent := []byte("wxkey-watch-absent-91b27d04")
	cmd := exec.Command("powershell.exe", "-NoProfile", "-Command", "Start-Sleep -Seconds 60")
	cmd.Dir = t.TempDir()
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot spawn child: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill() // only this owned inert test helper
		_ = cmd.Wait()
	}()

	if windowsProcessContainsAny(uint32(cmd.Process.Pid), [][]byte{absent}, 5*time.Second) {
		t.Fatal("absent needle matched in child process")
	}
	start := time.Now()
	windowsProcessContainsAny(uint32(cmd.Process.Pid), [][]byte{absent}, 300*time.Millisecond)
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("budget not honored: took %s", elapsed)
	}
}

// WindowsWaitForDBOpen must reject garbage salts and not hang when nothing is
// running.
func TestWindowsWaitForDBOpenBadInput(t *testing.T) {
	if _, err := WindowsWaitForDBOpen([]string{"zzzz"}, time.Second, time.Second, nil); err == nil {
		t.Fatal("invalid salt accepted")
	}
}
