package guicore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"weixin-key/internal/testutil"
	"weixin-key/internal/wxkey"
)

func TestCloseWaitsForCaptureCleanupAndRejectsNewRuns(t *testing.T) {
	t.Setenv("WECHAT_CLI_PASSPHRASE_HEX", "")
	t.Setenv("WX_MCP_PASSPHRASE_HEX", "")
	t.Setenv("WECHAT_CLI_CONFIG", filepath.Join(t.TempDir(), "unused.json"))
	acct := filepath.Join(t.TempDir(), "wxid_recovery_fixture")
	storage := filepath.Join(acct, "db_storage")
	if err := os.MkdirAll(storage, 0o700); err != nil {
		t.Fatal(err)
	}
	testutil.MustNewPassphraseDB(storage, "one.db", 1)
	started, cancelled, release := make(chan *wxkey.CaptureRecovery, 1), make(chan struct{}), make(chan struct{})
	t.Cleanup(func() { close(release) })
	j := NewJob()
	opts := JobOptions{DBRoot: acct, OutDir: t.TempDir(), Resolve: wxkey.ResolveOptions{NoConfig: true},
		Capture: func(ctx context.Context, opts wxkey.SetupOptions) error {
			started <- opts.Recovery
			<-ctx.Done()
			close(cancelled)
			<-release
			return errors.Join(ctx.Err(), wxkey.ErrCaptureCleanup)
		}}
	if err := j.Start(opts); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		stage, _, _, _, _ := j.State()
		if stage == StageNeedsCapture {
			break
		}
		if time.Now().After(deadline) {
			j.Cancel()
			t.Fatal("job did not reach capture approval")
		}
		time.Sleep(time.Millisecond)
	}
	j.ApproveCapture()
	select {
	case r := <-started:
		if r == nil {
			t.Fatal("capture did not receive recovery controller")
		}
	case <-time.After(time.Second):
		t.Fatal("capture did not start")
	}
	done := j.RequestClose()
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("close did not request cancellation")
	}
	j.recoveryChanged(true, "synthetic retained owner")
	stage, _, _, _, _ := j.State()
	if stage != StageRecovering || !j.Running() {
		t.Fatal("recovery notification marked job finished")
	}
	select {
	case <-done:
		t.Fatal("close released owner before cleanup/join")
	default:
	}
	if err := j.Start(opts); err == nil {
		t.Fatal("close allowed a second capture")
	}
	release <- struct{}{}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("safe cleanup did not release close waiter")
	}
	stage, _, _, _, err := j.State()
	if stage != StageFailed || !errors.Is(err, wxkey.ErrCaptureCleanup) {
		t.Fatalf("concurrent cancellation hid cleanup error: stage=%s err=%v", stage, err)
	}
	if err := j.Start(opts); err == nil {
		t.Fatal("completed shutdown allowed a new run")
	}
}

func TestCloseIdleAndFailedValidation(t *testing.T) {
	for _, invalidStart := range []bool{false, true} {
		j := NewJob()
		if invalidStart && j.Start(JobOptions{}) == nil {
			t.Fatal("empty path accepted")
		}
		for i := 0; i < 2; i++ {
			select {
			case <-j.RequestClose():
			default:
				t.Fatal("idle/failed job blocked close")
			}
		}
	}
}
