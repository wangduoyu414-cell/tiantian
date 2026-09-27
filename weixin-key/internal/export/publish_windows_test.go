//go:build windows

package export

import (
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestPublishStateCommitFailureRollsBackContentAndCursors(t *testing.T) {
	p, plan, state := preparePublish(t)
	var held windows.Handle
	p.afterWrite = func(i int) error {
		if i != len(plan)-1 {
			return nil
		}
		name, err := windows.UTF16PtrFromString(filepath.Join(p.Root, ".export-state", "state.json"))
		if err != nil {
			return err
		}
		held, err = windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
		return err
	}
	_, err := p.Publish("state-blocked", plan)
	if held != 0 && held != windows.InvalidHandle {
		windows.CloseHandle(held)
	}
	if err == nil {
		t.Fatal("locked state unexpectedly committed")
	}
	assertOldPublish(t, p, plan, state)
}
