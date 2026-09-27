//go:build windows

package wxkey

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsReadNativeCaptureFileFiltersAndDeduplicates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capture.txt")
	keyA := strings.Repeat("01", 32)
	keyB := strings.Repeat("ab", 32)
	payload := "noise\n" + keyA + "\n" + strings.ToUpper(keyA) + "\n" + keyB + "\n"
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := windowsReadNativeCaptureFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != keyA || got[1] != keyB {
		t.Fatalf("candidates = %#v", got)
	}
}

func TestWindowsReadNativeCaptureFileRejectsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capture.txt")
	if err := os.WriteFile(path, []byte("not-a-key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := windowsReadNativeCaptureFile(path); err == nil {
		t.Fatal("expected invalid capture file error")
	}
}

func TestWindowsReadNativeCaptureFileRequiresOneCandidatePerLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capture.txt")
	keyA := strings.Repeat("01", 32)
	keyB := strings.Repeat("ab", 32)
	if err := os.WriteFile(path, []byte(keyA+" "+keyB+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := windowsReadNativeCaptureFile(path); err == nil {
		t.Fatal("expected same-line candidates to be rejected")
	}
}

func TestWindowsReadNativeCaptureFileRejectsOversize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capture.txt")
	if err := os.WriteFile(path, make([]byte, maxNativeCaptureFileSize+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := windowsReadNativeCaptureFile(path); err == nil {
		t.Fatal("expected oversized capture file error")
	}
}

func TestWindowsReadNativeCaptureFileRejectsDirectory(t *testing.T) {
	if _, err := windowsReadNativeCaptureFile(t.TempDir()); err == nil {
		t.Fatal("expected directory capture path error")
	}
}

func TestWindowsReadNativeCaptureFileRejectsTooManyCandidates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capture.txt")
	var payload strings.Builder
	for i := 0; i <= maxNativeCaptureCandidates; i++ {
		payload.WriteString(strings.Repeat(fmt.Sprintf("%02x", i), 32))
		payload.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(payload.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := windowsReadNativeCaptureFile(path); err == nil {
		t.Fatal("expected candidate count limit error")
	}
}

func TestWindowsNativeCapturePID(t *testing.T) {
	t.Setenv("WECHAT_CLI_CAPTURE_PID", "12345")
	pid, err := windowsNativeCapturePID()
	if err != nil || pid != 12345 {
		t.Fatalf("pid=%d err=%v", pid, err)
	}
	t.Setenv("WECHAT_CLI_CAPTURE_PID", "invalid")
	if _, err := windowsNativeCapturePID(); err == nil {
		t.Fatal("expected invalid PID error")
	}
}
