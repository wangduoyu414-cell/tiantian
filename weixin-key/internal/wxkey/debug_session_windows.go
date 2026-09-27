//go:build windows

package wxkey

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
)

const (
	exitThreadDebugEvent = 4
	ctxOffDR1            = 0x50
	ctxOffDR2            = 0x58
	ctxOffDR3            = 0x60
	ctxOffDR6            = 0x68
	ctxOffRIP            = 0xF8
	debugStatusCauses    = 0xF | 1<<13 | 1<<14 | 1<<15
)

var errDebugWaitTimeout = errors.New("debug event wait timeout")

// Only hFile is owned by the debugger in CREATE_PROCESS/LOAD_DLL events.
// Event process/thread handles are closed by Windows after the exit event is
// continued. Thread handles below are separately opened and owned by us.
type debugEvent struct {
	code, pid, tid uint32
	exception      uint32
	address        uintptr
	firstChance    bool
	file           uintptr
}

type debugAPI interface {
	wait() (debugEvent, error)
	continueEvent(debugEvent, uint32) error
	detach(uint32) error
	openThread(uint32, uint32) (uintptr, error)
	context(uintptr) (*alignedContext, error)
	setContext(uintptr, *alignedContext) error
	exited(uintptr) (bool, error)
	suspend(uintptr) error
	resume(uintptr) error
	close(uintptr) error
	targetState(uint32) (debugTargetState, error)
}

type debugThread struct {
	handle    uintptr // pins this thread identity, not a reusable TID
	original  *alignedContext
	stepping  bool
	restored  bool
	suspended bool
}

type debugSession struct {
	api           debugAPI
	pid           uint32
	breakAddr     uintptr
	threads       map[uint32]*debugThread
	pending       *debugEvent
	status        uint32
	attempted     bool // never blindly retry a failed ContinueDebugEvent
	classified    bool // false is UNKNOWN, not evidence of a foreign exception
	targetExit    bool
	cleaned       bool
	cleanupErr    error
	cleanupErrors map[string]bool
	detached      bool
	recovery      *CaptureRecovery
	cleanupState  debugTargetState // last observation from an actual attempt
	stateErr      error

	// A caller must supply a proven loader breakpoint address to swallow it.
	// A generic STATUS_BREAKPOINT or "the first one" is NOT sufficient.
	startupAddr uintptr
	startupSeen bool
}

func newDebugSession(api debugAPI, pid uint32, breakAddr uintptr) *debugSession {
	return &debugSession{api: api, pid: pid, breakAddr: breakAddr, threads: map[uint32]*debugThread{}, recovery: NewCaptureRecovery(nil)}
}

func (s *debugSession) threadIDs() []uint32 {
	ids := make([]uint32, 0, len(s.threads))
	for tid := range s.threads {
		ids = append(ids, tid)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func (s *debugSession) arm(t *debugThread) error {
	if t.original != nil || s.breakAddr == 0 {
		return nil
	}
	c, err := s.api.context(t.handle)
	if err != nil {
		return err
	}
	// Do not overwrite another debugger's breakpoints, GD or trap flag.
	if ctxField(c, ctxOffDR7)&(0xFF|1<<13) != 0 || ctxField32(c, ctxOffEFlags)&eflagsTrapFlag != 0 {
		return errors.New("debug register/trap flag conflict; refusing to arm")
	}
	saved := newAlignedContext(contextControl | contextDebugRegisters)
	copy(saved.view(), c.view())
	t.original = saved // retain BEFORE SetThreadContext, including partial failure
	setCtxField(c, ctxOffDR0, uint64(s.breakAddr))
	setCtxField(c, ctxOffDR7, (ctxField(c, ctxOffDR7)&^(uint64(0xF)<<16))|1)
	setCtxField(c, ctxOffDR6, ctxField(c, ctxOffDR6)&^uint64(debugStatusCauses))
	return s.api.setContext(t.handle, c)
}

func (s *debugSession) restore(t *debugThread) error {
	if t.original == nil {
		return nil
	}
	c, err := s.api.context(t.handle)
	if err != nil {
		return err
	}
	// An event-time GetThreadContext failure leaves exception ownership
	// UNKNOWN. Recovery may now have obtained the missing evidence; classify
	// it BEFORE restoring DR6/TF destroys that evidence.
	if s.pending != nil && !s.attempted && !s.classified &&
		s.pending.pid == s.pid && s.threads[s.pending.tid] == t {
		s.classifySingleStep(*s.pending, t, c)
	}
	if debugContextRestored(c, t.original) {
		t.restored = true
		return nil
	}
	for _, off := range []uintptr{ctxOffDR0, ctxOffDR1, ctxOffDR2, ctxOffDR3, ctxOffDR6, ctxOffDR7} {
		setCtxField(c, off, ctxField(t.original, off))
	}
	// Keep current RIP/RSP/other flags, not an old execution snapshot.
	flags := ctxField32(c, ctxOffEFlags)&^uint32(eflagsTrapFlag) |
		ctxField32(t.original, ctxOffEFlags)&eflagsTrapFlag
	setCtxField32(c, ctxOffEFlags, flags)
	setErr := s.api.setContext(t.handle, c)
	// A failed write may have partially applied. Never release the target on
	// the strength of either a boolean success or an error alone.
	after, readErr := s.api.context(t.handle)
	if readErr != nil {
		return errors.Join(setErr, readErr)
	}
	t.restored = debugContextRestored(after, t.original)
	if !t.restored {
		return errors.Join(setErr, errors.New("debug context restoration readback mismatch"))
	}
	return setErr
}

func (s *debugSession) classifySingleStep(ev debugEvent, t *debugThread, c *alignedContext) bool {
	s.classified = true
	if ev.code != exceptionDebugEventCode || ev.exception != exceptionSingleStep {
		return false
	}
	cause := ctxField(c, ctxOffDR6) & debugStatusCauses
	owned := t.stepping && cause == 1<<14 && ctxField32(c, ctxOffEFlags)&eflagsTrapFlag != 0 ||
		!t.stepping && cause == 1 && ev.address == s.breakAddr && ctxField(c, ctxOffRIP) == uint64(s.breakAddr)
	if owned {
		s.status = dbgContinue
	}
	return owned
}

func (s *debugSession) continuePending() error {
	if s.pending == nil {
		return nil
	}
	if s.attempted {
		return errors.New("debug event continuation already failed; not retried")
	}
	s.attempted = true
	if err := s.api.continueEvent(*s.pending, s.status); err != nil {
		return err
	}
	if s.pending.code == exitProcessDebugEvent && s.pending.pid == s.pid {
		s.targetExit = true
	}
	s.pending = nil
	s.attempted = false
	return nil
}

// handle executes only while the event has stopped the target. resolve must
// be bounded and memory-only; sampling must not perform KDF or filesystem IO.
func (s *debugSession) handle(ev debugEvent, resolve func() (uintptr, error), sample func(uint32)) error {
	if ev.pid != s.pid {
		return errors.New("unexpected process in single-target debug session")
	}
	switch ev.code {
	case createProcessDebugEvent, createThreadDebugEvent:
		if _, exists := s.threads[ev.tid]; exists {
			return errors.New("duplicate thread creation event")
		}
		if len(s.threads) >= 4096 {
			return errors.New("debug thread tracking limit reached")
		}
		h, err := s.api.openThread(s.pid, ev.tid)
		if err != nil {
			return err
		}
		s.threads[ev.tid] = &debugThread{handle: h}
	case exitThreadDebugEvent:
		if t := s.threads[ev.tid]; t != nil {
			delete(s.threads, ev.tid)
			return s.api.close(t.handle)
		}
	case exitProcessDebugEvent:
		// All threads have exited. Continue is still mandatory to let Windows
		// release the event's system-managed process/thread handles.
		return errors.New("debug target exited before capture completed")
	}
	if s.breakAddr == 0 && resolve != nil && (ev.code == createProcessDebugEvent || ev.code == loadDLLDebugEvent) {
		addr, err := resolve()
		if err != nil {
			return err
		}
		s.breakAddr = addr
	}
	if ev.code == createProcessDebugEvent || ev.code == createThreadDebugEvent || ev.code == loadDLLDebugEvent {
		for _, tid := range s.threadIDs() {
			if err := s.arm(s.threads[tid]); err != nil {
				return fmt.Errorf("arm thread %d: %w", tid, err)
			}
		}
	}
	if ev.code != exceptionDebugEventCode {
		return nil
	}
	if ev.exception == exceptionBreakpoint && ev.firstChance && !s.startupSeen &&
		s.startupAddr != 0 && ev.address == s.startupAddr {
		s.startupSeen = true
		s.status = dbgContinue
		s.classified = true
		return nil
	}
	t := s.threads[ev.tid]
	if ev.exception != exceptionSingleStep || t == nil || t.original == nil {
		s.classified = true
		return nil // default is DBG_EXCEPTION_NOT_HANDLED
	}
	c, err := s.api.context(t.handle)
	if err != nil {
		return err
	}
	if !s.classifySingleStep(ev, t, c) {
		return nil // confirmed not solely our DR0/TF event
	}
	if t.stepping {
		setCtxField32(c, ctxOffEFlags, ctxField32(c, ctxOffEFlags)&^uint32(eflagsTrapFlag))
		setCtxField(c, ctxOffDR0, uint64(s.breakAddr))
		setCtxField(c, ctxOffDR7, (ctxField(c, ctxOffDR7)&^(uint64(0xF)<<16))|1)
		setCtxField(c, ctxOffDR6, ctxField(c, ctxOffDR6)&^uint64(debugStatusCauses))
		if err := s.api.setContext(t.handle, c); err != nil {
			return fmt.Errorf("rearm: %w", err)
		}
		t.stepping = false
	} else {
		setCtxField(c, ctxOffDR7, ctxField(c, ctxOffDR7)&^uint64(1))
		setCtxField(c, ctxOffDR6, ctxField(c, ctxOffDR6)&^uint64(debugStatusCauses))
		setCtxField32(c, ctxOffEFlags, ctxField32(c, ctxOffEFlags)|eflagsTrapFlag)
		// Establish the step BEFORE accepting samples. Cleanup restores the
		// original state if SetThreadContext fails.
		if err := s.api.setContext(t.handle, c); err != nil {
			return fmt.Errorf("step over: %w", err)
		}
		t.stepping = true
		if sample != nil {
			sample(ev.tid)
		}
	}
	return nil
}

func (s *debugSession) pump(ctx context.Context, deadline time.Time, complete func() bool,
	resolve func() (uintptr, error), sample func(uint32)) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !deadline.IsZero() && !time.Now().Before(deadline) {
			return errWindowsKeyScanDeadline
		}
		if complete() {
			return nil
		}
		ev, err := s.api.wait()
		if errors.Is(err, errDebugWaitTimeout) {
			continue
		}
		if err != nil {
			return err
		}
		s.pending, s.status, s.attempted, s.classified = &ev, dbgContinue, false, true
		if ev.code == exceptionDebugEventCode {
			s.status = dbgExceptionNotHandled
			s.classified = false
		}
		// Close the only debugger-owned event handle, exactly once, including
		// all error/cancel/unknown-PID branches after receiving the event.
		if (ev.code == createProcessDebugEvent || ev.code == loadDLLDebugEvent) && ev.file != 0 {
			if err := s.api.close(ev.file); err != nil {
				return fmt.Errorf("debug event hFile: %w", err)
			}
		}
		if err := s.handle(ev, resolve, sample); err != nil {
			return err
		}
		if err := s.continuePending(); err != nil {
			return err
		}
	}
}

// Shutdown order is part of the contract: cancel verification, restore and
// detach the target, then join the worker before setupScan's context can change.
func runDebugSession(s *debugSession, cancel, join func(), run func() error) (err error) {
	defer func() {
		cancel()
		err = errors.Join(err, s.finishCleanup())
		join()
	}()
	return run()
}
