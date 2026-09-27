//go:build windows && amd64

package wxkey

import (
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestNativeDebugPreparedPolicyCoversFutureOwnerExit(t *testing.T) {
	type result struct {
		child windows.ProcessInformation
		owner windows.Handle
		err   error
	}
	finished := make(chan result, 1)
	go func() {
		runtime.LockOSThread()
		// Intentionally no UnlockOSThread: exit this OWNED test OS thread to
		// verify survival of its next debug connection. No DR/TF is modified.
		owner, err := windows.OpenThread(windows.SYNCHRONIZE, false, windows.GetCurrentThreadId())
		if err != nil {
			finished <- result{err: err}
			return
		}
		ready, err := prepareDebuggerOwner(nil)
		if err == nil {
			err = ready.check()
		}
		var child windows.ProcessInformation
		if err == nil {
			child, err = createDebugBootstrap() // future connection, no new setter
		}
		finished <- result{child, owner, err}
	}()
	var got result
	select {
	case got = <-finished:
	case <-time.After(15 * time.Second):
		t.Fatal("owned preparation did not complete")
	}
	if got.owner != 0 {
		defer windows.CloseHandle(got.owner)
	}
	if got.child.Process != 0 {
		t.Cleanup(func() {
			// Exact test-created child only; never a reusable bare PID.
			_ = windows.TerminateProcess(got.child.Process, 99)
			_, _ = windows.WaitForSingleObject(got.child.Process, 5000)
			_ = windows.CloseHandle(got.child.Thread)
			_ = windows.CloseHandle(got.child.Process)
		})
	}
	if got.err != nil {
		t.Fatal(got.err)
	}
	if status, err := windows.WaitForSingleObject(got.owner, 5000); err != nil || status != windows.WAIT_OBJECT_0 {
		t.Fatalf("test owner OS thread did not exit: status=%d err=%v", status, err)
	}
	state, err := (nativeDebugAPI{process: uintptr(got.child.Process)}).targetState(got.child.ProcessId)
	if err != nil || state.exited || state.attached {
		t.Fatalf("future fixture did not survive/detach on owner exit: state=%+v err=%v", state, err)
	}
	if previous, err := windows.ResumeThread(got.child.Thread); err != nil || previous != 1 {
		t.Fatalf("future fixture creation suspension changed: previous=%d err=%v", previous, err)
	}
	if status, err := windows.WaitForSingleObject(got.child.Process, 5000); err != nil || status != windows.WAIT_OBJECT_0 {
		t.Fatalf("surviving fixture did not exit after resume: status=%d err=%v", status, err)
	}
	var exitCode uint32
	if err := windows.GetExitCodeProcess(got.child.Process, &exitCode); err != nil || exitCode != 0 {
		t.Fatalf("surviving fixture exit=%d err=%v", exitCode, err)
	}
	t.Log("prepared on owned bootstrap; future debug child survived actual owner OS-thread exit, detached, resumed once, exit=0")
}

type bootstrapFaultAPI struct {
	nativeDebugAPI
	failing *atomic.Bool
	failure error
}

func (f bootstrapFaultAPI) detach(pid uint32) error {
	if f.failing.Load() {
		return f.failure
	}
	return f.nativeDebugAPI.detach(pid)
}

func TestNativeDebugPreparationDoubleFailureNeverReturnsReadiness(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	policyFailure, detachFailure := errors.New("synthetic policy failure"), errors.New("synthetic bootstrap detach failure")
	var failing atomic.Bool
	failing.Store(true)
	owner := windows.GetCurrentThreadId()
	var notified bool
	recovery := NewCaptureRecovery(nil)
	recovery.notify = func(waiting bool, _ string) {
		if waiting {
			notified = true
			if windows.GetCurrentThreadId() != owner {
				t.Error("bootstrap recovery migrated OS owner")
			}
			failing.Store(false)
			recovery.Retry() // explicit fixture action after observing held state
		}
	}
	ready, err := prepareDebuggerOwnerWith(recovery, func() error { return policyFailure },
		func(api nativeDebugAPI) debugAPI { return bootstrapFaultAPI{api, &failing, detachFailure} })
	if ready != nil || !notified || !errors.Is(err, policyFailure) || !errors.Is(err, detachFailure) ||
		!errors.Is(err, ErrCaptureCleanup) || windows.GetCurrentThreadId() != owner {
		t.Fatalf("double failure lost error/owner or returned readiness: ready=%v notified=%v err=%v", ready, notified, err)
	}
	if waiting, _ := recovery.State(); waiting {
		t.Fatal("bootstrap remained marked pending after confirmed cleanup")
	}
}

func TestDebugReadinessRejectsAbsentOrForeignOwner(t *testing.T) {
	if (*debugOwnerReady)(nil).check() == nil || (&debugOwnerReady{}).check() == nil ||
		(&debugOwnerReady{threadID: ^windows.GetCurrentThreadId()}).check() == nil {
		t.Fatal("invalid debugger owner accepted")
	}
}
