//go:build windows

package wxkey

import (
	"errors"
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// A readiness value is local to the locked OS thread that established the
// policy. It is not a profile approval and cannot bypass the capture gate.
type debugOwnerReady struct{ threadID uint32 }

func (r *debugOwnerReady) check() error {
	if r == nil || r.threadID == 0 || r.threadID != windows.GetCurrentThreadId() {
		return errors.New("debugger survival policy was not prepared on this OS thread")
	}
	return nil
}

// The inert bootstrap is OUR process, not a user application. /d disables cmd
// AutoRun; CREATE_SUSPENDED prevents its exit before policy establishment.
func createDebugBootstrap() (windows.ProcessInformation, error) {
	var pi windows.ProcessInformation
	systemDir, err := windows.GetSystemDirectory()
	if err != nil {
		return pi, err
	}
	exe := filepath.Join(systemDir, "cmd.exe")
	app, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return pi, err
	}
	args, err := windows.UTF16PtrFromString(syscall.EscapeArg(exe) + " /d /q /c exit 0")
	if err != nil {
		return pi, err
	}
	si := windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{}))}
	err = windows.CreateProcess(app, args, nil, nil, false,
		debugOnlyThisProcess|windows.CREATE_SUSPENDED|windows.CREATE_NO_WINDOW, nil, nil, &si, &pi)
	return pi, err
}

// prepareDebuggerOwner must be called while locked, BEFORE any account close,
// target attach or target launch. Windows requires an existing connection to
// set this per-thread policy, which then covers future connections.
func prepareDebuggerOwner(recovery *CaptureRecovery) (*debugOwnerReady, error) {
	return prepareDebuggerOwnerWith(recovery, func() error {
		if ok, _, err := procDebugSetProcessKillOnExit.Call(0); ok == 0 {
			return debugCallError("DebugSetProcessKillOnExit preparation", err)
		}
		return nil
	}, func(api nativeDebugAPI) debugAPI { return api })
}

// Injection only changes the policy/cleanup adapter for owned-fixture tests.
// No target-app argument is accepted here.
func prepareDebuggerOwnerWith(recovery *CaptureRecovery, setPolicy func() error,
	wrap func(nativeDebugAPI) debugAPI) (ready *debugOwnerReady, err error) {
	owner := windows.GetCurrentThreadId()
	pi, err := createDebugBootstrap()
	if err != nil {
		return nil, fmt.Errorf("create owned debugger bootstrap: %w", err)
	}
	defer func() {
		err = errors.Join(err, windows.CloseHandle(pi.Thread), windows.CloseHandle(pi.Process))
		if err != nil {
			ready = nil
		}
	}()
	s := newDebugSession(wrap(nativeDebugAPI{process: uintptr(pi.Process)}), pi.ProcessId, 0)
	if recovery != nil {
		s.recovery = recovery
	}
	// Register owner-preserving cleanup before calling any injected function.
	defer func() {
		err = errors.Join(err, s.finishCleanup())
		// After confirmed detach, release just the bootstrap's creation
		// suspension. No code/DR/TF was ever modified on this helper.
		previous, resumeErr := windows.ResumeThread(pi.Thread)
		if resumeErr != nil || previous != 1 {
			err = errors.Join(err, fmt.Errorf("bootstrap resume count=%d", previous), resumeErr)
		}
		status, waitErr := windows.WaitForSingleObject(pi.Process, 5000)
		if waitErr != nil || status != windows.WAIT_OBJECT_0 {
			err = errors.Join(err, fmt.Errorf("bootstrap exit not confirmed, wait=%d", status), waitErr)
			// Only this exact, newly-created inert helper may be terminated.
			// There is never a process-name enumeration or user PID here.
			err = errors.Join(err, windows.TerminateProcess(pi.Process, 99))
			status, waitErr = windows.WaitForSingleObject(pi.Process, 5000)
			err = errors.Join(err, waitErr)
			if status != windows.WAIT_OBJECT_0 {
				err = errors.Join(err, fmt.Errorf("terminated bootstrap exit still unconfirmed, wait=%d", status))
			}
		} else {
			var code uint32
			if codeErr := windows.GetExitCodeProcess(pi.Process, &code); codeErr != nil || code != 0 {
				err = errors.Join(err, fmt.Errorf("bootstrap exit code=%d", code), codeErr)
			}
		}
		if err != nil {
			ready = nil
		}
	}()
	if err = setPolicy(); err != nil {
		return nil, err
	}
	return &debugOwnerReady{threadID: owner}, nil
}
