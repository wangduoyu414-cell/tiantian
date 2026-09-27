//go:build windows

package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// Initialization accepts ordinary local DOS paths only. Pin every ancestor
// without write/delete sharing before any file is created. This deliberately
// rejects links instead of resolving them and later reopening a moving alias.
// The pins last only for the short metadata transaction, not key acquisition.
type initializationDirectory struct {
	root    *os.Root
	path    string
	handles []windows.Handle
}

func (d *initializationDirectory) close() {
	if d.root != nil {
		_ = d.root.Close()
	}
	for i := len(d.handles) - 1; i >= 0; i-- {
		_ = windows.CloseHandle(d.handles[i])
	}
}

func localInitializationVolume(path string, driveType func(*uint16) uint32) bool {
	volume := filepath.VolumeName(path)
	if len(volume) != 2 || volume[1] != ':' ||
		strings.Contains(strings.TrimPrefix(path, volume), ":") {
		return false
	}
	p, err := windows.UTF16PtrFromString(volume + `\`)
	if err != nil {
		return false
	}
	kind := driveType(p)
	return kind == 2 || kind == 3 // DRIVE_REMOVABLE or DRIVE_FIXED; never REMOTE
}

func openInitializationDirectory(path string) (_ *initializationDirectory, err error) {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) || !localInitializationVolume(path, windows.GetDriveType) {
		return nil, errors.New("initialization requires an existing local directory, not a network/device path")
	}
	d := &initializationDirectory{}
	defer func() {
		if err != nil {
			d.close()
		}
	}()
	volume := filepath.VolumeName(path)
	current := volume + `\`
	paths := []string{current}
	suffix := strings.TrimPrefix(path, current)
	if suffix != "" {
		for _, part := range strings.Split(suffix, `\`) {
			current = filepath.Join(current, part)
			paths = append(paths, current)
		}
	}
	var last windows.ByHandleFileInformation
	for _, item := range paths {
		name, err := windows.UTF16PtrFromString(item)
		if err != nil {
			return nil, err
		}
		// FILE_READ_ATTRIBUTES alone is metadata-only and does not establish
		// the sharing exclusion. LIST_DIRECTORY participates as data access.
		h, err := windows.CreateFile(name, windows.FILE_LIST_DIRECTORY|windows.FILE_READ_ATTRIBUTES,
			windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING,
			windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
		if err != nil {
			return nil, err
		}
		d.handles = append(d.handles, h)
		if err := windows.GetFileInformationByHandle(h, &last); err != nil {
			return nil, err
		}
		if last.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 ||
			last.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 {
			return nil, errors.New("initialization directory ancestors must be real directories, not reparse points")
		}
	}
	var buffer [32768]uint16
	n, err := windows.GetFinalPathNameByHandle(d.handles[len(d.handles)-1], &buffer[0], uint32(len(buffer)), 0)
	if err != nil || n == 0 || n >= uint32(len(buffer)) {
		return nil, errors.New("cannot resolve pinned initialization directory")
	}
	resolved := strings.TrimPrefix(windows.UTF16ToString(buffer[:n]), `\\?\`)
	// A mapped network drive or a DOS-device alias must not defeat the input
	// check. Recheck the actual handle's final DOS-volume path before writes.
	if !localInitializationVolume(resolved, windows.GetDriveType) {
		return nil, errors.New("pinned initialization directory is not on a local volume")
	}
	d.path = resolved
	d.root, err = os.OpenRoot(resolved) // never MkdirAll
	if err != nil {
		return nil, err
	}
	opened, err := d.root.Open(".")
	if err != nil {
		return nil, err
	}
	var actual windows.ByHandleFileInformation
	infoErr := windows.GetFileInformationByHandle(windows.Handle(opened.Fd()), &actual)
	closeErr := opened.Close()
	if err := errors.Join(infoErr, closeErr); err != nil {
		return nil, err
	}
	if actual.VolumeSerialNumber != last.VolumeSerialNumber ||
		actual.FileIndexHigh != last.FileIndexHigh || actual.FileIndexLow != last.FileIndexLow {
		return nil, errors.New("initialization write root differs from pinned directory")
	}
	return d, nil
}
