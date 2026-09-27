package export

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublisherRejectsCorruptStateBeforeWriting(t *testing.T) {
	out, stage := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(out, ".export-state"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, ".export-state", "state.json"), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(stage, "new")
	if err := os.WriteFile(src, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := (&Publisher{Root: out}).Publish("bad-state", []plannedFile{{rel: "data.txt", srcPath: src}})
	if err == nil {
		t.Fatal("corrupt publish state was silently treated as a clean export")
	}
	if _, err := os.Stat(filepath.Join(out, "data.txt")); !os.IsNotExist(err) {
		t.Fatal("content changed before state validation")
	}
}

func TestPublisherFailurePreservesPreviousGeneration(t *testing.T) {
	out, stage := t.TempDir(), t.TempDir()
	src := filepath.Join(stage, "new")
	if err := os.WriteFile(src, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	pub := &Publisher{Root: out}
	if _, err := pub.Publish("initial", []plannedFile{{rel: "a.txt", srcPath: src}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(out, "z.txt"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := pub.Publish("failing", []plannedFile{{rel: "a.txt", srcPath: src}, {rel: "z.txt", srcPath: src}}); err == nil {
		t.Fatal("invalid second target did not fail")
	}
	got, err := os.ReadFile(filepath.Join(out, "a.txt"))
	if err != nil || string(got) != "old" {
		t.Fatal("failed publish damaged the previously committed generation")
	}
}

func TestExportCancellationAtParseDoesNotPublish(t *testing.T) {
	acct, pp := buildFixtureAccount(t)
	out := t.TempDir()
	cancel := make(chan struct{})
	closed := false
	rep, err := Run(Options{DBRoot: acct, OutDir: out, Resolve: mustResolvePassphrase(t, pp), Cancel: cancel,
		OnProgress: func(p Progress) {
			if p.Stage == "parse" && !closed {
				close(cancel)
				closed = true
			}
		}})
	if err == nil || rep.Status != "cancelled" {
		t.Fatalf("cancel during parse ignored; status=%s err=%v", rep.Status, err)
	}
	if _, err := os.Stat(filepath.Join(out, "manifest.json")); !os.IsNotExist(err) {
		t.Fatal("cancelled export published a manifest")
	}
}
