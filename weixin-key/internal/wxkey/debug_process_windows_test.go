//go:build windows && amd64

package wxkey

import (
	"errors"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// This fixture creates only System32 cmd.exe with AutoRun DISABLED and an
// explicit "exit 0". It never locates/attaches WeChat, reads process memory or
// touches an account. PROCESS_INFORMATION pins the exact child for cleanup.
func newOwnedDebugProcess(t *testing.T) (nativeDebugAPI, windows.ProcessInformation) {
	t.Helper()
	systemDir, err := windows.GetSystemDirectory()
	if err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(systemDir, "cmd.exe")
	app, _ := windows.UTF16PtrFromString(exe)
	args, _ := windows.UTF16PtrFromString(syscall.EscapeArg(exe) + " /d /q /c exit 0")
	si := windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{}))}
	var pi windows.ProcessInformation
	if err := windows.CreateProcess(app, args, nil, nil, false,
		debugOnlyThisProcess|windows.CREATE_NO_WINDOW, nil, nil, &si, &pi); err != nil {
		t.Fatal(err)
	}
	api := nativeDebugAPI{process: uintptr(pi.Process)}
	t.Cleanup(func() {
		// Only our newly-created child may be terminated on test failure.
		status, _ := windows.WaitForSingleObject(pi.Process, 0)
		if status != windows.WAIT_OBJECT_0 {
			_ = windows.TerminateProcess(pi.Process, 99)
		}
		_ = api.detach(pi.ProcessId)
		_, _ = windows.WaitForSingleObject(pi.Process, 5000)
		_ = windows.CloseHandle(pi.Thread)
		_ = windows.CloseHandle(pi.Process)
	})
	if ok, _, err := procDebugSetProcessKillOnExit.Call(0); ok == 0 {
		t.Fatal(debugCallError("fixture DebugSetProcessKillOnExit", err))
	}
	return api, pi
}

func TestNativeDebugOwnedProcessExitEvent(t *testing.T) {
	runtime.LockOSThread()
	// t.Cleanup callbacks must run before unlocking the creating OS thread.
	t.Cleanup(runtime.UnlockOSThread)
	api, pi := newOwnedDebugProcess(t)
	state, err := api.targetState(pi.ProcessId)
	if err != nil || !state.attached || state.exited {
		t.Fatalf("pinned fixture not attached: %+v err=%v", state, err)
	}
	s := newDebugSession(api, pi.ProcessId, 0)
	seenCreate, seenExit, events := false, false, 0
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		ev, err := api.wait()
		if errors.Is(err, errDebugWaitTimeout) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		events++
		if ev.pid != pi.ProcessId {
			t.Fatal("fixture received an unrelated PID")
		}
		if ev.file != 0 {
			if err := api.close(ev.file); err != nil {
				t.Fatal(err)
			}
		}
		s.pending, s.status, s.attempted, s.classified = &ev, dbgContinue, false, true
		switch ev.code {
		case createProcessDebugEvent:
			seenCreate = true
		case exceptionDebugEventCode:
			// Fixture-only loader breakpoint policy, NOT evidence for the
			// production startupAddr/profile or user-process exceptions.
			if ev.exception != exceptionBreakpoint || !ev.firstChance {
				s.status = dbgExceptionNotHandled
			}
		case exitProcessDebugEvent:
			seenExit = true
			if err := s.cleanup(); err != nil {
				t.Fatal(err)
			}
			if !s.cleaned || !s.targetExit || s.pending != nil {
				t.Fatal("EXIT_PROCESS event not continued during cleanup")
			}
		}
		if seenExit {
			break
		}
		if err := s.continuePending(); err != nil {
			t.Fatal(err)
		}
	}
	if !seenCreate || !seenExit {
		t.Fatalf("native event flow incomplete: create=%v exit=%v events=%d", seenCreate, seenExit, events)
	}
	var exitCode uint32
	if err := windows.GetExitCodeProcess(pi.Process, &exitCode); err != nil || exitCode != 0 {
		t.Fatalf("owned fixture exit=%d err=%v", exitCode, err)
	}
	t.Logf("owned child CREATE_PROCESS→EXIT_PROCESS continued, %d native events, exit=0; no account/profile evidence", events)
}
