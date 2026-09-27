//go:build windows

package wxkey

import (
	"context"
	"errors"
	"slices"
	"syscall"
	"testing"
	"time"
)

type fakeAccountController struct {
	log        []string
	targets    []windowsProcess
	owners     []accountProcess
	paths      map[uint32]string
	pinFail    uint32
	aliveFail  uint32
	prepareErr error
	change     bool
	remain     bool
	calls      int
}

func (f *fakeAccountController) deps() accountRestartDeps {
	return accountRestartDeps{
		processes: func() ([]windowsProcess, error) {
			f.calls++
			if f.calls == 2 && f.change {
				return append(slices.Clone(f.targets), windowsProcess{pid: 999}), nil
			}
			if f.calls >= 3 && !f.remain {
				return nil, nil
			}
			return f.targets, nil
		},
		owners: func([]windowsSourceDB) ([]accountProcess, error) { return f.owners, nil },
		pin: func(p accountProcess) (*accountControlTarget, error) {
			f.log = append(f.log, "pin")
			if p.pid == f.pinFail {
				return nil, errors.New("synthetic identity failure")
			}
			return &accountControlTarget{
				identity: p, exe: f.paths[p.pid],
				alive: func() error {
					if p.pid == f.aliveFail {
						return errors.New("synthetic exit")
					}
					return nil
				},
				close:   func() error { f.log = append(f.log, "close"); return nil },
				wait:    func(context.Context) error { f.log = append(f.log, "wait"); return nil },
				release: func() { f.log = append(f.log, "release") },
			}, nil
		},
		exe:     func() (string, error) { return `D:\synthetic\Weixin.exe`, nil },
		prepare: func(string) error { f.log = append(f.log, "prepare"); return f.prepareErr },
		launch: func(string, []windowsSourceDB, map[string]bool, *setupScan, time.Time) (uint32, error) {
			f.log = append(f.log, "launch")
			return 777, nil
		},
	}
}

func newFakeAccountController() *fakeAccountController {
	return &fakeAccountController{
		targets: []windowsProcess{{pid: 1}, {pid: 2}},
		owners:  []accountProcess{{pid: 1}, {pid: 2}},
		paths:   map[uint32]string{1: `D:\synthetic\Weixin.exe`, 2: `D:\synthetic\Weixin.exe`},
	}
}

func TestAccountRestartPreflightsEverythingBeforeAnyClose(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*fakeAccountController)
	}{
		{"later-pin-fails", func(f *fakeAccountController) { f.pinFail = 2 }},
		{"later-exe-differs", func(f *fakeAccountController) { f.paths[2] = `D:\other\Weixin.exe` }},
		{"unassociated-process", func(f *fakeAccountController) { f.owners = f.owners[:1] }},
		{"no-owners", func(f *fakeAccountController) { f.owners = nil }},
		{"capture-not-ready", func(f *fakeAccountController) { f.prepareErr = ErrActiveCaptureUnavailable }},
		{"new-process-before-close", func(f *fakeAccountController) { f.change = true }},
		{"exited-identity", func(f *fakeAccountController) { f.aliveFail = 2 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakeAccountController()
			tc.edit(f)
			_, err := restartAccountWithDeps(context.Background(), SetupOptions{}, nil, nil, nil, f.deps())
			if err == nil {
				t.Fatal("invalid preflight accepted")
			}
			if slices.Contains(f.log, "close") || slices.Contains(f.log, "launch") {
				t.Fatalf("failure had external side effects: %v", f.log)
			}
			pins, releases := 0, 0
			for _, event := range f.log {
				if event == "pin" {
					pins++
				}
				if event == "release" {
					releases++
				}
			}
			if f.pinFail != 0 {
				pins--
			}
			if pins != releases {
				t.Fatalf("pinned handles leaked: %v", f.log)
			}
		})
	}
}

func TestAccountRestartPreflightAndShutdownOrder(t *testing.T) {
	f := newFakeAccountController()
	pid, err := restartAccountWithDeps(context.Background(), SetupOptions{}, nil, nil, nil, f.deps())
	if err != nil || pid != 777 {
		t.Fatalf("synthetic restart: %v", err)
	}
	want := []string{"pin", "pin", "prepare", "close", "wait", "close", "wait", "launch", "release", "release"}
	if !slices.Equal(f.log, want) {
		t.Fatalf("order %v, want %v", f.log, want)
	}
	f = newFakeAccountController()
	f.remain = true
	if _, err := restartAccountWithDeps(context.Background(), SetupOptions{}, nil, nil, nil, f.deps()); err == nil || slices.Contains(f.log, "launch") {
		t.Fatal("launched into an unconfirmed remaining session")
	}
}

func TestAccountTargetsExcludeUnassociatedPIDsAndHoldIdentity(t *testing.T) {
	f := newFakeAccountController()
	f.owners = []accountProcess{{pid: 2}, {pid: 909}} // unrelated indexer is not WeChat
	deps := f.deps()
	targets, err := selectAccountTargets(context.Background(), f.targets, nil, false, deps.owners, deps.pin)
	if err != nil {
		t.Fatal(err)
	}
	defer releaseAccountTargets(targets)
	if len(targets) != 1 || targets[0].identity.pid != 2 || len(f.log) != 1 {
		t.Fatal("non-restart route received unrelated targets")
	}
	f.aliveFail = 2
	if targets[0].alive() == nil {
		t.Fatal("exited identity retained for route execution")
	}
}

func TestAccountRestartCancellationBeforeSideEffects(t *testing.T) {
	f := newFakeAccountController()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := restartAccountWithDeps(ctx, SetupOptions{}, nil, nil, nil, f.deps()); !errors.Is(err, context.Canceled) || f.calls != 0 {
		t.Fatal("cancelled restart reached process work")
	}
}

func TestAccountRestartInventoryCannotBeNarrowedBySelectionOverrides(t *testing.T) {
	t.Setenv("WECHAT_CLI_WECHAT_PID", "1")
	t.Setenv("WECHAT_CLI_WECHAT_PROCESS", "Weixin.exe")
	f := newFakeAccountController()
	f.owners = []accountProcess{{pid: 1}}
	actual := []windowsProcess{{pid: 1, exe: "Weixin.exe"}, {pid: 2, exe: "WeChat.exe"}, {pid: 3, exe: "unrelated.exe"}}
	enumerate := func() ([]windowsProcess, error) { return actual, nil }
	procs, err := allWeChatProcesses(enumerate)
	if err != nil || len(procs) != 2 || procs[0].pid != 1 || procs[1].pid != 2 {
		t.Fatal("actual inventory was filtered by target-selection environment")
	}
	// Use the production default inventory wiring; all other dependencies are
	// fake so even a regression can never close a real window.
	deps := f.deps()
	deps.processes = defaultAccountRestartDeps(enumerate, nil).processes
	if _, err := restartAccountWithDeps(context.Background(), SetupOptions{}, nil, nil, nil, deps); err == nil {
		t.Fatal("unassociated actual instance did not block restart")
	}
	if slices.Contains(f.log, "close") || slices.Contains(f.log, "launch") ||
		!slices.Equal(f.log, []string{"pin", "release"}) {
		t.Fatalf("other-instance refusal had side effects or leaked identity: %v", f.log)
	}
	// A stale selected PID cannot remain in inventory after its actual exit.
	actual = nil
	if procs, err := allWeChatProcesses(enumerate); err != nil || len(procs) != 0 {
		t.Fatal("exited configured PID was treated as a running instance")
	}
	denied := errors.New("synthetic inventory failure")
	if _, err := allWeChatProcesses(func() ([]windowsProcess, error) { return nil, denied }); !errors.Is(err, denied) {
		t.Fatal("inventory failure was treated as no running instances")
	}
}

func TestProcessSnapshotDoesNotReturnPartialInventoryAfterFailure(t *testing.T) {
	for _, lastErr := range []error{syscall.ERROR_ACCESS_DENIED, nil, syscall.Errno(0)} {
		procs, err := readProcessSnapshot(func(first bool, entry *windowsProcessEntry32) (uintptr, error) {
			if first {
				entry.ProcessID = 1
				copy(entry.ExeFile[:], syscall.StringToUTF16("Weixin.exe"))
				return 1, nil
			}
			return 0, lastErr
		})
		if err == nil || procs != nil {
			t.Errorf("failed enumeration returned successful/partial inventory: count=%d err=%v", len(procs), err)
		}
		if lastErr == syscall.ERROR_ACCESS_DENIED && !errors.Is(err, lastErr) {
			t.Error("original Win32 failure was lost")
		}
	}
	procs, err := readProcessSnapshot(func(first bool, entry *windowsProcessEntry32) (uintptr, error) {
		if first {
			entry.ProcessID = 1
			return 1, nil
		}
		return 0, syscall.ERROR_NO_MORE_FILES
	})
	if err != nil || len(procs) != 1 {
		t.Fatal("normal end of nonempty inventory failed")
	}
	procs, err = readProcessSnapshot(func(bool, *windowsProcessEntry32) (uintptr, error) {
		return 0, syscall.ERROR_NO_MORE_FILES
	})
	if err != nil || len(procs) != 0 {
		t.Fatal("normal empty inventory failed")
	}
}

func TestAccountRestartSnapshotFailureAfterPinDoesNotClose(t *testing.T) {
	f := newFakeAccountController()
	f.owners = f.owners[:1]
	calls := 0
	enumerate := func() ([]windowsProcess, error) {
		calls++
		return readProcessSnapshot(func(first bool, entry *windowsProcessEntry32) (uintptr, error) {
			if first && calls <= 2 {
				entry.ProcessID = 1
				copy(entry.ExeFile[:], syscall.StringToUTF16("Weixin.exe"))
				return 1, nil
			}
			if calls == 2 {
				return 0, syscall.ERROR_ACCESS_DENIED
			}
			return 0, syscall.ERROR_NO_MORE_FILES
		})
	}
	deps := f.deps()
	deps.processes = defaultAccountRestartDeps(enumerate, nil).processes
	if _, err := restartAccountWithDeps(context.Background(), SetupOptions{}, nil, nil, nil, deps); !errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
		t.Error("partial second snapshot allowed shutdown instead of preserving Win32 error")
	}
	if !slices.Equal(f.log, []string{"pin", "prepare", "release"}) {
		t.Fatalf("snapshot failure after pin had side effects or leaked handle: %v", f.log)
	}
}
