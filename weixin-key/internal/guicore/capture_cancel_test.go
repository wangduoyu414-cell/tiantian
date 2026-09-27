package guicore

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"weixin-key/internal/testutil"
	"weixin-key/internal/wxkey"
)

func TestCaptureReceivesSelectedAccountAndCancellation(t *testing.T) {
	t.Setenv("WECHAT_CLI_PASSPHRASE_HEX", "")
	t.Setenv("WECHAT_CLI_CONFIG", filepath.Join(t.TempDir(), "unused.json"))
	acct := filepath.Join(t.TempDir(), "wxid_selected_test")
	dir := filepath.Join(acct, "db_storage")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	testutil.MustNewPassphraseDB(dir, "one.db", 1)
	started := make(chan wxkey.SetupOptions, 1)
	stopped := make(chan struct{})
	j := NewJob()
	if err := j.Start(JobOptions{DBRoot: acct, OutDir: t.TempDir(), Resolve: wxkey.ResolveOptions{NoConfig: true},
		Capture: func(ctx context.Context, opts wxkey.SetupOptions) error {
			started <- opts
			<-ctx.Done()
			close(stopped)
			return ctx.Err()
		}}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		s, _, _, _, _ := j.State()
		if s == StageNeedsCapture {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("job did not reach approval gate")
		}
		time.Sleep(time.Millisecond)
	}
	j.ApproveCapture()
	select {
	case opts := <-started:
		if opts.DBRoot != acct || !opts.Restart || !opts.Resolve.NoConfig {
			t.Fatal("capture input lost account, approval, or material policy")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("capture not started")
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); j.Cancel() }()
	}
	wg.Wait()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("capture did not receive cancellation")
	}
	waitForJob(t, j)
	stage, _, _, _, _ := j.State()
	if stage != StageCancelled {
		t.Fatalf("stage=%s", stage)
	}
}
