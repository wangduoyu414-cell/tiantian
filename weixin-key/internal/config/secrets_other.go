//go:build !windows

package config

import (
	"errors"
	"os"
)

func protectConfigBytes([]byte) (*protectedValue, error) {
	return nil, errors.New("protected secret persistence is not implemented on this platform; legacy files remain readable")
}
func unprotectConfigBytes(*protectedValue) ([]byte, error) {
	return nil, errors.New("this protected configuration requires its Windows user; no plaintext fallback")
}
func privateConfigFile(f *os.File) error { return f.Chmod(0o600) }

func verifyPrivateConfigFile(f *os.File) error {
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Mode().Perm()&0o077 != 0 {
		return errors.New("protected config backup has non-private permissions")
	}
	return nil
}
