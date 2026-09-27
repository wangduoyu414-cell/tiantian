package config

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CurrentSchemaVersion is the latest config schema written by this tool.
// Schema 5 protects secret fields using the current Windows user's DPAPI.
// Old schemas remain readable; old plaintext-only consumers cannot read v5.
const CurrentSchemaVersion = 5

// KeyEntry is one verified per-DB enc_key with the identity it is bound to:
// the account, the DB salt, the crypto profile (KDF), where the material came
// from, and when it last verified. Presence of a salt key alone is never
// treated as proof of validity - consumers re-verify against the live DB.
type KeyEntry struct {
	EncKey     string `json:"enc_key"`
	Kind       string `json:"kind,omitempty"` // always "enc_key" today
	Source     string `json:"source,omitempty"`
	WxID       string `json:"wxid,omitempty"`
	KDF        string `json:"kdf,omitempty"`
	VerifiedAt int64  `json:"verified_at,omitempty"`
}

// Config is wechat-cli's persistent key map. By default it intentionally stays
// at ~/.config/wxcli/config.json for wxkey / wx-cli compatibility;
// WECHAT_CLI_CONFIG or the legacy WX_MCP_CONFIG can point at an explicit file.
//
// Schema 2 carries a per-DB enc_key map: each WCDB file's SQLCipher salt maps
// to its 32-byte post-PBKDF2 encryption key.
//
// Schema 3 adds an account-scoped passphrase captured from WeChat 4.1+. The
// actual per-DB enc_keys are derived via PBKDF2-HMAC-SHA512. The derived keys
// are cached in Keys for fast runtime access; Passphrase is kept so re-keying
// (e.g. new DBs appearing after setup) does not require re-attaching to WeChat.
//
// Schema 4 adds KeyEntries (typed per-salt records with provenance).
// Schema 5 separates public metadata and protected_secrets on disk. Migration
// preserves original file bytes in <name>.schema4-backup.protected; it creates
// no new plaintext backup and does not delete any historical plaintext backup.
type Config struct {
	SchemaVersion int               `json:"schema_version,omitempty"`
	Wxid          string            `json:"wxid"`
	DBRoot        string            `json:"db_root"`
	Keys          map[string]string `json:"keys,omitempty"`
	// KeyEntries is the schema-4 typed view of Keys (same salt keys).
	KeyEntries  map[string]KeyEntry `json:"key_entries,omitempty"`
	ImageKey    string              `json:"image_key,omitempty"`
	ImageXORKey *int                `json:"image_xor_key,omitempty"`
	Key         string              `json:"key,omitempty"`
	KeyPID      int                 `json:"key_pid,omitempty"`
	KeyEpoch    int64               `json:"key_epoch,omitempty"`

	// Schema 3 fields
	Passphrase       string `json:"passphrase,omitempty"`        // hex-encoded 32-byte raw passphrase
	PassphraseSource string `json:"passphrase_source,omitempty"` // e.g. "env", "wechatwin_setkey_v1"
	PassphraseEpoch  int64  `json:"passphrase_epoch,omitempty"`  // unix seconds when passphrase was saved
	KDF              string `json:"kdf,omitempty"`               // empty = legacy raw-key mode; "pbkdf2-sha512-256000" = passphrase mode
}

// Ready reports whether the config has enough material to open WCDB files via
// wechat-cli's supported runtime path: schema-2 per-salt enc_keys or a schema-3
// passphrase from which those keys can be derived.
func (c *Config) Ready() bool {
	if c == nil {
		return false
	}
	if len(c.Keys) > 0 || len(c.KeyEntries) > 0 {
		return true
	}
	return c.HasPassphrase()
}

// HasPassphrase reports whether the config holds a usable schema-3 passphrase.
func (c *Config) HasPassphrase() bool {
	return c != nil && c.Passphrase != "" && c.KDF != ""
}

// normalize backfills KeyEntries from the legacy Keys map so readers always
// see the typed view, without touching the on-disk file by itself.
func (c *Config) normalize() {
	if len(c.Keys) == 0 {
		return
	}
	if c.KeyEntries == nil {
		c.KeyEntries = make(map[string]KeyEntry, len(c.Keys))
	}
	for salt, key := range c.Keys {
		if _, ok := c.KeyEntries[salt]; !ok {
			c.KeyEntries[salt] = KeyEntry{EncKey: key, Kind: "enc_key", Source: "keys-map"}
		}
	}
}

// SetVerifiedKey records one verified per-DB enc_key in both the legacy Keys
// map and the schema-4 KeyEntries view, bumping the schema version.
func (c *Config) SetVerifiedKey(saltHex, encKeyHex, source string, epoch int64) {
	if c.Keys == nil {
		c.Keys = map[string]string{}
	}
	if c.KeyEntries == nil {
		c.KeyEntries = map[string]KeyEntry{}
	}
	c.Keys[saltHex] = encKeyHex
	c.KeyEntries[saltHex] = KeyEntry{
		EncKey:     encKeyHex,
		Kind:       "enc_key",
		Source:     source,
		WxID:       c.Wxid,
		KDF:        c.KDF,
		VerifiedAt: epoch,
	}
	if c.SchemaVersion < CurrentSchemaVersion {
		c.SchemaVersion = CurrentSchemaVersion
	}
}

func dir() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(h, ".config", "wxcli")
	if err := validateHomeConfigComponents(filepath.Join(d, "config.json")); err != nil {
		return "", err
	}
	return d, nil
}

func Path() (string, error) {
	if p := firstEnv("WECHAT_CLI_CONFIG", "WX_MCP_CONFIG"); p != "" {
		return filepath.Clean(p), nil
	}
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	file, err := os.Open(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := &Config{}
			applyEnvOverrides(cfg)
			return cfg, nil
		}
		return nil, err
	}
	defer file.Close()
	b, err := readConfigBytes(file)
	if err != nil {
		return nil, err
	}
	defer clear(b)
	c, err := decodeConfig(b)
	if err != nil {
		return nil, err
	}
	applyEnvOverrides(c)
	return c, nil
}

func Save(c *Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	return withConfigWriteLock(p, func(root *os.Root, base string) error {
		return saveConfigToRoot(root, base, c)
	})
}

// Update performs a config read-modify-write transaction while holding the
// same cross-process lock used by wxkey. The callback must not call Save or
// Update recursively.
func Update(mutate func(*Config) error) error {
	if mutate == nil {
		return errors.New("config update callback is nil")
	}
	p, err := Path()
	if err != nil {
		return err
	}
	return withConfigWriteLock(p, func(root *os.Root, base string) error {
		cfg, err := loadConfigFromRoot(root, base)
		if err != nil {
			return err
		}
		applyEnvOverrides(cfg)
		if err := mutate(cfg); err != nil {
			return err
		}
		cfg.normalize()
		if len(cfg.KeyEntries) > 0 && cfg.SchemaVersion < CurrentSchemaVersion {
			cfg.SchemaVersion = CurrentSchemaVersion
		}
		return saveConfigToRoot(root, base, cfg)
	})
}

// backupLegacyConfig preserves the current on-disk config as
// <base>.schema4-backup.protected once under the config write lock. An existing
// readable backup is never replaced. Failure leaves the old config untouched.
func backupLegacyConfig(root *os.Root, base string) error {
	return backupConfig(root, base, false)
}

// Explicit cache updates also permit a readable metadata-only recovery backup
// for this same selected account. Legacy migration keeps its stricter rule.
func backupConfig(root *os.Root, base string, allowMetadata bool) error {
	src, err := root.Open(base)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer src.Close()
	raw, err := readConfigBytes(src)
	if err != nil {
		return err
	}
	defer clear(raw)
	name := base + ".schema4-backup.protected"
	if err := validateRootSaveTarget(root, name); err != nil {
		return err
	}
	if f, err := root.Open(name); err == nil {
		if err := verifyPrivateConfigFile(f); err != nil {
			f.Close()
			return err
		}
		existing, readErr := readConfigBytes(f)
		closeErr := f.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		var backup protectedValue
		if err := json.Unmarshal(existing, &backup); err != nil {
			return err
		}
		check, err := unprotectConfigBytes(&backup)
		defer clear(check)
		if err != nil {
			return err
		}
		restored, err := decodeConfig(check)
		if err != nil {
			return errors.New("migration backup is not a readable configuration")
		}
		metadataOK := false
		if allowMetadata && restored.DBRoot != "" && restored.Wxid != "" {
			current, err := decodeConfig(raw)
			metadataOK = err == nil && current.DBRoot == restored.DBRoot && current.Wxid == restored.Wxid
		}
		if !hasSecrets(restored) && !metadataOK {
			return errors.New("migration backup does not contain a readable secret configuration")
		}
		return nil // do not overwrite a prior usable recovery backup
	} else if !os.IsNotExist(err) {
		return err
	}
	sealed, err := protectConfigBytes(raw)
	if err != nil {
		return err
	}
	check, err := unprotectConfigBytes(sealed)
	if err != nil {
		return err
	}
	equal := bytes.Equal(check, raw)
	clear(check)
	if !equal {
		return errors.New("migration backup verification failed")
	}
	b, err := json.Marshal(sealed)
	if err != nil {
		return err
	}
	if len(b) > maxConfigBytes {
		return errors.New("protected migration backup exceeds size limit")
	}
	return writeConfigToRoot(root, name, b)
}

func withConfigWriteLock(path string, fn func(*os.Root, string) error) error {
	return coordinateConfigWrites(true, func() error {
		return withConfigWriteLockUncoordinated(path, lockConfigFile, fn)
	})
}

func withConfigWriteLockUsing(path string, acquire func(*os.File) (func() error, error), fn func(*os.Root, string) error) error {
	return coordinateConfigWrites(false, func() error {
		return withConfigWriteLockUncoordinated(path, acquire, fn)
	})
}

func withConfigWriteLockUncoordinated(path string, acquire func(*os.File) (func() error, error), fn func(*os.Root, string) error) error {
	root, base, err := openConfigWriteRoot(path)
	if err != nil {
		return err
	}
	defer root.Close()
	return withConfigRootWriteLockUsing(root, base, acquire, fn)
}

// withConfigRootWriteLockUsing shares the existing lock protocol with callers
// that have already opened and validated a write root without creating paths.
func withConfigRootWriteLockUsing(root *os.Root, base string, acquire func(*os.File) (func() error, error), fn func(*os.Root, string) error) error {
	lockName := "." + base + ".lock"
	if info, err := root.Lstat(lockName); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return errors.New("config lock path must be a regular file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	lockFile, err := root.OpenFile(lockName, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer lockFile.Close()
	lockInfo, err := lockFile.Stat()
	if err != nil || !lockInfo.Mode().IsRegular() {
		return errors.New("config lock is not a regular file")
	}
	pathInfo, err := root.Lstat(lockName)
	if err != nil || pathInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(pathInfo, lockInfo) {
		return errors.New("config lock path changed during validation")
	}
	if err := lockFile.Chmod(0o600); err != nil {
		return err
	}
	unlock, err := acquire(lockFile)
	if err != nil {
		return err
	}
	defer func() { _ = unlock() }()
	return fn(root, base)
}

func openConfigWriteRoot(p string) (*os.Root, string, error) {
	if err := validateHomeConfigComponents(p); err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return nil, "", err
	}
	if err := validateSavePath(p); err != nil {
		return nil, "", err
	}
	parent := filepath.Dir(p)
	root, err := os.OpenRoot(parent)
	if err != nil {
		return nil, "", err
	}
	openedDir, err := root.Open(".")
	if err != nil {
		_ = root.Close()
		return nil, "", err
	}
	openedInfo, statErr := openedDir.Stat()
	_ = openedDir.Close()
	if statErr != nil {
		_ = root.Close()
		return nil, "", statErr
	}
	parentInfo, err := os.Lstat(parent)
	if err != nil || parentInfo.Mode()&os.ModeSymlink != 0 || !parentInfo.IsDir() || !os.SameFile(parentInfo, openedInfo) {
		_ = root.Close()
		return nil, "", errors.New("config parent changed during validation")
	}
	if err := validateHomeConfigComponents(p); err != nil {
		_ = root.Close()
		return nil, "", err
	}
	return root, filepath.Base(p), nil
}

func saveConfigToRoot(root *os.Root, base string, c *Config) error {
	if old, err := loadConfigFromRoot(root, base); err == nil && old.SchemaVersion < CurrentSchemaVersion && hasSecrets(old) {
		if err := backupLegacyConfig(root, base); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	b, err := encodeConfig(c)
	if err != nil {
		return err
	}
	defer clear(b)
	return writeConfigToRoot(root, base, b)
}

func loadConfigFromRoot(root *os.Root, base string) (*Config, error) {
	file, err := root.Open(base)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("config path must be a regular file")
	}
	pathInfo, err := root.Lstat(base)
	if err != nil || pathInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(pathInfo, info) {
		return nil, errors.New("config path changed during validation")
	}
	b, err := readConfigBytes(file)
	if err != nil {
		return nil, err
	}
	defer clear(b)
	return decodeConfig(b)
}

func writeConfigToRoot(root *os.Root, base string, data []byte) error {
	if err := validateRootSaveTarget(root, base); err != nil {
		return err
	}
	var nonce [12]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	tmpName := fmt.Sprintf(".%s.tmp-%x", base, nonce[:])
	tmp, err := root.OpenFile(tmpName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	keepTemp := true
	defer func() {
		_ = tmp.Close()
		if keepTemp {
			_ = root.Remove(tmpName)
		}
	}()
	if err := privateConfigFile(tmp); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := validateRootSaveTarget(root, base); err != nil {
		return err
	}
	if err := root.Rename(tmpName, base); err != nil {
		return err
	}
	keepTemp = false
	return nil
}

func validateRootSaveTarget(root *os.Root, name string) error {
	info, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("config path must not be a symbolic link")
	}
	if !info.Mode().IsRegular() {
		return errors.New("config path must be a regular file")
	}
	return nil
}

func validateSavePath(path string) error {
	dirInfo, err := os.Lstat(filepath.Dir(path))
	if err != nil {
		return err
	}
	if !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("config parent must be a real directory, not a symbolic link")
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return validateHomeConfigComponents(path)
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("config path must not be a symbolic link")
	}
	if !info.Mode().IsRegular() {
		return errors.New("config path must be a regular file")
	}
	return validateHomeConfigComponents(path)
}

func validateHomeConfigComponents(path string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	home, err = filepath.Abs(filepath.Clean(home))
	if err != nil {
		return err
	}
	clean, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(home, clean)
	if err != nil || rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil
	}
	root, err := os.OpenRoot(home)
	if err != nil {
		return err
	}
	defer root.Close()
	current := ""
	parts := strings.Split(filepath.Clean(rel), string(os.PathSeparator))
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := root.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("config path components must not be symbolic links")
		}
		if i < len(parts)-1 && !info.IsDir() {
			return errors.New("config parent component must be a directory")
		}
	}
	return nil
}

func applyEnvOverrides(c *Config) {
	if c == nil {
		return
	}
	if root := firstEnv("WECHAT_CLI_DB_ROOT", "WX_MCP_DB_ROOT"); root != "" {
		c.DBRoot = filepath.Clean(root)
		if c.Wxid == "" {
			c.Wxid = wxidFromAccountDir(c.DBRoot)
		}
	}
	if key := firstEnv("WECHAT_CLI_IMAGE_KEY", "WX_MCP_IMAGE_KEY"); key != "" {
		c.ImageKey = key
	}
}

func DefaultWeChatBase() (string, error) {
	bases, err := DefaultWeChatBases()
	if err != nil {
		return "", err
	}
	if len(bases) == 0 {
		return "", fmt.Errorf("no default WeChat data roots for this platform")
	}
	return bases[0], nil
}

func AutoDetectDBRoot() (string, string, error) {
	if root := firstEnv("WECHAT_CLI_DB_ROOT", "WX_MCP_DB_ROOT"); root != "" {
		root = filepath.Clean(root)
		if _, err := os.Stat(filepath.Join(root, "db_storage")); err != nil {
			return "", "", fmt.Errorf("WECHAT_CLI_DB_ROOT=%s does not contain db_storage: %w", root, err)
		}
		return root, wxidFromAccountDir(root), nil
	}

	type cand struct{ full, wxid string }
	var cands []cand
	var checked []string
	bases, err := DefaultWeChatBases()
	if err != nil {
		return "", "", err
	}
	for _, base := range bases {
		checked = append(checked, base)
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			switch name {
			case "all_users", "applet", "backup", "wmpf":
				continue
			}
			full := filepath.Join(base, name)
			if _, err := os.Stat(filepath.Join(full, "db_storage")); err == nil {
				cands = append(cands, cand{full, wxidFromAccountDir(full)})
			}
		}
	}
	switch len(cands) {
	case 0:
		return "", "", fmt.Errorf("no account directory with db_storage found under checked WeChat roots:\n%s", strings.Join(checked, "\n"))
	case 1:
		return cands[0].full, cands[0].wxid, nil
	}

	var lines []string
	for _, c := range cands {
		lines = append(lines, fmt.Sprintf("  %s  (wxid=%s)", c.full, c.wxid))
	}
	return "", "", fmt.Errorf("multiple WeChat accounts found; refusing to autodetect.\nCandidates:\n%s\nSet WECHAT_CLI_DB_ROOT to the intended account directory", strings.Join(lines, "\n"))
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	return ""
}

func wxidFromAccountDir(path string) string {
	name := filepath.Base(filepath.Clean(path))
	if idx := lastIndex(name, "_"); idx > 0 {
		return name[:idx]
	}
	return name
}

func withXWeChatFilesBase(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	return []string{path, filepath.Join(path, "xwechat_files")}
}

func lastIndex(s, sep string) int {
	for i := len(s) - len(sep); i >= 0; i-- {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}
