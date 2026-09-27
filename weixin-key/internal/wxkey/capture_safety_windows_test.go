//go:build windows

package wxkey

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"weixin-key/internal/config"
	"weixin-key/internal/testutil"
)

func TestSetupNonRestartFailurePreservesConfigBytes(t *testing.T) {
	for _, fullCoverage := range []bool{false, true} {
		t.Run(map[bool]string{false: "partial-cache-and-gate-error", true: "full-coverage-and-cleanup-error"}[fullCoverage], func(t *testing.T) {
			cfgPath := filepath.Join(t.TempDir(), "config.json")
			t.Setenv("WECHAT_CLI_CONFIG", cfgPath)
			t.Setenv("WECHAT_CLI_PASSPHRASE_HEX", "")
			t.Setenv("WECHAT_CLI_CAPTURE_KEY_FILE", "")
			t.Setenv("WX_MCP_CAPTURE_KEY_FILE", "")
			root := t.TempDir()
			dir := filepath.Join(root, "db_storage")
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			_, _, firstSalt, firstKey := testutil.MustNewPassphraseDB(dir, "one.db", 1)
			_, _, secondSalt, secondKey := testutil.MustNewPassphraseDB(dir, "two.db", 1)
			c := &config.Config{DBRoot: root}
			c.SetVerifiedKey(firstSalt, firstKey, "synthetic-cache", 1)
			if err := config.Save(c); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(cfgPath)
			if err != nil {
				t.Fatal(err)
			}
			wantErr := ErrActiveCaptureUnavailable
			if fullCoverage {
				wantErr = errors.New("synthetic non-restart cleanup failure")
			}
			deps := defaultSetupDeps()
			deps.processes = func() ([]windowsProcess, error) { return []windowsProcess{{pid: 111}}, nil }
			deps.routes = func(_ []windowsProcess, _ []windowsSourceDB, _ map[string]bool, scan *setupScan, _ *windowsSetupStats, _ *uint32, _ func(string)) error {
				if fullCoverage {
					scan.noteRawKey(secondSalt, secondKey, "synthetic-route")
				}
				return wantErr
			}
			res, _, err := runSetupWithDeps(context.Background(), SetupOptions{DBRoot: root}, deps)
			if res != nil || !errors.Is(err, wantErr) {
				t.Errorf("non-restart error became a successful/partial result: %v", err)
			}
			after, err := os.ReadFile(cfgPath)
			if err != nil || !bytes.Equal(before, after) {
				t.Error("failed non-restart capture changed config bytes")
			}
		})
	}
}

func TestActiveCaptureSafetyGateCannotBeBypassed(t *testing.T) {
	t.Setenv("WECHAT_CLI_KEY_HOOK", "1")
	t.Setenv("WECHAT_CLI_KEY_RESTART", "1")
	t.Setenv("WECHAT_CLI_WX_KEY_DLL", filepath.Join(t.TempDir(), "absent-helper.dll"))
	// Nil scan/receivers deliberately prove the stop happens before use of
	// live capture dependencies. No real PID, executable or helper is supplied.
	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{"hook", func() error { return windowsHookScan(0, nil, nil, nil, time.Time{}) }},
		{"launch", func() error { _, err := windowsLaunchCapture("", nil, nil, nil, time.Time{}); return err }},
		{"prepared-launch", func() error { _, err := windowsLaunchCaptureOnOwner(nil, "", nil, nil, nil, time.Time{}); return err }},
		{"legacy-launch", func() error {
			_, err := LaunchWeChat(filepath.Join(t.TempDir(), "absent.exe"))
			return err
		}},
		{"restart", func() error {
			_, err := windowsRestartAccount(context.Background(), SetupOptions{Restart: true}, nil, nil, nil)
			return err
		}},
		{"dll-load", func() error { _, err := windowsDLLScan(nil, nil, nil, nil, time.Time{}); return err }},
		{"dll-call", func() error {
			var d *windowsWxKeyDLL
			_, err := d.getKey(0, time.Time{}, nil)
			return err
		}},
		{"route-no-fallback", func() error { return runWindowsRoutes(nil, nil, nil, nil, nil, nil, nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, ErrActiveCaptureUnavailable) {
				t.Fatalf("active capture did not fail closed: %v", err)
			}
		})
	}
}

func TestTrapFlagAccessDoesNotModifyAdjacentDR0(t *testing.T) {
	c := newAlignedContext(contextControl | contextDebugRegisters)
	const sentinel uint64 = 0xAABBCCDDEEFF0011
	setCtxField(c, ctxOffDR0, sentinel)
	setCtxField32(c, ctxOffEFlags, 0x202)
	setCtxField32(c, ctxOffEFlags, ctxField32(c, ctxOffEFlags)|eflagsTrapFlag)
	if ctxField32(c, ctxOffEFlags) != 0x302 || ctxField(c, ctxOffDR0) != sentinel {
		t.Fatal("TF update crossed the 32-bit EFlags boundary")
	}
	setCtxField32(c, ctxOffEFlags, ctxField32(c, ctxOffEFlags)&^uint32(eflagsTrapFlag))
	if ctxField32(c, ctxOffEFlags) != 0x202 || ctxField(c, ctxOffDR0) != sentinel {
		t.Fatal("TF clear crossed the 32-bit EFlags boundary")
	}
}

func TestSetupDoesNotHideCaptureFailureAfterCoverage(t *testing.T) {
	t.Setenv("WECHAT_CLI_CONFIG", filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("WECHAT_CLI_PASSPHRASE_HEX", "")
	t.Setenv("WECHAT_CLI_CAPTURE_KEY_FILE", "")
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "db_storage"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, _, salt, key := testutil.MustNewPassphraseDB(filepath.Join(root, "db_storage"), "one.db", 1)
	captureErr := errors.New("synthetic cleanup failure")
	deps := defaultSetupDeps()
	deps.restart = func(_ context.Context, _ SetupOptions, _ []windowsSourceDB, _ map[string]bool, scan *setupScan) (uint32, error) {
		scan.noteRawKey(salt, key, "synthetic")
		return 0, captureErr
	}
	res, _, err := runSetupWithDeps(context.Background(), SetupOptions{DBRoot: root, Restart: true}, deps)
	if res != nil || !errors.Is(err, captureErr) {
		t.Fatalf("capture/cleanup failure became success: %v", err)
	}
	got, err := config.Load()
	if err != nil || got.Ready() {
		t.Fatal("failed capture committed a new configuration")
	}
}
