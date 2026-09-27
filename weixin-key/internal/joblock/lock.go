// Package joblock provides a fail-fast, cross-process lock released by the OS
// after cancellation, normal exit, or a crash. Lock files are never deleted.
package joblock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrBusy = errors.New("another job owns this resource")

func Acquire(path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	name := filepath.Base(path)
	info, err := root.Lstat(name)
	if err != nil && !os.IsNotExist(err) {
		root.Close()
		return nil, err
	}
	if err == nil && !info.Mode().IsRegular() {
		root.Close()
		return nil, errors.New("lock is not a regular file")
	}
	f, err := root.OpenFile(name, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		root.Close()
		return nil, err
	}
	after, err := f.Stat()
	if err != nil || (info != nil && !os.SameFile(info, after)) {
		f.Close()
		root.Close()
		return nil, errors.New("lock identity changed")
	}
	if err := lock(f); err != nil {
		f.Close()
		root.Close()
		return nil, fmt.Errorf("%w: %s: %v", ErrBusy, path, err)
	}
	return func() { f.Close(); root.Close() }, nil
}
