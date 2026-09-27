package export

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func preparePublish(t *testing.T) (*Publisher, []plannedFile, []byte) {
	t.Helper()
	out, stage := t.TempDir(), t.TempDir()
	var plan []plannedFile
	for _, name := range []string{"a.txt", "b.txt", "manifest.json"} {
		src := filepath.Join(stage, name)
		if err := os.WriteFile(src, []byte("old-"+name), 0o600); err != nil {
			t.Fatal(err)
		}
		plan = append(plan, plannedFile{rel: name, srcPath: src})
	}
	p := &Publisher{Root: out, Cursors: map[string]SnapshotInfo{"db": {SourceSize: 1}}}
	if _, err := p.Publish("old", plan); err != nil {
		t.Fatal(err)
	}
	state, err := os.ReadFile(filepath.Join(out, ".export-state", "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range plan {
		if err := os.WriteFile(f.srcPath, []byte("new-"+f.rel), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	p.Cursors = map[string]SnapshotInfo{"db": {SourceSize: 2}}
	return p, plan, state
}
func assertOldPublish(t *testing.T, p *Publisher, plan []plannedFile, state []byte) {
	t.Helper()
	for _, f := range plan {
		b, err := os.ReadFile(filepath.Join(p.Root, f.rel))
		if err != nil || string(b) != "old-"+f.rel {
			t.Fatalf("old generation lost at %s: %v", f.rel, err)
		}
	}
	got, err := os.ReadFile(filepath.Join(p.Root, ".export-state", "state.json"))
	if err != nil || !bytes.Equal(got, state) {
		t.Fatal("failed publication advanced or damaged state/cursors")
	}
}
func TestPublishRollbackAfterActualWrites(t *testing.T) {
	for _, stop := range []int{0, 2} {
		t.Run(string(rune('0'+stop)), func(t *testing.T) {
			p, plan, state := preparePublish(t)
			injected := errors.New("synthetic write failure")
			p.afterWrite = func(i int) error {
				if i == stop {
					return injected
				}
				return nil
			}
			if _, err := p.Publish("failed", plan); !errors.Is(err, injected) {
				t.Fatalf("failure injection: %v", err)
			}
			assertOldPublish(t, p, plan, state)
			if _, err := os.Stat(filepath.Join(p.Root, transactionPath)); !os.IsNotExist(err) {
				t.Fatal("successful rollback left a journal")
			}
		})
	}
}
func TestPublishRejectsChangedStagingBeforeReplacingTarget(t *testing.T) {
	p, plan, state := preparePublish(t)
	p.afterWrite = func(i int) error {
		if i == 0 {
			return os.WriteFile(plan[1].srcPath, []byte("unplanned content"), 0o600)
		}
		return nil
	}
	if _, err := p.Publish("modified-stage", plan); err == nil {
		t.Fatal("changed staging accepted")
	}
	assertOldPublish(t, p, plan, state)
}
func TestPublishCrashChild(t *testing.T) {
	if os.Getenv("WEIXIN_EXPORT_TEST_CRASH") != "1" {
		return
	}
	root, stage := os.Getenv("WEIXIN_EXPORT_TEST_ROOT"), os.Getenv("WEIXIN_EXPORT_TEST_STAGE")
	p := &Publisher{Root: root, afterWrite: func(int) error { os.Exit(37); return nil }}
	_, err := p.Publish("crashed", []plannedFile{{rel: "a.txt", srcPath: filepath.Join(stage, "a.txt")}, {rel: "b.txt", srcPath: filepath.Join(stage, "b.txt")}, {rel: "manifest.json", srcPath: filepath.Join(stage, "manifest.json")}})
	t.Fatalf("child did not reach crash point: %v", err)
}
func TestPublishRecoversActualProcessCrash(t *testing.T) {
	p, plan, state := preparePublish(t)
	cmd := exec.Command(os.Args[0], "-test.run=^TestPublishCrashChild$")
	cmd.Env = append(os.Environ(), "WEIXIN_EXPORT_TEST_CRASH=1", "WEIXIN_EXPORT_TEST_ROOT="+p.Root, "WEIXIN_EXPORT_TEST_STAGE="+filepath.Dir(plan[0].srcPath))
	err := cmd.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 37 {
		t.Fatalf("crash child outcome: %v", err)
	}
	if _, err := os.Stat(filepath.Join(p.Root, transactionPath)); err != nil {
		t.Fatal("process crash did not leave recovery evidence")
	}
	if err := p.Recover(); err != nil {
		t.Fatal(err)
	}
	assertOldPublish(t, p, plan, state)
	if err := p.Recover(); err != nil {
		t.Fatal("recovery is not idempotent", err)
	}
	p.afterWrite = nil
	if _, err := p.Publish("retry", plan); err != nil {
		t.Fatal("recovered output cannot be retried", err)
	}
}
func TestRecoveryPreservesExternalEdit(t *testing.T) {
	p, plan, _ := preparePublish(t)
	func() {
		defer func() { _ = recover() }()
		p.afterWrite = func(int) error { panic("simulated interruption") }
		p.Publish("interrupted", plan)
	}()
	path := filepath.Join(p.Root, "a.txt")
	if err := os.WriteFile(path, []byte("user edit"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := p.Recover(); err == nil {
		t.Fatal("external edit was not reported")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "user edit" {
		t.Fatal("recovery overwrote user edit")
	}
	if _, err := os.Stat(filepath.Join(p.Root, transactionPath)); err != nil {
		t.Fatal("recovery evidence lost")
	}
}
func TestExportRejectsDifferentAccountBeforeSnapshot(t *testing.T) {
	a := newIncrAccount(t, baseRows)
	out := t.TempDir()
	opts := a.options(out)
	opts.AccountID = "account-one"
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(out, "data", "messages.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	opts.AccountID = "account-two"
	opts.OnProgress = func(p Progress) {
		if p.Stage == "snapshot" {
			t.Fatal("wrong output account was not rejected before snapshot")
		}
	}
	if _, err := Run(opts); err == nil {
		t.Fatal("cross-account output accepted")
	}
	after, _ := os.ReadFile(filepath.Join(out, "data", "messages.jsonl"))
	if !bytes.Equal(before, after) {
		t.Fatal("different account altered prior messages")
	}
}
func TestHistoryRejectsForeignOrMalformedLines(t *testing.T) {
	for _, content := range []string{`{broken`, `{"account_id":"foreign","message_id":"one"}`} {
		path := filepath.Join(t.TempDir(), "history.jsonl")
		os.WriteFile(path, []byte(content+"\n"), 0o600)
		if _, err := loadPrevIndex(context.Background(), path, "selected"); err == nil {
			t.Fatal("invalid history was accepted")
		}
	}
}
func TestLegacyOutputAdoptionRequiresHashBoundAccount(t *testing.T) {
	root, stage := t.TempDir(), t.TempDir()
	manifest := filepath.Join(stage, "manifest.json")
	b, _ := json.Marshal(Report{AccountID: "one"})
	os.WriteFile(manifest, b, 0o600)
	p := &Publisher{Root: root}
	if _, err := p.Publish("legacy", []plannedFile{{rel: "manifest.json", srcPath: manifest}}); err != nil {
		t.Fatal(err)
	}
	if err := validateOutputAccount(root, "one"); err != nil {
		t.Fatal(err)
	}
	if err := validateOutputAccount(root, "two"); err == nil {
		t.Fatal("legacy output adopted by another account")
	}
}

func TestIncrementalHistoryUsesOwnedConflictSibling(t *testing.T) {
	a := newIncrAccount(t, baseRows)
	out := t.TempDir()
	userFile := filepath.Join(out, "data", "messages.jsonl")
	os.MkdirAll(filepath.Dir(userFile), 0o700)
	os.WriteFile(userFile, []byte("unrelated user notes"), 0o600)
	if _, err := Run(a.options(out)); err != nil {
		t.Fatal(err)
	}
	rep, err := Run(a.options(out))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Merge.Unchanged != 2 || rep.Merge.New != 0 {
		t.Fatalf("owned conflict history not reused: %+v", rep.Merge)
	}
	b, _ := os.ReadFile(userFile)
	if string(b) != "unrelated user notes" {
		t.Fatal("user file changed")
	}
}

func TestRecoveryJournalCannotTargetState(t *testing.T) {
	p, _, state := preparePublish(t)
	root, err := os.OpenRoot(p.Root)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	tx := publishTransaction{RunID: "unsafe", BackupDir: ".export-state/backup-unsafe", Entries: []publishEntry{{Rel: ".export-state/state.json", NewHash: "bad"}}}
	if err := rootJSON(root, transactionPath, tx); err != nil {
		t.Fatal(err)
	}
	if err := p.Recover(); err == nil {
		t.Fatal("journal was allowed to target internal state")
	}
	got, _ := os.ReadFile(filepath.Join(p.Root, ".export-state", "state.json"))
	if !bytes.Equal(got, state) {
		t.Fatal("invalid journal altered state")
	}
}
