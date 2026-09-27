//go:build windows

package wxkey

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestAccountFileOwnerHelper(t *testing.T) {
	path := os.Getenv("WEIXIN_OWNER_TEST_FILE")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fmt.Fprintln(os.Stdout, "owner-ready")
	io.Copy(io.Discard, os.Stdin)
}

func startSyntheticFileOwner(t *testing.T, path string) uint32 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAccountFileOwnerHelper$")
	cmd.Env = append(os.Environ(), "WEIXIN_OWNER_TEST_FILE="+path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() { stdin.Close(); cmd.Wait(); cancel() })
	sc := bufio.NewScanner(stdout)
	if !sc.Scan() || sc.Text() != "owner-ready" {
		t.Fatal("synthetic file-owner helper did not become ready")
	}
	return uint32(cmd.Process.Pid)
}

func TestRestartManagerSelectedFileAndPIDIdentity(t *testing.T) {
	// ABI must match RM_UNIQUE_PROCESS and RM_PROCESS_INFO on Windows.
	if unsafe.Sizeof(accountProcess{}) != 12 || unsafe.Sizeof(rmProcessInfo{}) != 668 {
		t.Fatal("Restart Manager structure layout mismatch")
	}
	dir := t.TempDir()
	selected, unrelated := filepath.Join(dir, "selected.db"), filepath.Join(dir, "unrelated.db")
	for _, path := range []string{selected, unrelated} {
		if err := os.WriteFile(path, []byte("synthetic"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	pid := startSyntheticFileOwner(t, selected)
	other := startSyntheticFileOwner(t, unrelated)
	owners, err := accountFileOwners([]windowsSourceDB{{path: selected}})
	if err != nil {
		t.Fatal(err)
	}
	var target *accountProcess
	for i := range owners {
		if owners[i].pid == other {
			t.Fatal("selected file was associated with the unrelated process")
		}
		if owners[i].pid == pid {
			target = &owners[i]
		}
	}
	if target == nil {
		t.Fatalf("Restart Manager did not return the synthetic selected-file owner (count=%d)", len(owners))
	}
	h, path, err := pinAccountProcess(*target)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(h)
	if filepath.Base(path) != filepath.Base(os.Args[0]) {
		t.Fatal("pinned process path differs from our own test executable")
	}
	stale := *target
	stale.start.LowDateTime ^= 1
	if unexpected, _, err := pinAccountProcess(stale); err == nil {
		windows.CloseHandle(unexpected)
		t.Fatal("stale process creation time was accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitPinnedProcess(ctx, h); err != context.Canceled {
		t.Fatalf("pinned-process cancellation: %v", err)
	}
}
