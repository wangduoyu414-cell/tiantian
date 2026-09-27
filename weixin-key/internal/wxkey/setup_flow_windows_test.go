//go:build windows

package wxkey

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"weixin-key/internal/config"
	"weixin-key/internal/testutil"
)

func TestRestartBudgetDoesNotConsumeScanBudget(t *testing.T) {
	// An unlimited global computation budget still leaves each route bounded.
	before := time.Now()
	d := windowsRouteDeadline(time.Time{}, time.Second)
	if d.IsZero() || d.Before(before) || d.After(before.Add(2*time.Second)) {
		t.Fatal("route lost its own deadline when global budget was disabled")
	}
}

func TestSetupDiagnosticsAreJobLocal(t *testing.T) {
	a, b := newSetupScan(), newSetupScan()
	a.addDiag("first run only")
	b.addDiag("second run only")
	if strings.Contains(b.diagString(), "first run") || strings.Contains(a.diagString(), "second run") {
		t.Fatal("diagnostics leaked between jobs")
	}
}

func TestSetupCancelledBeforeAnyProcessWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err := RunSetupContext(ctx, SetupOptions{DBRoot: filepath.Join(t.TempDir(), "absent")})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled setup returned %v", err)
	}
}

func TestSetupOfflineCacheUsesExplicitAccountWithoutProcess(t *testing.T) {
	t.Setenv("WECHAT_CLI_PASSPHRASE_HEX", "")
	t.Setenv("WECHAT_CLI_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	root := filepath.Join(t.TempDir(), "wxid_selected_test")
	dir := filepath.Join(root, "db_storage")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path, _, salt, key := testutil.MustNewPassphraseDB(dir, "one.db", 1)
	_ = path
	cfg := &config.Config{DBRoot: filepath.Join(t.TempDir(), "old-account"), Wxid: "old-account"}
	cfg.SetVerifiedKey(salt, key, "synthetic-test", 1)
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	deps := defaultSetupDeps()
	deps.processes = func() ([]windowsProcess, error) {
		t.Fatal("valid offline cache must not enumerate processes")
		return nil, nil
	}
	res, _, err := runSetupWithDeps(context.Background(), SetupOptions{DBRoot: root}, deps)
	if err != nil || res == nil || res.Root != root || len(res.Keys) != 1 {
		t.Fatalf("offline explicit-account setup failed: %v", err)
	}
}

func TestSetupColdStartReachesLaunchBeforeScanBudget(t *testing.T) {
	t.Setenv("WECHAT_CLI_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("WECHAT_CLI_PASSPHRASE_HEX", "")
	t.Setenv("WECHAT_CLI_KEY_SCAN_TIMEOUT", "1ms")
	root := filepath.Join(t.TempDir(), "wxid_cold_test")
	if err := os.MkdirAll(filepath.Join(root, "db_storage"), 0o700); err != nil {
		t.Fatal(err)
	}
	testutil.MustNewPassphraseDB(filepath.Join(root, "db_storage"), "one.db", 1)
	deps := defaultSetupDeps()
	deps.processes = func() ([]windowsProcess, error) { return nil, nil }
	called := false
	stop := errors.New("test launch boundary reached")
	deps.restart = func(ctx context.Context, opts SetupOptions, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan) (uint32, error) {
		called = true
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) < 9*time.Minute {
			t.Fatal("login budget was capped by computation budget")
		}
		return 0, stop
	}
	_, _, err := runSetupWithDeps(context.Background(), SetupOptions{DBRoot: root, Restart: true}, deps)
	if !called || !errors.Is(err, stop) {
		t.Fatalf("cold start did not reach restart: %v", err)
	}
}

func TestSetupRestartCancellationPreservesCleanupFailure(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "absent-config.json")
	t.Setenv("WECHAT_CLI_CONFIG", configPath)
	t.Setenv("WECHAT_CLI_PASSPHRASE_HEX", "")
	t.Setenv("WECHAT_CLI_CAPTURE_KEY_FILE", "")
	t.Setenv("WX_MCP_CAPTURE_KEY_FILE", "")
	root := filepath.Join(t.TempDir(), "wxid_cancel_recovery")
	dir := filepath.Join(root, "db_storage")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	testutil.MustNewPassphraseDB(dir, "one.db", 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	failure := errors.New("synthetic restart cleanup error")
	deps := defaultSetupDeps()
	deps.restart = func(context.Context, SetupOptions, []windowsSourceDB, map[string]bool, *setupScan) (uint32, error) {
		cancel()
		return 0, errors.Join(ErrCaptureCleanup, failure)
	}
	res, _, err := runSetupWithDeps(ctx, SetupOptions{DBRoot: root, Restart: true}, deps)
	if res != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, failure) || !errors.Is(err, ErrCaptureCleanup) {
		t.Fatalf("restart cancellation hid cleanup failure: %v", err)
	}
	if _, err := os.Stat(configPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed/cancelled capture persisted config: %v", err)
	}
}

func TestSetupImportsAuthorizedFileWithoutProcess(t *testing.T) {
	t.Setenv("WECHAT_CLI_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("WECHAT_CLI_PASSPHRASE_HEX", "")
	t.Setenv("WECHAT_CLI_CAPTURE_PID", "")
	root := filepath.Join(t.TempDir(), "wxid_import_test")
	dir := filepath.Join(root, "db_storage")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	_, _, _, key := testutil.MustNewPassphraseDB(dir, "one.db", 1)
	capture := filepath.Join(t.TempDir(), "material.txt")
	if err := os.WriteFile(capture, []byte(key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WECHAT_CLI_CAPTURE_KEY_FILE", capture)
	deps := defaultSetupDeps()
	deps.processes = func() ([]windowsProcess, error) {
		return nil, errors.New("offline import incorrectly reached process enumeration")
	}
	deps.restart = func(context.Context, SetupOptions, []windowsSourceDB, map[string]bool, *setupScan) (uint32, error) {
		return 0, errors.New("offline import incorrectly reached restart")
	}
	res, _, err := runSetupWithDeps(context.Background(), SetupOptions{DBRoot: root, Restart: true}, deps)
	if err != nil || res == nil || len(res.Keys) != 1 {
		t.Fatalf("offline file import: %v", err)
	}
}

func TestSetupSourceDiscoveryDoesNotHideFailures(t *testing.T) {
	t.Run("missing-storage", func(t *testing.T) {
		_, _, err := windowsListSourceDBs(t.TempDir())
		if err == nil {
			t.Fatal("missing storage was silently accepted")
		}
	})
	t.Run("short-header", func(t *testing.T) {
		root := t.TempDir()
		dir := filepath.Join(root, "db_storage")
		os.MkdirAll(dir, 0o700)
		os.WriteFile(filepath.Join(dir, "broken.db"), []byte("short"), 0o600)
		_, _, err := windowsListSourceDBs(root)
		if err == nil {
			t.Fatal("truncated DB was accepted as a capture target")
		}
	})
	t.Run("skip-plaintext", func(t *testing.T) {
		root := t.TempDir()
		dir := filepath.Join(root, "db_storage")
		os.MkdirAll(dir, 0o700)
		os.WriteFile(filepath.Join(dir, "plain.db"), []byte("SQLite format 3\x00"), 0o600)
		dbs, salts, err := windowsListSourceDBs(root)
		if err != nil || len(dbs) != 0 || len(salts) != 0 {
			t.Fatal("plaintext incorrectly requires key capture")
		}
	})
}
