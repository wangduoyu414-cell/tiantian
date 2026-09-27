package wxkey

import (
	"encoding/hex"
	"fmt"
	"path/filepath"

	"weixin-key/internal/wcdb"
)

// DecryptOptions selects key material for DecryptDatabase. Explicit fields
// win over the config cache; resolution itself lives in KeyResolver.
type DecryptOptions struct {
	// EncKeyHex is a 64-hex per-DB enc_key (raw SQLCipher key, salt appended
	// from the file header at open time).
	EncKeyHex string
	// PassphraseHex is the 64-hex account-level passphrase; per-DB enc_key is
	// derived via PBKDF2-HMAC-SHA512 against the file's own salt.
	PassphraseHex string
	// NoConfig disables the config fallback (cached per-salt enc_keys and the
	// stored schema-3 passphrase). By default the config IS consulted, so a
	// previously verified setup lets decrypt run fully offline.
	NoConfig bool
}

// DecryptResult summarizes a completed decrypt. It intentionally carries no
// key material - only paths, page count, and which source satisfied the key.
type DecryptResult struct {
	DBPath    string `json:"db_path"`
	OutPath   string `json:"out_path"`
	Pages     int    `json:"pages"`
	KeySource string `json:"key_source"`
	Verified  bool   `json:"verified"`
}

// DecryptDatabase decrypts one WeChat 4.1+ SQLCipher .db into a plaintext
// SQLite file at outPath. Key material resolution order (via KeyResolver):
//
//  1. --enc-key / opts.EncKeyHex
//  2. --passphrase / opts.PassphraseHex
//  3. WECHAT_CLI_PASSPHRASE_HEX
//  4. config cached per-salt enc_key (re-verified against the live file)
//  5. config schema-3 passphrase
//
// The chosen material is verified against page-1 HMAC before any output is
// written, so a wrong key fails fast with no partial plaintext.
func DecryptDatabase(dbPath, outPath string, opts DecryptOptions) (*DecryptResult, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("decrypt: --db is required")
	}
	if outPath == "" {
		return nil, fmt.Errorf("decrypt: --out is required")
	}
	if filepath.Clean(dbPath) == filepath.Clean(outPath) {
		return nil, fmt.Errorf("decrypt: --out must differ from --db (refusing to overwrite the encrypted source)")
	}

	encKeyHex, source, err := resolveDecryptKey(dbPath, opts)
	if err != nil {
		return nil, err
	}

	// Fail fast: verify the derived/claimed key before touching outPath.
	ok, err := wcdb.VerifyEncKeyPage1(dbPath, encKeyHex)
	if err != nil {
		return nil, fmt.Errorf("verify key against %s: %w", dbPath, err)
	}
	if !ok {
		return nil, fmt.Errorf("key from %s does not match %s (page-1 HMAC)", source, dbPath)
	}

	encKey, err := hex.DecodeString(encKeyHex)
	if err != nil {
		return nil, err
	}
	pages, err := wcdb.DecryptDB(dbPath, outPath, encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt %s: %w", dbPath, err)
	}
	return &DecryptResult{
		DBPath:    dbPath,
		OutPath:   outPath,
		Pages:     pages,
		KeySource: source,
		Verified:  true,
	}, nil
}

// resolveDecryptKey delegates to the shared KeyResolver so CLI, and later the
// GUI, resolve key material identically (cache-first, offline-only).
func resolveDecryptKey(dbPath string, opts DecryptOptions) (encKeyHex, source string, err error) {
	r := NewKeyResolver()
	res, err := r.ResolveForDB(dbPath, ResolveOptions{
		EncKeyHex:     opts.EncKeyHex,
		PassphraseHex: opts.PassphraseHex,
		NoConfig:      opts.NoConfig,
	})
	if err != nil {
		return "", "", err
	}
	return res.EncKeyHex, res.Source, nil
}
