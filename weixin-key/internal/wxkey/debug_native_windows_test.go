//go:build windows && amd64

package wxkey

import (
	"errors"
	"os"
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// No app/debuggee is attached. This fixture owns one native thread whose only
// entry point is kernel32!Sleep(1); it starts suspended and never runs with our
// breakpoint installed. Existing Go runtime threads are not touched.
func newOwnedSuspendedThread(t *testing.T) (uintptr, uint32) {
	t.Helper()
	var tid uint32
	h, _, err := kernel32.NewProc("CreateThread").Call(0, 0, kernel32.NewProc("Sleep").Addr(),
		1, windows.CREATE_SUSPENDED, uintptr(unsafe.Pointer(&tid)))
	if h == 0 {
		t.Fatal(debugCallError("CreateThread fixture", err))
	}
	t.Cleanup(func() { _ = windows.CloseHandle(windows.Handle(h)) })
	return h, tid
}

type threadFixtureAPI struct{ nativeDebugAPI }

// cleanup restores an owned fixture thread; no debugger was attached here.
func (threadFixtureAPI) detach(uint32) error { return nil }
func (threadFixtureAPI) targetState(uint32) (debugTargetState, error) {
	return debugTargetState{attached: true}, nil
}

func TestNativeDebugThreadRestoreAndSuspendBalance(t *testing.T) {
	originalHandle, tid := newOwnedSuspendedThread(t)
	api := threadFixtureAPI{}
	h, err := api.openThread(uint32(os.Getpid()), tid)
	if err != nil {
		_, _ = windows.ResumeThread(windows.Handle(originalHandle))
		t.Fatal(err)
	}
	original, err := api.context(h)
	if err != nil {
		_ = api.close(h)
		_, _ = windows.ResumeThread(windows.Handle(originalHandle))
		t.Fatal(err)
	}
	// Even a failing assertion must not leave the fixture with a live DR/TF.
	t.Cleanup(func() {
		_ = api.setContext(originalHandle, original)
		_, _ = windows.ResumeThread(windows.Handle(originalHandle))
		_, _ = windows.WaitForSingleObject(windows.Handle(originalHandle), 5000)
	})
	s := newDebugSession(api, uint32(os.Getpid()), kernel32.NewProc("Sleep").Addr())
	s.threads[tid] = &debugThread{handle: h}
	if err := s.arm(s.threads[tid]); err != nil {
		_ = s.cleanup()
		t.Fatal(err)
	}
	armed, err := api.context(h)
	if err != nil || ctxField(armed, ctxOffDR0) != uint64(s.breakAddr) || ctxField(armed, ctxOffDR7)&1 == 0 {
		_ = s.cleanup()
		t.Fatalf("fixture breakpoint not armed: %v", err)
	}
	// Emulate cancellation after step-over. The thread is still suspended
	// from CreateThread; cleanup must add/remove only its OWN suspension.
	setCtxField32(armed, ctxOffEFlags, ctxField32(armed, ctxOffEFlags)|eflagsTrapFlag)
	if err := api.setContext(h, armed); err != nil {
		_ = s.cleanup()
		t.Fatal(err)
	}
	if err := s.cleanup(); err != nil {
		t.Fatal(err)
	}
	restored, err := api.context(originalHandle)
	if err != nil {
		t.Fatal(err)
	}
	assertDebugRestore(t, original, restored)
	previous, err := windows.ResumeThread(windows.Handle(originalHandle))
	if err != nil || previous != 1 {
		t.Fatalf("owned suspension unbalanced: previous=%d err=%v", previous, err)
	}
	status, err := windows.WaitForSingleObject(windows.Handle(originalHandle), 5000)
	if err != nil || status != windows.WAIT_OBJECT_0 {
		t.Fatalf("fixture did not exit normally after restore: status=%d err=%v", status, err)
	}
}

func TestNativeDebugWaitWithoutDebuggeeIsFailure(t *testing.T) {
	// No debuggee exists on this locked test thread. Windows must report its
	// actual error instead of allowing the caller to spin until its deadline.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	_, err := (nativeDebugAPI{}).wait()
	if err == nil || errors.Is(err, errDebugWaitTimeout) {
		t.Fatalf("unattached WaitForDebugEvent treated as success/timeout: %v", err)
	}
	t.Logf("unattached native wait rejected: %v", err)
}
