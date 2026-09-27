//go:build windows

package config

import (
	"errors"
	"runtime"

	"golang.org/x/sys/windows"
)

// No environment/CLI override. Tests in this package isolate their kernel
// namespace from other concurrently running packages and real applications.
var configWriterMutexPrefix = `Global\wxcli-config-directories-v1-`

// All cooperating config writers serialize before opening directories/locks.
// Initialization's ancestor pins would otherwise interfere with another
// writer's rename even at a different config path on the same volume.
// This named current-user mutex contains no data and disappears with its last
// handle. It also covers the release of initialization's directory pins.
func coordinateConfigWrites(wait bool, fn func() error) (retErr error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	name, err := windows.UTF16PtrFromString(configWriterMutexPrefix + user.User.Sid.String())
	if err != nil {
		return err
	}
	handle, err := windows.CreateMutexEx(nil, name, 0, windows.SYNCHRONIZE|windows.MUTEX_MODIFY_STATE)
	// x/sys returns ERROR_ALREADY_EXISTS together with a valid opened handle.
	// That is the normal contention path, not a failed creation.
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return err
	}
	if handle == 0 || handle == windows.InvalidHandle {
		return errors.New("configuration coordination returned an invalid handle")
	}
	defer func() { retErr = errors.Join(retErr, windows.CloseHandle(handle)) }()
	// Windows mutex ownership is an OS-thread property, not a goroutine one.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	timeout := uint32(0)
	if wait {
		timeout = windows.INFINITE
	}
	state, err := windows.WaitForSingleObject(handle, timeout)
	if err != nil {
		return err
	}
	switch state {
	case uint32(windows.WAIT_TIMEOUT):
		return ErrConfigBusy
	case windows.WAIT_OBJECT_0, windows.WAIT_ABANDONED:
		// An abandoned mutex grants ownership; callers still validate actual
		// files and acquire their usual file lock. No previous state is trusted.
	default:
		return errors.New("unexpected configuration writer coordination state")
	}
	defer func() { retErr = errors.Join(retErr, windows.ReleaseMutex(handle)) }()
	return fn()
}
