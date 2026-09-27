//go:build windows

package wxkey

import (
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type nativeDebugAPI struct {
	process uintptr // borrowed pinned handle; caller closes only AFTER finishCleanup
}

func (api nativeDebugAPI) targetState(pid uint32) (debugTargetState, error) {
	if api.process == 0 {
		return debugTargetState{}, errors.New("debugger recovery has no pinned process handle")
	}
	actual, _, err := kernel32.NewProc("GetProcessId").Call(api.process)
	if actual == 0 || actual != uintptr(pid) {
		return debugTargetState{}, debugCallError("debug process identity", err)
	}
	status, waitErr := windows.WaitForSingleObject(windows.Handle(api.process), 0)
	if waitErr != nil {
		return debugTargetState{}, waitErr
	}
	if status == windows.WAIT_OBJECT_0 {
		return debugTargetState{exited: true}, nil
	}
	if status != uint32(windows.WAIT_TIMEOUT) {
		return debugTargetState{}, fmt.Errorf("debug process wait returned %d", status)
	}
	var attached int32
	if ok, _, err := kernel32.NewProc("CheckRemoteDebuggerPresent").Call(api.process, uintptr(unsafe.Pointer(&attached))); ok == 0 {
		return debugTargetState{}, debugCallError("CheckRemoteDebuggerPresent", err)
	}
	return debugTargetState{attached: attached != 0}, nil
}

func debugCallError(name string, err error) error {
	if err == nil || errors.Is(err, syscall.Errno(0)) {
		return fmt.Errorf("%s failed without a Win32 error", name)
	}
	return fmt.Errorf("%s: %w", name, err)
}

func decodeDebugEvent(raw *[192]byte) debugEvent {
	ev := debugEvent{
		code: binary.LittleEndian.Uint32(raw[0:4]),
		pid:  binary.LittleEndian.Uint32(raw[4:8]),
		tid:  binary.LittleEndian.Uint32(raw[8:12]),
	}
	switch ev.code {
	case exceptionDebugEventCode:
		ev.exception = binary.LittleEndian.Uint32(raw[16:20])
		ev.address = uintptr(binary.LittleEndian.Uint64(raw[32:40]))
		ev.firstChance = binary.LittleEndian.Uint32(raw[168:172]) != 0
	case createProcessDebugEvent, loadDLLDebugEvent:
		ev.file = uintptr(binary.LittleEndian.Uint64(raw[16:24]))
	}
	return ev
}

func (nativeDebugAPI) wait() (debugEvent, error) {
	if runtime.GOARCH != "amd64" {
		return debugEvent{}, errors.New("debug event/context ABI is only implemented for windows/amd64")
	}
	var raw [192]byte
	ok, _, err := procWaitForDebugEvent.Call(uintptr(unsafe.Pointer(&raw[0])), 100)
	if ok == 0 {
		// WaitForDebugEvent reports ERROR_SEM_TIMEOUT (121), not a generic
		// failure/zero GetLastError or WaitForSingleObject's WAIT_TIMEOUT.
		if errors.Is(err, windows.ERROR_SEM_TIMEOUT) {
			return debugEvent{}, errDebugWaitTimeout
		}
		return debugEvent{}, debugCallError("WaitForDebugEvent", err)
	}
	return decodeDebugEvent(&raw), nil
}

func (nativeDebugAPI) continueEvent(ev debugEvent, status uint32) error {
	if ok, _, err := procContinueDebugEvent.Call(uintptr(ev.pid), uintptr(ev.tid), uintptr(status)); ok == 0 {
		return debugCallError("ContinueDebugEvent", err)
	}
	return nil
}

func (nativeDebugAPI) detach(pid uint32) error {
	if ok, _, err := procDebugActiveProcessStop.Call(uintptr(pid)); ok == 0 {
		return debugCallError("DebugActiveProcessStop", err)
	}
	return nil
}

func (nativeDebugAPI) openThread(pid, tid uint32) (uintptr, error) {
	h, err := windows.OpenThread(windows.THREAD_GET_CONTEXT|windows.THREAD_SET_CONTEXT|
		windows.THREAD_SUSPEND_RESUME|windows.THREAD_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, tid)
	if err != nil {
		return 0, fmt.Errorf("OpenThread: %w", err)
	}
	actual, _, callErr := kernel32.NewProc("GetProcessIdOfThread").Call(uintptr(h))
	if actual == 0 || actual != uintptr(pid) {
		return 0, errors.Join(debugCallError("thread process identity", callErr), windows.CloseHandle(h))
	}
	return uintptr(h), nil
}

func (nativeDebugAPI) context(h uintptr) (*alignedContext, error) {
	c := newAlignedContext(contextControl | contextDebugRegisters)
	if ok, _, err := procGetThreadContext.Call(h, uintptr(c.basePtr())); ok == 0 {
		return nil, debugCallError("GetThreadContext", err)
	}
	return c, nil
}

func (nativeDebugAPI) setContext(h uintptr, c *alignedContext) error {
	if ok, _, err := procSetThreadContext.Call(h, uintptr(c.basePtr())); ok == 0 {
		return debugCallError("SetThreadContext", err)
	}
	return nil
}

func (nativeDebugAPI) exited(h uintptr) (bool, error) {
	status, err := windows.WaitForSingleObject(windows.Handle(h), 0)
	if err != nil {
		return false, err
	}
	switch status {
	case windows.WAIT_OBJECT_0:
		return true, nil
	case uint32(windows.WAIT_TIMEOUT):
		return false, nil
	default:
		return false, fmt.Errorf("thread wait returned unexpected status %d", status)
	}
}

func (nativeDebugAPI) suspend(h uintptr) error {
	if n, _, err := kernel32.NewProc("SuspendThread").Call(h); uint32(n) == ^uint32(0) {
		return debugCallError("SuspendThread", err)
	}
	return nil
}

func (nativeDebugAPI) resume(h uintptr) error {
	previous, err := windows.ResumeThread(windows.Handle(h))
	if err != nil {
		return fmt.Errorf("ResumeThread: %w", err)
	}
	if previous == 0 {
		return errDebugSuspensionLost
	}
	return nil
}

func (nativeDebugAPI) close(h uintptr) error {
	if err := windows.CloseHandle(windows.Handle(h)); err != nil {
		return fmt.Errorf("CloseHandle: %w", err)
	}
	return nil
}
