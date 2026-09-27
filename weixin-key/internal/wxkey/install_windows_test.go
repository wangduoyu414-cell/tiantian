//go:build windows

package wxkey

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func syntheticInstallFile(t *testing.T, root, name string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("discovery fixture, never executed"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInstallDiscoveryPrefersRunningIdentityToStaleRegistry(t *testing.T) {
	path := syntheticInstallFile(t, t.TempDir(), "Weixin.exe")
	deps := installDiscovery{
		processes: func() ([]WeChatProcess, error) {
			return []WeChatProcess{{PID: 1, ExePath: path}, {PID: 2, ExePath: path}}, nil
		},
		roots: func() ([]string, error) { t.Fatal("running identity must not fall back to registry"); return nil, nil },
	}
	got, err := discoverWeixinExe("", deps)
	if err != nil || got != path {
		t.Fatalf("discovery=%s err=%v", got, err)
	}
}
func TestInstallDiscoveryColdStartAndNumericVersions(t *testing.T) {
	root := t.TempDir()
	syntheticInstallFile(t, filepath.Join(root, "4.1.9.0"), "Weixin.exe")
	want := syntheticInstallFile(t, filepath.Join(root, "4.1.13.65"), "Weixin.exe")
	deps := installDiscovery{
		processes: func() ([]WeChatProcess, error) { return nil, nil },
		roots:     func() ([]string, error) { return []string{filepath.Join(t.TempDir(), "stale"), root}, nil },
	}
	got, err := discoverWeixinExe("", deps)
	if err != nil || got != want {
		t.Fatalf("numeric versions=%s err=%v", got, err)
	}
	flat := syntheticInstallFile(t, root, "Weixin.exe")
	got, err = discoverWeixinExe("", deps)
	if err != nil || got != flat {
		t.Fatal("flat launcher not supported", err)
	}
}
func TestInstallDiscoveryRefusesAmbiguityAndPermissionFallback(t *testing.T) {
	a, b := syntheticInstallFile(t, t.TempDir(), "Weixin.exe"), syntheticInstallFile(t, t.TempDir(), "Weixin.exe")
	deps := installDiscovery{processes: func() ([]WeChatProcess, error) { return []WeChatProcess{{ExePath: a}, {ExePath: b}}, nil }}
	if _, err := discoverWeixinExe("", deps); err == nil {
		t.Fatal("multiple running installations silently guessed")
	}
	denied := errors.New("query permission denied")
	deps.processes = func() ([]WeChatProcess, error) { return nil, denied }
	if _, err := discoverWeixinExe("", deps); !errors.Is(err, denied) {
		t.Fatal("process query error hidden by fallback")
	}
	got, err := discoverWeixinExe(a, deps)
	if err != nil || got != a {
		t.Fatal("explicit selection ignored", err)
	}
	if _, err := discoverWeixinExe(filepath.Join(t.TempDir(), "missing.exe"), deps); err == nil {
		t.Fatal("invalid explicit path accepted")
	}
}
func TestInstallVersionOverflowRejected(t *testing.T) {
	if _, ok := parseDottedVersion("4." + strings.Repeat("9", 40)); ok {
		t.Fatal("overflowed version accepted")
	}
}
func TestBarePIDCloseRefusedWithoutProcessOperations(t *testing.T) {
	if err := RequestCloseWeChat(0); err == nil {
		t.Fatal("unassociated PID close accepted")
	}
}
