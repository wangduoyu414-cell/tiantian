package guicore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"weixin-key/internal/testutil"
	"weixin-key/internal/wxkey"
)

// buildPlainAccount makes an account dir with PLAINTEXT message DBs, so
// preflight and export need no key material at all.
func buildPlainAccount(t *testing.T) string {
	t.Helper()
	acct := filepath.Join(t.TempDir(), "wxid_gui_test")
	msgDir := filepath.Join(acct, "db_storage", "message")
	if err := os.MkdirAll(msgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := testutil.BuildSQLiteDB(filepath.Join(msgDir, "msg_0.db"),
		`CREATE TABLE Msg_x (localId INTEGER PRIMARY KEY, serverId INTEGER, createTime INTEGER, localType INTEGER, messageContent TEXT, isSender INTEGER, talker TEXT)`,
		`INSERT INTO Msg_x VALUES (1, 1001, 1715000000, 1, '你好', 0, 'wxid_a')`,
		`INSERT INTO Msg_x VALUES (2, 1002, 1715000100, 1, '第二条', 1, '')`,
	); err != nil {
		t.Fatal(err)
	}
	return acct
}

func TestSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WECHAT_CLI_CONFIG", filepath.Join(dir, "config.json"))
	s := &Settings{LastExportDir: `D:\Exports\微信`, DBRoot: `D:\data\wxid_x`}
	if err := SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if got.LastExportDir != s.LastExportDir || got.DBRoot != s.DBRoot {
		t.Fatalf("settings = %+v, want %+v", got, s)
	}
	// Corrupt file -> error, not panic.
	p, _ := SettingsPath()
	if err := os.WriteFile(p, []byte("{bad json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSettings(); err == nil {
		t.Fatal("corrupt settings must surface an error")
	}
}

func TestValidateExportDir(t *testing.T) {
	src := t.TempDir()
	if err := ValidateExportDir("", ""); err == nil {
		t.Fatal("empty dir accepted")
	}
	if err := ValidateExportDir(filepath.Join(src, "sub"), src); err == nil {
		t.Fatal("dir inside source accepted")
	}
	if err := ValidateExportDir(src, src); err == nil {
		t.Fatal("dir == source accepted")
	}
	good := filepath.Join(t.TempDir(), "out")
	if err := ValidateExportDir(good, src); err != nil {
		t.Fatalf("good dir rejected: %v", err)
	}
	// A file is not a directory.
	f := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateExportDir(f, src); err == nil {
		t.Fatal("file accepted as export dir")
	}
}

func TestPreflightPlaintextNeedsNothing(t *testing.T) {
	acct := buildPlainAccount(t)
	pf, err := RunPreflight(acct, wxkey.ResolveOptions{NoConfig: true})
	if err != nil {
		t.Fatal(err)
	}
	if !pf.OfflineReady || len(pf.NeedsCapture) != 0 {
		t.Fatalf("plaintext account should be offline-ready: %+v", pf)
	}
	if pf.AccountID != "wxid_gui" {
		t.Fatalf("account id = %q", pf.AccountID)
	}
}

// The one-click happy path on a plaintext account: validate -> preflight ->
// export -> done, with no capture ever needed.
func TestJobOneClickPlaintext(t *testing.T) {
	acct := buildPlainAccount(t)
	out := filepath.Join(t.TempDir(), "导出 结果")
	j := NewJob()
	captureCalled := false
	err := j.Start(JobOptions{
		DBRoot:  acct,
		OutDir:  out,
		Resolve: wxkey.ResolveOptions{NoConfig: true},
		Capture: func(context.Context, wxkey.SetupOptions) error { captureCalled = true; return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	waitForJob(t, j)
	stage, events, rep, pf, err := j.State()
	if err != nil {
		t.Fatal(err)
	}
	if stage != StageDone {
		t.Fatalf("stage = %s, want done; events: %v", stage, events)
	}
	if rep == nil || rep.TotalMessages != 2 || rep.Status != "complete" {
		t.Fatalf("report = %+v", rep)
	}
	if pf == nil || !pf.OfflineReady {
		t.Fatal("preflight missing or not ready")
	}
	if captureCalled {
		t.Fatal("capture must not run when material is already sufficient")
	}
	if _, err := os.Stat(filepath.Join(out, "data", "messages.jsonl")); err != nil {
		t.Fatal("export output missing")
	}
}

// Encrypted DBs without material park at capture; the injected CaptureFunc
// runs, and when it cannot produce material the job fails honestly.
func TestJobNeedsCaptureFailsHonestlyWithoutMaterial(t *testing.T) {
	acct := filepath.Join(t.TempDir(), "wxid_enc_test")
	msgDir := filepath.Join(acct, "db_storage", "message")
	if err := os.MkdirAll(msgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, _, _, _ = testutil.MustNewPassphraseDB(msgDir, "msg_0.db", 1) // encrypted, no key provided
	j := NewJob()
	var events []string
	err := j.Start(JobOptions{
		DBRoot:  acct,
		OutDir:  filepath.Join(t.TempDir(), "out"),
		Resolve: wxkey.ResolveOptions{NoConfig: true},
		Capture: func(_ context.Context, opts wxkey.SetupOptions) error {
			opts.Progress("尝试采集")
			return errors.New("无法获取材料（测试注入）")
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = events
	// The job parks at needs_capture until the user approves the restart.
	{
		deadline := time.Now().Add(10 * time.Second)
		for {
			stage, _, _, _, _ := j.State()
			if stage == StageNeedsCapture {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("job never parked; stage=%s", stage)
			}
			time.Sleep(10 * time.Millisecond)
		}
		j.ApproveCapture()
	}
	waitForJob(t, j)
	stage, _, _, pf, err := j.State()
	if stage != StageFailed {
		t.Fatalf("stage = %s, want failed", stage)
	}
	if err == nil || !strings.Contains(err.Error(), "采集") && !strings.Contains(err.Error(), "材料") {
		t.Fatalf("error should mention capture/material: %v", err)
	}
	if pf == nil || len(pf.NeedsCapture) == 0 {
		t.Fatal("preflight should record the unresolvable DB")
	}
}

// Re-entry protection: a second Start while running is rejected.
func TestJobRejectsReentry(t *testing.T) {
	acct := buildPlainAccount(t)
	j := NewJob()
	block := make(chan struct{})
	if err := j.Start(JobOptions{
		DBRoot:  acct,
		OutDir:  filepath.Join(t.TempDir(), "out"),
		Resolve: wxkey.ResolveOptions{NoConfig: true},
		Capture: func(context.Context, wxkey.SetupOptions) error { <-block; return nil },
	}); err != nil {
		t.Fatal(err)
	}
	// Wait until running.
	deadline := time.Now().Add(5 * time.Second)
	for !j.Running() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if err := j.Start(JobOptions{DBRoot: acct, OutDir: t.TempDir()}); err == nil {
		t.Fatal("re-entry accepted while running")
	}
	// Unblock and finish. Note: plaintext account never calls Capture, so the
	// job may already be done; both outcomes are fine as long as no double-run.
	close(block)
	waitForJob(t, j)
}

func waitForJob(t *testing.T, j *Job) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if !j.Running() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("job did not finish in time")
}

// A job that needs material must PARK at needs_capture until approved; it
// must not restart WeChat on its own.
func TestJobParksForCaptureApproval(t *testing.T) {
	acct := filepath.Join(t.TempDir(), "wxid_enc_test")
	msgDir := filepath.Join(acct, "db_storage", "message")
	if err := os.MkdirAll(msgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, _, _, _ = testutil.MustNewPassphraseDB(msgDir, "msg_0.db", 1)
	j := NewJob()
	captureRan := make(chan struct{}, 1)
	if err := j.Start(JobOptions{
		DBRoot:  acct,
		OutDir:  filepath.Join(t.TempDir(), "out"),
		Resolve: wxkey.ResolveOptions{NoConfig: true},
		Capture: func(context.Context, wxkey.SetupOptions) error { captureRan <- struct{}{}; return errors.New("stop") },
	}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		stage, _, _, _, _ := j.State()
		if stage == StageNeedsCapture {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job never parked; stage=%s", stage)
		}
		time.Sleep(10 * time.Millisecond)
	}
	select {
	case <-captureRan:
		t.Fatal("capture ran before user approval")
	default:
	}
	j.ApproveCapture()
	waitForJob(t, j)
	select {
	case <-captureRan:
	default:
		t.Fatal("capture never ran after approval")
	}
	stage, _, _, _, err := j.State()
	if stage != StageFailed || err == nil {
		t.Fatalf("stage=%s err=%v", stage, err)
	}
}

// Cancelling at the needs-capture gate must not run capture at all.
func TestJobCancelAtCaptureGate(t *testing.T) {
	acct := filepath.Join(t.TempDir(), "wxid_enc_test")
	msgDir := filepath.Join(acct, "db_storage", "message")
	if err := os.MkdirAll(msgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, _, _, _ = testutil.MustNewPassphraseDB(msgDir, "msg_0.db", 1)
	j := NewJob()
	captureRan := make(chan struct{}, 1)
	if err := j.Start(JobOptions{
		DBRoot:  acct,
		OutDir:  filepath.Join(t.TempDir(), "out"),
		Resolve: wxkey.ResolveOptions{NoConfig: true},
		Capture: func(context.Context, wxkey.SetupOptions) error { captureRan <- struct{}{}; return nil },
	}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		stage, _, _, _, _ := j.State()
		if stage == StageNeedsCapture {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job never parked; stage=%s", stage)
		}
		time.Sleep(10 * time.Millisecond)
	}
	j.Cancel()
	waitForJob(t, j)
	select {
	case <-captureRan:
		t.Fatal("capture ran despite cancel at the gate")
	default:
	}
	stage, _, _, _, _ := j.State()
	if stage != StageCancelled {
		t.Fatalf("stage=%s, want cancelled", stage)
	}
}

// Multi-account roots list every account with db_storage, in deterministic
// order, skipping tooling dirs.
func TestListAccounts(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"wxid_bbb_1", "all_users", "wxid_aaa_2", "backup"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"wxid_bbb_1", "wxid_aaa_2"} {
		if err := os.MkdirAll(filepath.Join(root, name, "db_storage"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	accts, err := ListAccounts(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(accts) != 2 {
		t.Fatalf("accounts = %d, want 2", len(accts))
	}
	wxids := []string{accts[0].WxID, accts[1].WxID}
	if wxids[0] != "wxid_aaa" && wxids[1] != "wxid_bbb" {
		t.Fatalf("wxids = %v", wxids)
	}
	if _, err := ListAccounts(filepath.Join(root, "nonexistent")); err == nil {
		t.Fatal("missing root must error")
	}
}
