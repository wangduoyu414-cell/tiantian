//go:build windows && amd64

// Package sqliteengine supplies the pinned, local-only SQLite3 Multiple
// Ciphers engine for read-transaction snapshots. It does not capture keys.
package sqliteengine

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/windows"
	"weixin-key/internal/safefile"
	"weixin-key/internal/wcdb"
)

//go:embed bin/sqlite3mc_x64.dll
var library []byte

//go:embed LICENSE.sqlite3mc
var license []byte

const Version = "sqlite3mc-2.5.1/sqlite-3.53.4"
const librarySHA256 = "5030decc6d914539e3b9b7e28aa4f6de1e7161dac6fd4b21eb02e6754d9b175e"

var ErrUnavailable = errors.New("validated encrypted snapshot engine unavailable")
var once sync.Once
var initErr error
var api struct {
	open         func(string, *uintptr, int32, uintptr) int32
	close        func(uintptr) int32
	errmsg       func(uintptr) string
	interrupt    func(uintptr)
	key          func(uintptr, string, unsafe.Pointer, int32) int32
	exec         func(uintptr, string, uintptr, uintptr, uintptr) int32
	config       func(uintptr, string, int32) int32
	cipherConfig func(uintptr, string, string, int32) int32
	backupInit   func(uintptr, string, uintptr, string) uintptr
	backupStep   func(uintptr, int32) int32
	backupFinish func(uintptr) int32
}

func initialize() error {
	once.Do(func() { initErr = load() })
	return initErr
}
func load() error {
	if fmt.Sprintf("%x", sha256.Sum256(library)) != librarySHA256 {
		return errors.New("embedded cipher library checksum mismatch")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(cache, "weixin-key", "sqlite3mc-2.5.1", librarySHA256)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dir, "sqlite3mc_x64.dll")
	b, err := os.ReadFile(path)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(b)) != librarySHA256 {
		f, err := os.CreateTemp(dir, ".engine-*")
		if err != nil {
			return err
		}
		tmp := f.Name()
		defer os.Remove(tmp)
		_, err = f.Write(library)
		if err == nil {
			err = f.Sync()
		}
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if err := safefile.Replace(tmp, path); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "LICENSE.sqlite3mc"), license, 0o600); err != nil {
		return err
	}
	h, err := windows.LoadLibraryEx(path, 0, windows.LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|windows.LOAD_LIBRARY_SEARCH_SYSTEM32)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	for _, binding := range []struct {
		fn   any
		name string
	}{
		{&api.open, "sqlite3_open_v2"}, {&api.close, "sqlite3_close_v2"}, {&api.key, "sqlite3_key_v2"}, {&api.errmsg, "sqlite3_errmsg"}, {&api.interrupt, "sqlite3_interrupt"},
		{&api.exec, "sqlite3_exec"}, {&api.config, "sqlite3mc_config"}, {&api.cipherConfig, "sqlite3mc_config_cipher"},
		{&api.backupInit, "sqlite3_backup_init"}, {&api.backupStep, "sqlite3_backup_step"}, {&api.backupFinish, "sqlite3_backup_finish"},
	} {
		ptr, err := windows.GetProcAddress(h, binding.name)
		if err != nil {
			return fmt.Errorf("cipher engine export %s: %w", binding.name, err)
		}
		purego.RegisterFunc(binding.fn, ptr)
	}
	return nil
}

func openEncrypted(path, keyHex, saltHex string, flags int32) (uintptr, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return 0, errors.New("invalid raw key length")
	}
	salt, err := hex.DecodeString(saltHex)
	if err != nil || len(salt) != 16 {
		return 0, errors.New("invalid salt length")
	}
	defer clear(key)
	var h uintptr
	if rc := api.open(path, &h, flags|0x10000, 0); rc != 0 {
		if h != 0 {
			api.close(h)
		}
		return 0, fmt.Errorf("cipher source open rc=%d", rc)
	}
	fail := func(err error) (uintptr, error) { api.close(h); return 0, err }
	if api.config(h, "cipher", 4) < 0 {
		return fail(errors.New("SQLCipher adapter unavailable"))
	}
	for _, p := range []struct {
		name  string
		value int32
	}{
		{"legacy", 4}, {"legacy_page_size", 4096}, {"kdf_iter", 256000}, {"fast_kdf_iter", 2},
		{"hmac_use", 1}, {"hmac_pgno", 1}, {"hmac_salt_mask", 0x3a}, {"kdf_algorithm", 2}, {"hmac_algorithm", 2},
	} {
		if api.cipherConfig(h, "sqlcipher", p.name, p.value) < 0 {
			return fail(fmt.Errorf("cipher parameter unsupported: %s", p.name))
		}
	}
	// SQLCipher's WAL encrypts pages before calculating WAL checksums.
	// mc_legacy_wal=1 is the old SQLite3MC VFS format (checksum before
	// encryption), NOT SQLCipher's format. Pin the standard encrypted-page
	// checksum format, independently checked against the Go WAL reader.
	if api.config(h, "mc_legacy_wal", 0) < 0 {
		return fail(errors.New("SQLCipher WAL format unavailable"))
	}
	raw := make([]byte, 4+len(key)+len(salt))
	copy(raw, "raw:")
	copy(raw[4:], key)
	copy(raw[4+len(key):], salt)
	defer clear(raw)
	rc := api.key(h, "main", unsafe.Pointer(&raw[0]), int32(len(raw)))
	runtime.KeepAlive(raw)
	if rc != 0 {
		return fail(fmt.Errorf("cipher material setup rc=%d", rc))
	}
	if rc := api.exec(h, "PRAGMA busy_timeout=100", 0, 0, 0); rc != 0 {
		return fail(fmt.Errorf("cipher busy timeout rc=%d", rc))
	}
	return h, nil
}

// Backup pins a read transaction on a read-only SOURCE. SQLite3MC requires
// compatible codecs at both backup endpoints, so it first backs up to a
// private, identically encrypted destination and then verifies/decrypts every
// page in Go. It never falls back to a logical copy that can skip tables.
func Backup(ctx context.Context, source, destination, keyHex, saltHex string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := initialize(); err != nil {
		return err
	}
	before, err := os.Stat(source)
	if err != nil {
		return err
	}
	if dst, err := os.Stat(destination); err == nil && os.SameFile(before, dst) {
		return errors.New("snapshot destination aliases source")
	}
	tmp, err := os.CreateTemp(filepath.Dir(destination), ".cipher-snapshot-*")
	if err != nil {
		return err
	}
	encrypted := tmp.Name()
	if err := tmp.Close(); err != nil {
		os.Remove(encrypted)
		return err
	}
	defer os.Remove(encrypted)
	defer os.Remove(encrypted + "-journal")
	if err := backupEncrypted(ctx, source, encrypted, keyHex, saltHex, nil); err != nil {
		return err
	}
	after, err := os.Stat(source)
	if err != nil || !os.SameFile(before, after) {
		return errors.New("encrypted source identity changed during backup")
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return err
	}
	defer clear(key)
	_, err = wcdb.DecryptDBContext(ctx, encrypted, destination, key)
	return err
}

func backupEncrypted(ctx context.Context, source, destination, keyHex, saltHex string, afterPin func() error) error {
	src, err := openEncrypted(source, keyHex, saltHex, 1)
	if err != nil {
		return err
	}
	defer api.close(src)
	done, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		select {
		case <-ctx.Done():
			api.interrupt(src)
		case <-done:
		}
	}()
	defer func() { close(done); <-stopped }()
	if rc := api.exec(src, "BEGIN; SELECT count(*) FROM sqlite_master", 0, 0, 0); rc != 0 {
		return fmt.Errorf("cipher read transaction rc=%d", rc)
	}
	defer api.exec(src, "ROLLBACK", 0, 0, 0)
	if afterPin != nil {
		if err := afterPin(); err != nil {
			return err
		}
	}
	dst, err := openEncrypted(destination, keyHex, saltHex, 2|4)
	if err != nil {
		return err
	}
	defer api.close(dst)
	b := api.backupInit(dst, "main", src, "main")
	if b == 0 {
		return fmt.Errorf("cipher snapshot backup initialization failed; no fallback performed: %s", api.errmsg(dst))
	}
	finished := false
	defer func() {
		if !finished {
			api.backupFinish(b)
		}
	}()
	deadline := time.Now().Add(2 * time.Minute)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return errors.New("cipher snapshot backup timed out")
		}
		rc := api.backupStep(b, 128)
		switch rc {
		case 101:
			finished = true
			if rc := api.backupFinish(b); rc != 0 {
				return fmt.Errorf("cipher backup finish rc=%d", rc)
			}
			return nil
		case 0:
		case 5, 6:
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(20 * time.Millisecond):
			}
		default:
			return fmt.Errorf("cipher snapshot page copy rc=%d", rc)
		}
	}
}
