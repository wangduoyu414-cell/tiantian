//go:build windows

package export

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func openFrozenSource(path string) (*os.File, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		if err == windows.ERROR_FILE_NOT_FOUND || err == windows.ERROR_PATH_NOT_FOUND {
			return nil, &os.PathError{Op: "open frozen source", Path: path, Err: err}
		}
		return nil, fmt.Errorf("%w: %s: %w; no source files were modified; a compatible live snapshot engine is still required", ErrLiveEncryptedSnapshot, path, err)
	}
	return os.NewFile(uintptr(h), path), nil
}
