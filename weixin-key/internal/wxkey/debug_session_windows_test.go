//go:build windows

package wxkey

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

type fakeDebugStep struct {
	event debugEvent
	err   error
	do    func()
}

type fakeDebugAPI struct {
	steps           []fakeDebugStep
	log             []string
	counts          map[string]int
	fail            map[string]map[int]error
	contexts        map[uintptr]*alignedContext
	dead            map[uintptr]bool
	statuses        []uint32
	outstanding     bool
	partialSet      bool
	attached        bool
	targetDead      bool
	additionalError func(string) error
	stateOverride   func() debugTargetState
	ignoreSet       bool
}

func newFakeDebugAPI() *fakeDebugAPI {
	f := &fakeDebugAPI{
		counts: map[string]int{}, fail: map[string]map[int]error{},
		contexts: map[uintptr]*alignedContext{}, dead: map[uintptr]bool{},
		attached: true,
	}
	f.contexts[11] = fixtureDebugContext()
	return f
}

func fixtureDebugContext() *alignedContext {
	c := newAlignedContext(contextControl | contextDebugRegisters)
	for i, off := range []uintptr{ctxOffDR0, ctxOffDR1, ctxOffDR2, ctxOffDR3, ctxOffDR6} {
		setCtxField(c, off, uint64(0x1000+i))
	}
	setCtxField(c, ctxOffDR7, 0x400) // inactive; reserved bit retained
	setCtxField32(c, ctxOffEFlags, 0x202)
	setCtxField(c, ctxOffRIP, 0x9000)
	return c
}

func cloneDebugContext(c *alignedContext) *alignedContext {
	out := newAlignedContext(contextControl | contextDebugRegisters)
	copy(out.view(), c.view())
	return out
}

func (f *fakeDebugAPI) op(name string) error {
	f.log = append(f.log, name)
	f.counts[name]++
	if f.additionalError != nil {
		if err := f.additionalError(name); err != nil {
			return err
		}
	}
	return f.fail[name][f.counts[name]]
}

func (f *fakeDebugAPI) wait() (debugEvent, error) {
	if err := f.op("wait"); err != nil {
		return debugEvent{}, err
	}
	if f.outstanding {
		return debugEvent{}, errors.New("test: waited with a pending event")
	}
	if len(f.steps) == 0 {
		return debugEvent{}, errors.New("test: exhausted events")
	}
	step := f.steps[0]
	f.steps = f.steps[1:]
	if step.do != nil {
		step.do()
	}
	if step.err == nil {
		f.outstanding = true
	}
	return step.event, step.err
}

func (f *fakeDebugAPI) continueEvent(_ debugEvent, status uint32) error {
	f.statuses = append(f.statuses, status)
	if err := f.op("continue"); err != nil {
		return err
	}
	f.outstanding = false
	return nil
}
func (f *fakeDebugAPI) detach(uint32) error {
	if err := f.op("detach"); err != nil {
		return err
	}
	f.attached = false
	return nil
}
func (f *fakeDebugAPI) openThread(_, tid uint32) (uintptr, error) {
	return uintptr(tid), f.op(fmt.Sprintf("open:%d", tid))
}
func (f *fakeDebugAPI) context(h uintptr) (*alignedContext, error) {
	if err := f.op(fmt.Sprintf("get:%d", h)); err != nil {
		return nil, err
	}
	return cloneDebugContext(f.contexts[h]), nil
}
func (f *fakeDebugAPI) setContext(h uintptr, c *alignedContext) error {
	err := f.op(fmt.Sprintf("set:%d", h))
	if !f.ignoreSet && (err == nil || f.partialSet) {
		f.contexts[h] = cloneDebugContext(c)
	}
	return err
}
func (f *fakeDebugAPI) exited(h uintptr) (bool, error) {
	return f.dead[h], f.op(fmt.Sprintf("alive:%d", h))
}
func (f *fakeDebugAPI) suspend(h uintptr) error { return f.op(fmt.Sprintf("suspend:%d", h)) }
func (f *fakeDebugAPI) resume(h uintptr) error  { return f.op(fmt.Sprintf("resume:%d", h)) }
func (f *fakeDebugAPI) close(h uintptr) error   { return f.op(fmt.Sprintf("close:%d", h)) }
func (f *fakeDebugAPI) targetState(uint32) (debugTargetState, error) {
	state := debugTargetState{exited: f.targetDead, attached: f.attached}
	if f.stateOverride != nil {
		state = f.stateOverride()
	}
	return state, f.op("target-state")
}

func debugCreate(tid uint32) debugEvent {
	return debugEvent{code: createProcessDebugEvent, pid: 7, tid: tid, file: 100}
}

func runFakeSession(s *debugSession, ctx context.Context, complete func() bool, sample func(uint32)) error {
	f := s.api.(*fakeDebugAPI)
	return runDebugSession(s, func() { _ = f.op("cancel") }, func() { _ = f.op("join") }, func() error {
		return s.pump(ctx, time.Now().Add(time.Second), complete, nil, sample)
	})
}

func assertDebugRestore(t *testing.T, original, got *alignedContext) {
	t.Helper()
	for _, off := range []uintptr{ctxOffDR0, ctxOffDR1, ctxOffDR2, ctxOffDR3, ctxOffDR6, ctxOffDR7} {
		if ctxField(original, off) != ctxField(got, off) {
			t.Errorf("debug register offset %#x not restored", off)
		}
	}
	if ctxField32(original, ctxOffEFlags)&eflagsTrapFlag != ctxField32(got, ctxOffEFlags)&eflagsTrapFlag {
		t.Error("original TF not restored")
	}
}

func TestDebugSessionCancelRestoresRunningThreadsBeforeJoin(t *testing.T) {
	f := newFakeDebugAPI()
	original := cloneDebugContext(f.contexts[11])
	ctx, cancel := context.WithCancel(context.Background())
	f.steps = []fakeDebugStep{{event: debugCreate(11), do: cancel}}
	s := newDebugSession(f, 7, 0x5000)
	err := runFakeSession(s, ctx, func() bool { return false }, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	assertDebugRestore(t, original, f.contexts[11])
	want := []string{"wait", "close:100", "open:11", "get:11", "set:11", "continue", "cancel", "target-state",
		"alive:11", "suspend:11", "get:11", "set:11", "get:11", "resume:11", "detach", "close:11", "join"}
	if !reflect.DeepEqual(f.log, want) {
		t.Fatalf("shutdown order: %v", f.log)
	}
	before := len(f.log)
	if s.cleanup() != nil || len(f.log) != before {
		t.Fatal("cleanup repeated owned operations")
	}
}

func TestDebugSessionPartialArmFailureRestoresWhileEventStopped(t *testing.T) {
	f := newFakeDebugAPI()
	original := cloneDebugContext(f.contexts[11])
	wantErr := errors.New("partial SetThreadContext failure")
	f.fail["set:11"] = map[int]error{1: wantErr}
	f.partialSet = true
	f.steps = []fakeDebugStep{{event: debugCreate(11)}}
	err := runFakeSession(newDebugSession(f, 7, 0x5000), context.Background(), func() bool { return false }, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("set failure hidden: %v", err)
	}
	assertDebugRestore(t, original, f.contexts[11])
	if f.counts["suspend:11"] != 0 || f.counts["continue"] != 1 || f.counts["detach"] != 1 {
		t.Fatalf("pending-event cleanup incorrect: %v", f.log)
	}
}

func TestDebugSessionWaitFailureIsNotTimeout(t *testing.T) {
	for _, afterArm := range []bool{false, true} {
		t.Run(fmt.Sprint(afterArm), func(t *testing.T) {
			f := newFakeDebugAPI()
			failure := errors.New("wait access failure")
			f.steps = []fakeDebugStep{{err: errDebugWaitTimeout}}
			if afterArm {
				f.steps = append(f.steps, fakeDebugStep{event: debugCreate(11)})
			}
			f.steps = append(f.steps, fakeDebugStep{err: failure})
			expectedWaits := len(f.steps)
			err := runFakeSession(newDebugSession(f, 7, 0x5000), context.Background(), func() bool { return false }, nil)
			if !errors.Is(err, failure) || f.counts["wait"] != expectedWaits || f.counts["detach"] != 1 {
				t.Fatalf("wait failure retried/hidden: %v %v", err, f.log)
			}
		})
	}
}

func TestDebugSessionContinueFailureNeverWaitsOrRetries(t *testing.T) {
	f := newFakeDebugAPI()
	failure := errors.New("continue failure")
	detachFailure := errors.New("detach failure")
	f.fail["continue"] = map[int]error{1: failure}
	f.fail["detach"] = map[int]error{1: detachFailure}
	f.steps = []fakeDebugStep{{event: debugCreate(11)}}
	err := runFakeSession(newDebugSession(f, 7, 0x5000), context.Background(), func() bool { return false }, nil)
	if !errors.Is(err, failure) || !errors.Is(err, detachFailure) ||
		f.counts["continue"] != 1 || f.counts["wait"] != 1 || f.counts["suspend:11"] != 1 || f.counts["join"] != 1 {
		t.Fatalf("failure ownership: %v %v", err, f.log)
	}
}

func TestDebugSessionCleanupFailuresDoNotBecomeSuccess(t *testing.T) {
	for _, operation := range []string{"alive:11", "suspend:11", "get:11", "set:11", "resume:11", "detach", "close:11"} {
		t.Run(operation, func(t *testing.T) {
			f := newFakeDebugAPI()
			failure := errors.New("injected " + operation)
			n := 1
			if strings.HasPrefix(operation, "get:") || strings.HasPrefix(operation, "set:") {
				n = 2 // first invocation arms, second restores
			}
			f.fail[operation] = map[int]error{n: failure}
			f.steps = []fakeDebugStep{{event: debugCreate(11)}}
			err := runFakeSession(newDebugSession(f, 7, 0x5000), context.Background(),
				func() bool { return len(f.steps) == 0 }, nil)
			wantDetach := 1
			if operation == "detach" {
				wantDetach = 2 // second attempt follows fresh attached-state evidence
			}
			if !errors.Is(err, failure) || f.counts["detach"] != wantDetach || f.counts["join"] != 1 {
				t.Fatalf("cleanup error became success or skipped remaining cleanup: %v %v", err, f.log)
			}
			if operation == "suspend:11" {
				if f.counts["suspend:11"] != 2 || f.counts["resume:11"] != 1 ||
					!strings.Contains(strings.Join(f.log, ","), "suspend:11,target-state,alive:11,suspend:11,get:11") {
					t.Fatal("failed suspension was resumed/written before a newly confirmed suspension")
				}
			}
			if (operation == "get:11" || operation == "set:11") && f.counts["resume:11"] != 1 {
				t.Fatal("context failure leaked an owned suspension")
			}
		})
	}
}

func TestDebugSessionStepRearmAndUnknownExceptions(t *testing.T) {
	f := newFakeDebugAPI()
	original := cloneDebugContext(f.contexts[11])
	s := newDebugSession(f, 7, 0x5000)
	s.startupAddr = 0x8000 // exact, synthetic, independently supplied address
	exception := func(code uint32, addr uintptr) debugEvent {
		return debugEvent{code: exceptionDebugEventCode, pid: 7, tid: 11, exception: code, address: addr, firstChance: true}
	}
	f.steps = []fakeDebugStep{
		{event: debugCreate(11)},
		{event: exception(exceptionBreakpoint, 0x8000)},
		{event: exception(exceptionBreakpoint, 0x8000)}, // not swallowed twice
		{event: exception(exceptionBreakpoint, 0x8100)}, // app's breakpoint
		{event: exception(exceptionSingleStep, 0x5000), do: func() {
			setCtxField(f.contexts[11], ctxOffDR6, 2) // someone else's DR1
		}},
		{event: exception(exceptionSingleStep, 0x5000), do: func() {
			setCtxField(f.contexts[11], ctxOffDR6, 1)
			setCtxField(f.contexts[11], ctxOffRIP, 0x5000)
		}},
		{event: exception(exceptionSingleStep, 0x5007), do: func() {
			setCtxField(f.contexts[11], ctxOffDR6, 1<<14)
			setCtxField(f.contexts[11], ctxOffRIP, 0x5007)
			// A non-TF flag changed during execution; preserve it on restore.
			setCtxField32(f.contexts[11], ctxOffEFlags, 0x347)
		}},
	}
	samples := 0
	err := runFakeSession(s, context.Background(), func() bool { return len(f.steps) == 0 }, func(uint32) { samples++ })
	if err != nil || samples != 1 {
		t.Fatalf("step state: %v samples=%d", err, samples)
	}
	want := []uint32{dbgContinue, dbgContinue, dbgExceptionNotHandled, dbgExceptionNotHandled,
		dbgExceptionNotHandled, dbgContinue, dbgContinue}
	if !reflect.DeepEqual(f.statuses, want) {
		t.Fatalf("exception ownership: %x", f.statuses)
	}
	assertDebugRestore(t, original, f.contexts[11])
	if ctxField(f.contexts[11], ctxOffRIP) != 0x5007 || ctxField32(f.contexts[11], ctxOffEFlags) != 0x247 {
		t.Fatal("restoration rewound execution or unrelated flags")
	}
}

func TestDebugSessionStepAndRearmFailuresRestoreState(t *testing.T) {
	for _, failCall := range []int{2, 3} {
		t.Run(fmt.Sprint(failCall), func(t *testing.T) {
			f := newFakeDebugAPI()
			original := cloneDebugContext(f.contexts[11])
			failure := errors.New("step/rearm failure")
			f.fail["set:11"] = map[int]error{failCall: failure}
			f.partialSet = true
			f.steps = []fakeDebugStep{
				{event: debugCreate(11)},
				{event: debugEvent{code: exceptionDebugEventCode, pid: 7, tid: 11, exception: exceptionSingleStep, address: 0x5000},
					do: func() {
						setCtxField(f.contexts[11], ctxOffDR6, 1)
						setCtxField(f.contexts[11], ctxOffRIP, 0x5000)
					}},
				{event: debugEvent{code: exceptionDebugEventCode, pid: 7, tid: 11, exception: exceptionSingleStep, address: 0x5007},
					do: func() { setCtxField(f.contexts[11], ctxOffDR6, 1<<14) }},
			}
			err := runFakeSession(newDebugSession(f, 7, 0x5000), context.Background(), func() bool { return false }, nil)
			if !errors.Is(err, failure) {
				t.Fatalf("step/rearm failure hidden: %v", err)
			}
			assertDebugRestore(t, original, f.contexts[11])
			if f.statuses[len(f.statuses)-1] != dbgContinue {
				t.Fatal("our breakpoint was delivered to application on cleanup")
			}
		})
	}
}

func TestDebugSessionConflictDoesNotWriteThread(t *testing.T) {
	for _, conflict := range []string{"TF", "DR0", "DR3", "GD"} {
		t.Run(conflict, func(t *testing.T) {
			f := newFakeDebugAPI()
			switch conflict {
			case "TF":
				setCtxField32(f.contexts[11], ctxOffEFlags, 0x302)
			case "DR0":
				setCtxField(f.contexts[11], ctxOffDR7, 1)
			case "DR3":
				setCtxField(f.contexts[11], ctxOffDR7, 1<<6)
			case "GD":
				setCtxField(f.contexts[11], ctxOffDR7, 1<<13)
			}
			f.steps = []fakeDebugStep{{event: debugCreate(11)}}
			err := runFakeSession(newDebugSession(f, 7, 0x5000), context.Background(), func() bool { return false }, nil)
			if err == nil || f.counts["set:11"] != 0 || f.counts["close:11"] != 1 {
				t.Fatalf("conflicting context modified: %v %v", err, f.log)
			}
		})
	}
}

func TestDebugSessionEventHandlesAndThreadReuse(t *testing.T) {
	f := newFakeDebugAPI()
	f.steps = []fakeDebugStep{
		{event: debugCreate(11)},
		{event: debugEvent{code: loadDLLDebugEvent, pid: 7, tid: 11, file: 101}},
		{event: debugEvent{code: exitThreadDebugEvent, pid: 7, tid: 11}},
		{event: debugEvent{code: createThreadDebugEvent, pid: 7, tid: 11}, do: func() {
			f.contexts[11] = fixtureDebugContext()
			setCtxField(f.contexts[11], ctxOffDR2, 0x12345)
		}},
	}
	err := runFakeSession(newDebugSession(f, 7, 0x5000), context.Background(), func() bool { return len(f.steps) == 0 }, nil)
	if err != nil || f.counts["close:100"] != 1 || f.counts["close:101"] != 1 ||
		f.counts["open:11"] != 2 || f.counts["close:11"] != 2 || ctxField(f.contexts[11], ctxOffDR2) != 0x12345 {
		t.Fatalf("file/thread handle ownership or TID reuse: %v %v", err, f.log)
	}
}

func TestDebugSessionExitMustBeContinued(t *testing.T) {
	for _, continueFails := range []bool{false, true} {
		t.Run(fmt.Sprint(continueFails), func(t *testing.T) {
			f := newFakeDebugAPI()
			f.steps = []fakeDebugStep{{event: debugCreate(11)}, {event: debugEvent{code: exitProcessDebugEvent, pid: 7, tid: 11}}}
			if continueFails {
				f.fail["continue"] = map[int]error{2: errors.New("continue exit failed")}
			}
			err := runFakeSession(newDebugSession(f, 7, 0x5000), context.Background(), func() bool { return false }, nil)
			wantDetach := 0
			if continueFails {
				wantDetach = 1
			}
			if err == nil || f.counts["continue"] != 2 || f.counts["set:11"] != 1 || f.counts["detach"] != wantDetach {
				t.Fatalf("exit handling: %v %v", err, f.log)
			}
		})
	}
}

func TestDebugSessionFileCloseFailureAndForeignEvent(t *testing.T) {
	for _, kind := range []string{"file-close", "foreign-process", "open-thread", "get-context"} {
		t.Run(kind, func(t *testing.T) {
			f := newFakeDebugAPI()
			ev := debugCreate(11)
			switch kind {
			case "file-close":
				f.fail["close:100"] = map[int]error{1: errors.New("hFile close failed")}
			case "foreign-process":
				ev.pid = 99
			case "open-thread":
				f.fail["open:11"] = map[int]error{1: errors.New("open failed")}
			case "get-context":
				f.fail["get:11"] = map[int]error{1: errors.New("get failed")}
			}
			f.steps = []fakeDebugStep{{event: ev}}
			err := runFakeSession(newDebugSession(f, 7, 0x5000), context.Background(), func() bool { return false }, nil)
			if err == nil || f.counts["close:100"] != 1 || f.counts["continue"] != 1 || f.counts["join"] != 1 {
				t.Fatalf("event failure leaked ownership: %v %v", err, f.log)
			}
		})
	}
}

func TestDebugSessionDeadlineBeforeWaitStillDetachesBeforeJoin(t *testing.T) {
	f := newFakeDebugAPI()
	s := newDebugSession(f, 7, 0)
	err := runDebugSession(s, func() { _ = f.op("cancel") }, func() { _ = f.op("join") }, func() error {
		return s.pump(context.Background(), time.Now().Add(-time.Second), func() bool { return false }, nil, nil)
	})
	if !errors.Is(err, errWindowsKeyScanDeadline) || strings.Join(f.log, ",") != "cancel,target-state,detach,join" {
		t.Fatalf("deadline ordering: %v %v", err, f.log)
	}
}

func TestDecodeDebugEventAMD64Ownership(t *testing.T) {
	var raw [192]byte
	binary.LittleEndian.PutUint32(raw[0:4], exceptionDebugEventCode)
	binary.LittleEndian.PutUint32(raw[4:8], 7)
	binary.LittleEndian.PutUint32(raw[8:12], 11)
	binary.LittleEndian.PutUint32(raw[16:20], exceptionBreakpoint)
	binary.LittleEndian.PutUint64(raw[32:40], 0x8877665544332211)
	binary.LittleEndian.PutUint32(raw[168:172], 1)
	ev := decodeDebugEvent(&raw)
	if ev.pid != 7 || ev.tid != 11 || !ev.firstChance || ev.exception != exceptionBreakpoint ||
		ev.address != 0x8877665544332211 || ev.file != 0 {
		t.Fatalf("exception decoding: %#v", ev)
	}
	for _, code := range []uint32{createProcessDebugEvent, loadDLLDebugEvent} {
		binary.LittleEndian.PutUint32(raw[0:4], code)
		binary.LittleEndian.PutUint64(raw[16:24], 123)
		if got := decodeDebugEvent(&raw); got.file != 123 || got.exception != 0 {
			t.Fatalf("event hFile decoding: %#v", got)
		}
	}
}

func TestDebugCallErrorNeverConvertsFailureToNil(t *testing.T) {
	if debugCallError("synthetic", nil) == nil {
		t.Fatal("failed native operation lost error")
	}
}

func TestDebugSessionDelayedModuleArmsEveryTrackedThread(t *testing.T) {
	f := newFakeDebugAPI()
	f.contexts[12] = fixtureDebugContext()
	first := cloneDebugContext(f.contexts[11])
	second := cloneDebugContext(f.contexts[12])
	f.steps = []fakeDebugStep{
		{event: debugCreate(11)},
		{event: debugEvent{code: createThreadDebugEvent, pid: 7, tid: 12}},
		{event: debugEvent{code: loadDLLDebugEvent, pid: 7, tid: 11, file: 102}},
	}
	resolveCalls := 0
	s := newDebugSession(f, 7, 0)
	err := runDebugSession(s, func() {}, func() {}, func() error {
		return s.pump(context.Background(), time.Now().Add(time.Second), func() bool { return len(f.steps) == 0 },
			func() (uintptr, error) {
				resolveCalls++
				if resolveCalls == 1 {
					return 0, nil
				}
				return 0x5000, nil
			}, nil)
	})
	if err != nil || resolveCalls != 2 || f.counts["set:11"] != 2 || f.counts["set:12"] != 2 {
		t.Fatalf("delayed module registration: %v %v", err, f.log)
	}
	assertDebugRestore(t, first, f.contexts[11])
	assertDebugRestore(t, second, f.contexts[12])
}

func TestDebugSessionCancelledWorkerJoinsOnlyAfterDetach(t *testing.T) {
	f := newFakeDebugAPI()
	ctx, cancel := context.WithCancel(context.Background())
	detached := make(chan struct{})
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		<-ctx.Done()
		<-detached // a slow in-flight verification is independent of the target
	}()
	s := newDebugSession(f, 7, 0)
	err := runDebugSession(s, cancel, func() {
		if f.counts["detach"] != 1 {
			t.Error("worker join preceded target detach")
		}
		close(detached)
		<-workerDone
	}, func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
}

func TestDebugSessionAlreadyExitedThreadDoesNotGetContextWrites(t *testing.T) {
	f := newFakeDebugAPI()
	f.steps = []fakeDebugStep{{event: debugCreate(11)}, {err: errors.New("event wait failed"), do: func() { f.dead[11] = true }}}
	err := runFakeSession(newDebugSession(f, 7, 0x5000), context.Background(), func() bool { return false }, nil)
	if err == nil || f.counts["set:11"] != 1 || f.counts["suspend:11"] != 0 || f.counts["close:11"] != 1 {
		t.Fatalf("exited handle cleanup: %v %v", err, f.log)
	}
}

func TestDebugSessionContextFailureReclassifiesBeforeRestore(t *testing.T) {
	for _, kind := range []string{"owned-dr0", "owned-tf", "foreign-single-step"} {
		t.Run(kind, func(t *testing.T) {
			f := newFakeDebugAPI()
			original := cloneDebugContext(f.contexts[11])
			s := newDebugSession(f, 7, 0x5000)
			failure := errors.New("event context read failed once")
			f.fail["get:11"] = map[int]error{2: failure}
			f.steps = []fakeDebugStep{
				{event: debugCreate(11)},
				{event: debugEvent{code: exceptionDebugEventCode, pid: 7, tid: 11, exception: exceptionSingleStep, address: 0x5000},
					do: func() {
						switch kind {
						case "owned-dr0":
							setCtxField(f.contexts[11], ctxOffDR6, 1)
							setCtxField(f.contexts[11], ctxOffRIP, 0x5000)
						case "owned-tf":
							s.threads[11].stepping = true
							setCtxField(f.contexts[11], ctxOffDR6, 1<<14)
							setCtxField32(f.contexts[11], ctxOffEFlags, 0x302)
						case "foreign-single-step":
							setCtxField(f.contexts[11], ctxOffDR6, 2)
						}
					}},
			}
			err := runFakeSession(s, context.Background(), func() bool { return false }, nil)
			if !errors.Is(err, failure) {
				t.Fatalf("initial read error was hidden: %v", err)
			}
			assertDebugRestore(t, original, f.contexts[11])
			want := uint32(dbgContinue)
			if kind == "foreign-single-step" {
				want = dbgExceptionNotHandled
			}
			if len(f.statuses) != 2 || f.statuses[1] != want {
				t.Fatalf("pending exception delivered with %#x; want %#x", f.statuses, want)
			}
		})
	}
}
