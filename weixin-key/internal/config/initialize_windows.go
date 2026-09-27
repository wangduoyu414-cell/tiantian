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
	"strings"
	"unicode"
	"unicode/utf8"

	"weixin-key/internal/pathguard"
)

// InitializeMetadataAtPath bootstraps a fresh machine without running setup,
// reading environment material or overwriting any existing configuration.
// The parent must already exist. This is an exclusive metadata bootstrap, NOT
// an atomic key-cache update: an interrupted creation can leave an empty or
// incomplete file, with Applied=true and MetadataVerified=false. It is never
// retried by overwriting that file. No secret is accepted by this API.
func InitializeMetadataAtPath(ctx context.Context, path, accountRoot, account string) (MetadataInitProof, error) {
	return initializeMetadataAtPath(ctx, path, accountRoot, account, initializationHooks{})
}

type initializationHooks struct {
	create         func(*os.Root, string) (*os.File, error)
	beforeReadback func(*os.Root, string) error
}

func initializeMetadataAtPath(ctx context.Context, path, accountRoot, account string,
	hooks initializationHooks) (proof MetadataInitProof, err error) {
	if err := ctx.Err(); err != nil {
		return proof, err
	}
	err = coordinateConfigWrites(false, func() error {
		var inner error
		proof, inner = initializeMetadataWithWriterLease(ctx, path, accountRoot, account, hooks)
		return inner
	})
	return proof, errors.Join(err, ctx.Err())
}

func initializeMetadataWithWriterLease(ctx context.Context, path, accountRoot, account string,
	hooks initializationHooks) (proof MetadataInitProof, err error) {
	if err := ctx.Err(); err != nil {
		return proof, err
	}
	if !filepath.IsAbs(path) || !filepath.IsAbs(accountRoot) || account == "" || strings.HasPrefix(account, "-") ||
		len(account) > 128 || !utf8.ValidString(account) ||
		strings.ContainsAny(account, `/\:`) || strings.IndexFunc(account, func(r rune) bool {
		return unicode.IsControl(r) || unicode.IsSpace(r)
	}) >= 0 {
		return proof, errors.New("absolute config/db-root paths and an explicit account identifier are required")
	}
	volume := filepath.VolumeName(path)
	if strings.HasPrefix(volume, `\\`) || strings.Contains(strings.TrimPrefix(path, volume), ":") {
		return proof, errors.New("configuration must be a local file, not a network path or alternate stream")
	}
	source, err := openInitializationDirectory(accountRoot)
	if err != nil {
		return proof, err
	}
	defer source.close()
	accountRoot = source.path
	storage, err := source.root.Lstat("db_storage")
	if err != nil || !storage.IsDir() || storage.Mode()&os.ModeSymlink != 0 {
		return proof, errors.New("selected account must contain a real db_storage directory")
	}
	path = filepath.Clean(path)
	parent := filepath.Dir(path)
	target, err := openInitializationDirectory(parent)
	if err != nil {
		return proof, err
	}
	defer target.close()
	if _, err := pathguard.ResolveOutput(target.path, accountRoot); err != nil {
		return proof, err
	}
	raw, err := json.Marshal(&Config{SchemaVersion: CurrentSchemaVersion, Wxid: account, DBRoot: accountRoot})
	if err != nil {
		return proof, err
	}
	raw = append(raw, '\n')
	err = withConfigRootWriteLockUsing(target.root, filepath.Base(path), tryLockConfigFile, func(root *os.Root, base string) (retErr error) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := root.Lstat(base); err == nil {
			return errors.New("configuration already exists; initialization never overwrites it")
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		create := hooks.create
		if create == nil {
			create = func(root *os.Root, base string) (*os.File, error) {
				return root.OpenFile(base, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
			}
		}
		f, err := create(root, base)
		if err != nil {
			return err
		}
		proof.Applied = true // creation, not a complete/atomic publication
		defer func() {
			retErr = errors.Join(retErr, f.Close())
		}()
		// Apply actual Windows protection before writing even account metadata.
		if err := privateConfigFile(f); err != nil {
			return err
		}
		if err := verifyPrivateConfigFile(f); err != nil {
			return err
		}
		proof.PrivateFile = true
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := f.Write(raw); err != nil {
			return err
		}
		if err := f.Sync(); err != nil {
			return err
		}
		if hooks.beforeReadback != nil {
			if err := hooks.beforeReadback(root, base); err != nil {
				return err
			}
		}
		// Once written, complete readback even after cancellation. A later
		// failure must retain Applied so a caller never blindly retries.
		check, err := root.Open(base)
		if err != nil {
			return err
		}
		defer func() { retErr = errors.Join(retErr, check.Close()) }()
		original, originalErr := f.Stat()
		current, currentErr := check.Stat()
		link, linkErr := root.Lstat(base)
		if originalErr != nil || currentErr != nil || linkErr != nil ||
			link.Mode()&os.ModeSymlink != 0 || !os.SameFile(original, current) || !os.SameFile(current, link) {
			return errors.New("initialized configuration identity changed")
		}
		if err := verifyPrivateConfigFile(check); err != nil {
			return err
		}
		proof.PrivateFile = true
		readback, err := readConfigBytes(check)
		if err != nil {
			return err
		}
		defer clear(readback)
		if !bytes.Equal(raw, readback) {
			return errors.New("initialized configuration readback differs")
		}
		sum := sha256.Sum256(readback)
		proof.SHA256 = hex.EncodeToString(sum[:])
		proof.MetadataVerified = true
		return nil
	})
	return proof, errors.Join(err, ctx.Err())
}
