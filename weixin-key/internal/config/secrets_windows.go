//go:build windows

package config

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const protectionScheme = "dpapi-user-v1"

func configCrypt(data []byte, encrypt bool) ([]byte, error) {
	if len(data) == 0 || len(data) > maxConfigBytes {
		return nil, errors.New("protected payload size invalid")
	}
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	entropy := []byte("weixin-key/config/dpapi-user-v1")
	e := windows.DataBlob{Size: uint32(len(entropy)), Data: &entropy[0]}
	var out windows.DataBlob
	var err error
	if encrypt {
		err = windows.CryptProtectData(&in, nil, &e, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	} else {
		err = windows.CryptUnprotectData(&in, nil, &e, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	}
	runtime.KeepAlive(data)
	runtime.KeepAlive(entropy)
	if err != nil {
		return nil, fmt.Errorf("Windows current-user data protection: %w", err)
	}
	if out.Data == nil {
		return nil, errors.New("Windows returned empty protected data")
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	if out.Size > maxConfigBytes {
		return nil, errors.New("Windows returned oversized protected data")
	}
	native := unsafe.Slice(out.Data, int(out.Size))
	defer clear(native)
	return append([]byte(nil), native...), nil
}
func protectConfigBytes(b []byte) (*protectedValue, error) {
	encrypted, err := configCrypt(b, true)
	return &protectedValue{Scheme: protectionScheme, Ciphertext: encrypted}, err
}
func unprotectConfigBytes(v *protectedValue) ([]byte, error) {
	if v == nil || v.Scheme != protectionScheme {
		return nil, errors.New("unsupported protected configuration scheme")
	}
	return configCrypt(v.Ciphertext, false)
}

// Reopen by handle, not by path, so the private DACL is applied to the exact
// temporary file before writing. This changes no parent/system-wide ACL.
func privateConfigFile(f *os.File) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	reopen := windows.NewLazySystemDLL("kernel32.dll").NewProc("ReOpenFile")
	h, _, callErr := reopen.Call(f.Fd(), windows.WRITE_DAC|windows.READ_CONTROL, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, 0)
	if h == 0 || h == uintptr(windows.InvalidHandle) {
		return fmt.Errorf("reopen private config handle: %w", callErr)
	}
	defer windows.CloseHandle(windows.Handle(h))
	return windows.SetSecurityInfo(windows.Handle(h), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil)
}

// Check the opened file, not a requested chmod mode or path-based ACL. An
// existing recovery backup is evidence only when its actual permissions still
// match the current-user-only policy used by our writer. Do not silently widen
// or repair another file's ACL in a read/validation operation.
func verifyPrivateConfigFile(f *os.File) error {
	sd, err := windows.GetSecurityInfo(windows.Handle(f.Fd()), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return fmt.Errorf("read configuration file DACL: %w", err)
	}
	control, _, err := sd.Control()
	if err != nil {
		return err
	}
	acl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	if control&windows.SE_DACL_PROTECTED == 0 || acl == nil || acl.AceCount != 1 {
		return errors.New("protected config backup does not have a private DACL")
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(acl, 0, &ace); err != nil {
		return err
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	private := ace.Header.AceType == windows.ACCESS_ALLOWED_ACE_TYPE &&
		ace.Header.AceFlags == 0 && ace.Mask == windows.ACCESS_MASK(0x001F01FF) && // FILE_ALL_ACCESS
		sid.Equals(user.User.Sid)
	runtime.KeepAlive(sd)
	if !private {
		return errors.New("protected config backup is not restricted to the current user")
	}
	return nil
}
