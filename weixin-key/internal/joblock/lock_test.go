package joblock

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestLockRejectsOverlapAndReleases(t *testing.T) {
	path := filepath.Join(t.TempDir(), "job.lock")
	release, err := Acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	if r, err := Acquire(path); !errors.Is(err, ErrBusy) {
		if r != nil {
			r()
		}
		t.Fatalf("overlap err=%v", err)
	}
	release()
	r, err := Acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	r()
}
