//go:build windows

package wxkey

import (
	"context"
	"errors"
	"runtime"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func armedRecoveryFixture(t *testing.T) (*debugSession, *fakeDebugAPI, *alignedContext) {
	t.Helper()
	f := newFakeDebugAPI()
	original := cloneDebugContext(f.contexts[11])
	s := newDebugSession(f, 7, 0x5000)
	s.threads[11] = &debugThread{handle: 11}
	if err := s.arm(s.threads[11]); err != nil {
		t.Fatal(err)
	}
	return s, f, original
}

func TestDebugRecoveryFailureRetainsRestorationOwnership(t *testing.T) {
	for _, operation := range []string{"alive:11", "suspend:11", "get:11", "set:11"} {
		t.Run(operation, func(t *testing.T) {
			s, f, _ := armedRecoveryFixture(t)
			failure := errors.New("persistent recovery failure")
			// Fail successive calls too: no immediate retry may disguise the
			// safety property. This fixture never runs a real target.
			f.fail[operation] = map[int]error{}
			for n := f.counts[operation] + 1; n < 20; n++ {
				f.fail[operation][n] = failure
			}
			err := s.cleanup()
			if !errors.Is(err, failure) {
				t.Fatalf("original recovery error lost: %v", err)
			}
			if s.cleaned || s.threads[11] == nil || f.counts["resume:11"] != 0 ||
				f.counts["detach"] != 0 || f.counts["close:11"] != 0 || !f.attached {
				t.Fatalf("released unsafe target or lost restoration ownership: cleaned=%v log=%v", s.cleaned, f.log)
			}
		})
	}
}

func TestDebugRecoveryDetachFailureRetainsOwner(t *testing.T) {
	s, f, original := armedRecoveryFixture(t)
	failure := errors.New("detach failed while still attached")
	f.fail["detach"] = map[int]error{1: failure}
	err := s.cleanup()
	if !errors.Is(err, failure) {
		t.Fatalf("detach error lost: %v", err)
	}
	assertDebugRestore(t, original, f.contexts[11])
	if s.cleaned || s.threads[11] == nil || !f.attached || f.counts["close:11"] != 0 {
		t.Fatalf("still-attached owner released: cleaned=%v log=%v", s.cleaned, f.log)
	}
	// A fresh reconciliation cycle may finish the failed detach, not hide
	// the original failure or repeat context writes on a running thread.
	sets := f.counts["set:11"]
	err = s.cleanup()
	if !s.cleaned || f.attached || s.threads[11] != nil || f.counts["set:11"] != sets ||
		!errors.Is(err, failure) {
		t.Fatalf("reconciled detach lost error/state: cleaned=%v err=%v log=%v", s.cleaned, err, f.log)
	}
}

func TestDebugRecoveryReadbackIsMandatory(t *testing.T) {
	s, f, _ := armedRecoveryFixture(t)
	f.ignoreSet = true // native boolean success is not a readback guarantee
	err := s.cleanup()
	if err == nil || s.cleaned || s.threads[11].restored || !s.threads[11].suspended ||
		f.counts["resume:11"] != 0 || f.counts["detach"] != 0 {
		t.Fatalf("unverified write released target: err=%v log=%v", err, f.log)
	}
}

func TestDebugRecoveryFailedWriteAlreadyAppliedIsReconciled(t *testing.T) {
	s, f, original := armedRecoveryFixture(t)
	failure := errors.New("write applied but reported failure")
	f.fail["set:11"] = map[int]error{2: failure}
	f.partialSet = true
	err := s.cleanup()
	if !errors.Is(err, failure) || !s.cleaned || f.counts["set:11"] != 2 || f.counts["resume:11"] != 1 {
		t.Fatalf("readback did not reconcile uncertain write: %v %v", err, f.log)
	}
	assertDebugRestore(t, original, f.contexts[11])
}

func TestDebugRecoveryNoThreadReleasedUntilAllRestored(t *testing.T) {
	s, f, _ := armedRecoveryFixture(t)
	f.contexts[12] = fixtureDebugContext()
	s.threads[12] = &debugThread{handle: 12}
	if err := s.arm(s.threads[12]); err != nil {
		t.Fatal(err)
	}
	f.fail["set:12"] = map[int]error{2: errors.New("second thread restore failed")}
	if err := s.cleanup(); err == nil || s.cleaned || !s.threads[11].restored || s.threads[12].restored ||
		f.counts["resume:11"] != 0 || f.counts["resume:12"] != 0 || f.counts["detach"] != 0 {
		t.Fatalf("first thread released before second restored: %v", f.log)
	}
	if err := s.cleanup(); err == nil || !s.cleaned || f.counts["suspend:11"] != 1 ||
		f.counts["suspend:12"] != 1 || f.counts["resume:11"] != 1 || f.counts["resume:12"] != 1 {
		t.Fatalf("retained suspensions were reacquired/lost: %v", f.log)
	}
}

func TestDebugRecoveryPersistentFailureKeepsSameOSThreadOwner(t *testing.T) {
	for _, operation := range []string{"set:11", "detach"} {
		t.Run(operation, func(t *testing.T) {
			s, f, original := armedRecoveryFixture(t)
			failure := errors.New("persistent owned fixture failure")
			var failing atomic.Bool
			failing.Store(true)
			f.additionalError = func(name string) error {
				if name == operation && failing.Load() {
					return failure
				}
				return nil
			}
			type snapshot struct {
				threadID                        uint32
				cleaned                         bool
				retained                        bool
				attached                        bool
				sets, detaches, resumes, closes int
			}
			parked := make(chan snapshot, 1)
			s.recovery = NewCaptureRecovery(func(waiting bool, _ string) {
				if waiting {
					parked <- snapshot{windows.GetCurrentThreadId(), s.cleaned, s.threads[11] != nil,
						f.attached, f.counts["set:11"], f.counts["detach"], f.counts["resume:11"], f.counts["close:11"]}
				}
			})
			type outcome struct {
				err           error
				before, after uint32
				joined        bool
			}
			done := make(chan outcome, 1)
			t.Cleanup(func() {
				failing.Store(false)
				s.recovery.Retry()
			})
			go func() {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				before := windows.GetCurrentThreadId()
				joined := false
				err := runDebugSession(s, func() {}, func() { joined = true }, func() error { return context.Canceled })
				done <- outcome{err, before, windows.GetCurrentThreadId(), joined}
			}()
			var held snapshot
			select {
			case held = <-parked:
			case <-time.After(3 * time.Second):
				t.Fatal("persistent recovery did not report parked owner")
			}
			if held.cleaned || !held.retained || !held.attached || held.closes != 0 {
				t.Fatalf("persistent failure lost owner: %#v", held)
			}
			if operation == "set:11" && (held.resumes != 0 || held.detaches != 0) {
				t.Fatalf("unrestored target released: %#v", held)
			}
			select {
			case <-done:
				t.Fatal("capture returned/joined while recovery still pending")
			default:
			}
			// Atomic fault-state change + explicit request: no blind retry.
			failing.Store(false)
			if !s.recovery.Retry() {
				t.Fatal("parked owner rejected its job-scoped recovery request")
			}
			select {
			case got := <-done:
				if !errors.Is(got.err, failure) || !errors.Is(got.err, context.Canceled) || !got.joined ||
					got.before != held.threadID || got.after != held.threadID || !s.cleaned || f.attached {
					t.Fatalf("owner/initial failure lost across retry: %#v err=%v", got, got.err)
				}
				assertDebugRestore(t, original, f.contexts[11])
			case <-time.After(3 * time.Second):
				t.Fatal("explicit recovery did not release owner")
			}
		})
	}
}

func TestDebugRecoveryExitDuringWaitingNotificationIsNotMissed(t *testing.T) {
	s, f, _ := armedRecoveryFixture(t)
	f.additionalError = func(name string) error {
		if name == "get:11" {
			return errors.New("synthetic persistent read failure")
		}
		return nil
	}
	s.recovery = NewCaptureRecovery(func(waiting bool, _ string) {
		if waiting {
			// Exit occurs after the final cleanup observation, but before
			// finishCleanup starts health polling. No user retry is sent.
			f.targetDead, f.attached = true, false
		}
	})
	done := make(chan error, 1)
	go func() { done <- s.finishCleanup() }()
	t.Cleanup(func() { s.recovery.Retry() })
	select {
	case err := <-done:
		if !s.cleaned || !errors.Is(err, ErrCaptureCleanup) {
			t.Fatalf("exit lost failure/cleanup state: cleaned=%v err=%v", s.cleaned, err)
		}
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("exit in the cleanup→waiting gap was missed; owner still parked")
	}
}

func TestDebugRecoveryDetachedPendingEventIsNotStopEvidence(t *testing.T) {
	for _, suspendFails := range []bool{false, true} {
		t.Run(map[bool]string{false: "restores-under-owned-suspension", true: "failed-suspend-retains-owner"}[suspendFails], func(t *testing.T) {
			s, f, original := armedRecoveryFixture(t)
			ev := debugCreate(11)
			s.pending = &ev
			f.outstanding = true
			// Fresh pinned-process evidence says the former event's debug
			// connection is gone. The target is alive and may be running.
			f.attached = false
			f.log = nil
			if suspendFails {
				f.fail["suspend:11"] = map[int]error{1: errors.New("synthetic suspend failure")}
			}
			err := s.cleanup()
			if f.counts["suspend:11"] != 1 || f.counts["continue"] != 0 || f.counts["detach"] != 0 {
				t.Fatalf("stale pending event reused as stop/continue proof: %v", f.log)
			}
			if suspendFails {
				if err == nil || s.cleaned || slices.Contains(f.log, "get:11") ||
					f.counts["resume:11"] != 0 || f.counts["close:11"] != 0 {
					t.Fatalf("failed suspend released/lost recovery ownership: err=%v log=%v", err, f.log)
				}
				return
			}
			get, suspend, resume := slices.Index(f.log, "get:11"), slices.Index(f.log, "suspend:11"), slices.Index(f.log, "resume:11")
			if err != nil || !s.cleaned || suspend < 0 || get <= suspend || resume <= get ||
				f.counts["resume:11"] != 1 || f.counts["close:11"] != 1 {
				t.Fatalf("detached recovery did not establish/balance its own stop: err=%v log=%v", err, f.log)
			}
			assertDebugRestore(t, original, f.contexts[11])
		})
	}
}
