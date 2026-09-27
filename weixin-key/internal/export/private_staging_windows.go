//go:build windows

package export

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const exportStagingMargin int64 = 512 << 20

// createPrivateStaging creates the directory with a protected, current-user
// DACL already attached. Creating it permissively and tightening ACLs later
// would leave a window in which plaintext snapshots could be opened by
// another principal.
func createPrivateStaging(parent, runID string) (string, func() error, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", nil, err
	}
	dir := filepath.Join(parent, "staging-"+sanitizeFileComponent(runID)+"-"+hex.EncodeToString(nonce[:]))
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", nil, err
	}
	sd, err := windows.SecurityDescriptorFromString(
		"O:" + user.User.Sid.String() +
			"D:P(A;OICI;FA;;;" + user.User.Sid.String() + ")",
	)
	if err != nil {
		return "", nil, err
	}
	name, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return "", nil, err
	}
	attrs := windows.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		SecurityDescriptor: sd,
	}
	if err := windows.CreateDirectory(name, &attrs); err != nil {
		// A collision or an existing path is never removed.
		return "", nil, err
	}
	runtime.KeepAlive(sd)

	h, err := windows.CreateFile(
		name,
		windows.READ_CONTROL|windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		_ = os.Remove(dir)
		return "", nil, err
	}
	fail := func(err error) (string, func() error, error) {
		_ = windows.CloseHandle(h)
		_ = os.Remove(dir)
		return "", nil, err
	}

	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &info); err != nil {
		return fail(err)
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 ||
		info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return fail(errors.New("export staging identity is invalid"))
	}
	actual, err := windows.GetSecurityInfo(
		h,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.OWNER_SECURITY_INFORMATION,
	)
	if err != nil {
		return fail(err)
	}
	owner, _, err := actual.Owner()
	if err != nil || !owner.Equals(user.User.Sid) {
		return fail(errors.New("export staging owner is not the current user"))
	}
	control, _, err := actual.Control()
	if err != nil {
		return fail(err)
	}
	acl, _, err := actual.DACL()
	if err != nil || acl == nil || acl.AceCount != 1 ||
		control&windows.SE_DACL_PROTECTED == 0 {
		return fail(errors.New("export staging DACL is not protected/private"))
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(acl, 0, &ace); err != nil {
		return fail(err)
	}
	sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	private := ace.Header.AceType == windows.ACCESS_ALLOWED_ACE_TYPE &&
		ace.Header.AceFlags == 3 &&
		ace.Mask == windows.ACCESS_MASK(0x001F01FF) &&
		sid.Equals(user.User.Sid)
	runtime.KeepAlive(actual)
	if !private {
		return fail(errors.New("export staging is not current-user-only with child inheritance"))
	}

	cleanup := func() error {
		// Keep the exact run directory pinned until all child files have been
		// removed; never recursively remove the caller's output directory.
		closeErr := windows.CloseHandle(h)
		removeErr := os.RemoveAll(dir)
		if closeErr != nil && removeErr != nil {
			return errors.Join(closeErr, removeErr)
		}
		if closeErr != nil {
			return closeErr
		}
		return removeErr
	}
	return dir, cleanup, nil
}

func availableBytes(path string) (int64, error) {
	p, err := windows.UTF16PtrFromString(filepath.Clean(path))
	if err != nil {
		return 0, err
	}
	var free, total, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(p, &free, &total, &totalFree); err != nil {
		return 0, fmt.Errorf("free-space query for %s: %w", path, err)
	}
	if free > uint64(^uint64(0)>>1) {
		return int64(^uint64(0) >> 1), nil
	}
	return int64(free), nil
}
