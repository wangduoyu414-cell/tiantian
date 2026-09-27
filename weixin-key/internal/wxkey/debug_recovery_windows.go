//go:build windows

package wxkey

import (
	"errors"
	"fmt"
	"time"
)

type debugTargetState struct {
	exited   bool
	attached bool
}

var errDebugSuspensionLost = errors.New("owned thread suspension disappeared")

func debugContextRestored(current, original *alignedContext) bool {
	for _, off := range []uintptr{ctxOffDR0, ctxOffDR1, ctxOffDR2, ctxOffDR3, ctxOffDR6, ctxOffDR7} {
		if ctxField(current, off) != ctxField(original, off) {
			return false
		}
	}
	return ctxField32(current, ctxOffEFlags)&eflagsTrapFlag == ctxField32(original, ctxOffEFlags)&eflagsTrapFlag
}

func (s *debugSession) noteCleanupError(err error) {
	if err == nil {
		return
	}
	// Repeated read-only recovery health checks cannot grow an unbounded
	// error chain. Keep original failures, not just the final retry outcome.
	if s.cleanupErrors == nil {
		s.cleanupErrors = map[string]bool{}
	}
	if len(s.cleanupErrors) < 32 && !s.cleanupErrors[err.Error()] {
		s.cleanupErrors[err.Error()] = true
		s.cleanupErr = errors.Join(s.cleanupErr, err)
	}
}

// cleanup performs ONE reconciled attempt. Failure retains the session,
// original contexts and any owned suspensions. No resume/continue/detach is
// allowed until all live modified threads have verified restored state.
func (s *debugSession) cleanup() error {
	if s.cleaned {
		return s.cleanupErr
	}
	state, err := s.api.targetState(s.pid)
	s.cleanupState, s.stateErr = state, err
	if err != nil {
		s.noteCleanupError(err)
		return s.cleanupErr
	}
	if !state.attached {
		s.detached = true
	}
	targetDead := state.exited || s.targetExit ||
		s.pending != nil && s.pending.pid == s.pid && s.pending.code == exitProcessDebugEvent
	// A retained event only proves a stop while its debug connection is
	// still attached. After an observed detach the target may be running;
	// acquire our own suspension before touching its live context.
	stopped := state.attached && !s.detached && s.pending != nil && s.pending.pid == s.pid && !s.attempted
	allRestored := true
	// First suspend/restore ALL modified threads. A failure must not release
	// previously processed threads or forget the successful suspensions.
	for _, tid := range s.threadIDs() {
		t := s.threads[tid]
		if targetDead {
			t.restored, t.suspended = true, false
			continue
		}
		if t.original == nil || t.restored {
			continue
		}
		dead, err := s.api.exited(t.handle)
		if err != nil {
			s.noteCleanupError(fmt.Errorf("thread %d liveness: %w", tid, err))
			allRestored = false
			continue
		}
		if dead {
			t.restored, t.suspended = true, false
			continue
		}
		if !stopped && !t.suspended {
			if err := s.api.suspend(t.handle); err != nil {
				s.noteCleanupError(fmt.Errorf("suspend thread %d: %w", tid, err))
				allRestored = false
				continue
			}
			t.suspended = true
		}
		if err := s.restore(t); err != nil {
			s.noteCleanupError(fmt.Errorf("restore thread %d: %w", tid, err))
		}
		allRestored = allRestored && t.restored
	}
	if !allRestored {
		return s.cleanupErr // keep EVERY owned suspension and the pending event
	}
	for _, tid := range s.threadIDs() {
		t := s.threads[tid]
		if !t.suspended {
			continue
		}
		if err := s.api.resume(t.handle); err != nil {
			s.noteCleanupError(fmt.Errorf("resume thread %d: %w", tid, err))
			if !errors.Is(err, errDebugSuspensionLost) {
				return s.cleanupErr // failed call did not decrement; retain ownership
			}
		}
		t.suspended = false
	}
	if s.pending != nil && !s.attempted && (!s.detached || targetDead && s.pending.code == exitProcessDebugEvent) {
		s.noteCleanupError(s.continuePending())
	}
	if s.targetExit {
		s.detached = true
	}
	if !s.detached {
		err := s.api.detach(s.pid)
		if err == nil {
			s.detached = true
		} else {
			s.noteCleanupError(err)
			// A failed/uncertain detach is not proof that the session is still
			// attached, or that it is gone. Query the pinned process handle.
			after, stateErr := s.api.targetState(s.pid)
			s.cleanupState, s.stateErr = after, stateErr
			s.noteCleanupError(stateErr)
			if stateErr == nil && (after.exited || !after.attached) {
				s.detached = true
			}
		}
	}
	if !s.detached {
		return s.cleanupErr // do NOT close the last handles or unlock the owner
	}
	for _, tid := range s.threadIDs() {
		s.noteCleanupError(s.api.close(s.threads[tid].handle))
		delete(s.threads, tid)
	}
	s.pending = nil
	s.cleaned = true // restoration + suspension balance + detach confirmed
	return s.cleanupErr
}

// finishCleanup must run on the attaching/creating, locked OS thread. It
// cannot return to a defer that unlocks that thread while cleanup is pending.
// Two bounded attempts may recover a transient failure, each using fresh
// pinned-process/context evidence. Persistent failures park the SAME owner;
// cancellation does not waive cleanup. Only explicit retry or observed target
// exit/detachment starts another recovery attempt; no blind write retry loop.
func (s *debugSession) finishCleanup() error {
	if s.recovery == nil {
		s.recovery = NewCaptureRecovery(nil)
	}
	requests := s.recovery.retryRequests()
	for {
		for attempt := 0; attempt < 2; attempt++ {
			s.cleanup()
			if s.cleaned {
				s.recovery.setWaiting(false, "")
				if s.cleanupErr != nil {
					return errors.Join(ErrCaptureCleanup, s.cleanupErr)
				}
				return s.cleanupErr
			}
		}
		s.recovery.setWaiting(true, fmt.Sprintf("pid=%d 调试恢复未完成，仍保留控制线程和恢复句柄；请勿强制退出工具。可重试恢复。原因：%v", s.pid, s.cleanupErr))
		// Retry automatically only on a newly observed exit/detachment, not
		// forever when an already-dead target has an unresolved API failure.
		// Anchor to the actual cleanup attempt, not a fresh baseline read
		// that could swallow an exit between cleanup and notification.
		lastState, lastErr := s.cleanupState, s.stateErr
		for {
			select {
			case <-requests:
				goto retry
			case <-time.After(500 * time.Millisecond):
				state, err := s.api.targetState(s.pid)
				s.noteCleanupError(err)
				if err == nil && (lastErr != nil || state != lastState) && (state.exited || !state.attached) {
					goto retry
				}
				lastState, lastErr = state, err
			}
		}
	retry:
	}
}
