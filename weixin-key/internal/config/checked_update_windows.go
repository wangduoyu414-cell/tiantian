//go:build windows

package config

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

var ErrConfigBusy = errors.New("configuration is locked by another writer")
var ErrConfigChanged = errors.New("configuration differs from the explicitly observed version")

// ProtectedUpdateProof contains no material. Applied stays true if a later
// readback fails, so callers must reconcile instead of blindly retrying.
type ProtectedUpdateProof struct {
	Applied           bool   `json:"applied"`
	BackupVerified    bool   `json:"backup_verified"`
	ProtectedReadback bool   `json:"protected_readback"`
	SHA256            string `json:"sha256,omitempty"`
}

func tryLockConfigFile(file *os.File) (func() error, error) {
	var ov windows.Overlapped
	err := windows.LockFileEx(windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &ov)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return nil, ErrConfigBusy
	}
	if err != nil {
		return nil, err
	}
	return func() error { return windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &ov) }, nil
}

// UpdateProtectedAtPath never consults env/default paths and never waits on a
// held config lock. The existing config must match an explicit SHA256 and have
// an account identity; this operation may not switch that identity.
// The OS lock coordinates cooperating writers. The second digest check also
// catches edits made during the callback, not arbitrary hostile atomic races.
func UpdateProtectedAtPath(ctx context.Context, path, expectedSHA string, mutate func(*Config) error) (proof ProtectedUpdateProof, err error) {
	return updateProtectedAtPath(ctx, path, expectedSHA, mutate, verifyProtectedReadback)
}

func updateProtectedAtPath(ctx context.Context, path, expectedSHA string, mutate func(*Config) error,
	readback func(*os.Root, string, *Config, *ProtectedUpdateProof) error) (proof ProtectedUpdateProof, err error) {
	if err := ctx.Err(); err != nil {
		return proof, err
	}
	want, parseErr := hex.DecodeString(expectedSHA)
	if parseErr != nil || len(want) != 32 || !filepath.IsAbs(path) || mutate == nil {
		return proof, errors.New("explicit configuration path, SHA256 and mutation are required")
	}
	err = withConfigWriteLockUsing(path, tryLockConfigFile, func(root *os.Root, base string) error {
		read := func() ([]byte, error) {
			f, err := root.Open(base)
			if err != nil {
				return nil, err
			}
			defer f.Close()
			info, err := f.Stat()
			if err != nil || !info.Mode().IsRegular() {
				return nil, errors.New("configuration is not a regular file")
			}
			return readConfigBytes(f)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		before, err := read()
		if err != nil {
			return err
		}
		defer clear(before)
		hash := sha256.Sum256(before)
		if !bytes.Equal(hash[:], want) {
			return ErrConfigChanged
		}
		cfg, err := decodeConfig(before)
		if err != nil {
			return err
		}
		rootID, accountID := cfg.DBRoot, cfg.Wxid
		if rootID == "" || accountID == "" {
			return errors.New("existing configuration has no selected account")
		}
		if err := mutate(cfg); err != nil {
			return err
		}
		if cfg.DBRoot != rootID || cfg.Wxid != accountID || !hasSecrets(cfg) {
			return errors.New("protected update cannot change account identity or omit material")
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		current, err := read()
		if err != nil {
			return err
		}
		currentHash := sha256.Sum256(current)
		clear(current)
		if currentHash != hash {
			return ErrConfigChanged
		}
		// Ensure the current-user-protected, one-time migration backup
		// exists (including metadata-only configs). A prior backup is kept.
		if err := backupConfig(root, base, true); err != nil {
			return err
		}
		proof.BackupVerified = true
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := saveConfigToRoot(root, base, cfg); err != nil {
			return err
		}
		proof.Applied = true
		// Complete readback even if cancellation arrived during the atomic
		// commit. Applied/readback facts survive; cancellation still returns.
		return readback(root, base, cfg, &proof)
	})
	return proof, errors.Join(err, ctx.Err())
}

func verifyProtectedReadback(root *os.Root, base string, cfg *Config, proof *ProtectedUpdateProof) error {
	f, err := root.Open(base)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	pathInfo, pathErr := root.Lstat(base)
	if err != nil || pathErr != nil || !info.Mode().IsRegular() ||
		pathInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(info, pathInfo) {
		return errors.New("protected readback file identity is invalid")
	}
	after, err := readConfigBytes(f)
	if err != nil {
		return err
	}
	defer clear(after)
	afterHash := sha256.Sum256(after)
	proof.SHA256 = hex.EncodeToString(afterHash[:])
	var disk diskConfig
	if err := json.Unmarshal(after, &disk); err != nil || disk.Protected == nil || hasSecrets(&disk.Config) {
		return errors.New("protected configuration readback has visible material or no protection")
	}
	check, err := decodeConfig(after)
	if err != nil {
		return err
	}
	cfg.SchemaVersion = CurrentSchemaVersion
	cfg.normalize()
	wantJSON, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	defer clear(wantJSON)
	gotJSON, err := json.Marshal(check)
	if err != nil {
		return err
	}
	defer clear(gotJSON)
	if !bytes.Equal(wantJSON, gotJSON) {
		return errors.New("protected configuration readback differs from requested values")
	}
	if err := verifyPrivateConfigFile(f); err != nil {
		return err
	}
	proof.ProtectedReadback = true
	return nil
}

// ReadProtectedAtPath is a read-only, explicit digest-bound load. It does not
// apply environment overrides or create directories/locks/migration backups.
func ReadProtectedAtPath(path, expectedSHA string) (*Config, error) {
	want, err := hex.DecodeString(expectedSHA)
	if err != nil || len(want) != 32 || !filepath.IsAbs(path) {
		return nil, errors.New("explicit protected configuration path and SHA256 are required")
	}
	if err := validateSavePath(path); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer root.Close()
	f, err := root.Open(filepath.Base(path))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := verifyPrivateConfigFile(f); err != nil {
		return nil, err
	}
	raw, err := readConfigBytes(f)
	if err != nil {
		return nil, err
	}
	defer clear(raw)
	hash := sha256.Sum256(raw)
	if !bytes.Equal(hash[:], want) {
		return nil, ErrConfigChanged
	}
	var disk diskConfig
	if err := json.Unmarshal(raw, &disk); err != nil || disk.Protected == nil {
		return nil, errors.New("explicit cache must be protected")
	}
	return decodeConfig(raw)
}
