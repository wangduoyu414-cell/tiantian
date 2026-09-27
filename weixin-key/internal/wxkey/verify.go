package wxkey

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"weixin-key/internal/config"
	"weixin-key/internal/wcdb"
)

// VerifyResult reports the outcome of verifying one piece of key material
// against one or more local databases. It deliberately carries counts and
// per-DB booleans, never the secret material itself.
type VerifyResult struct {
	// Material is "passphrase" or "enc_key": what kind of input was verified.
	Material              string          `json:"material"`
	PassphraseValidLength bool            `json:"passphrase_valid_length"`
	Method                string          `json:"method"`
	DBs                   []VerifyDBEntry `json:"dbs"`
	Matched               int             `json:"matched"`
	Total                 int             `json:"total"`
}

// VerifyDBEntry is one database's verification outcome.
type VerifyDBEntry struct {
	Path  string `json:"path"`
	Salt  string `json:"salt"`
	Match bool   `json:"match"`
	Error string `json:"error,omitempty"`
}

// VerifyPassphraseAgainstDBs derives per-DB enc_keys from passphraseHex and
// verifies each target DB offline via SQLCipher page-1 HMAC (no native WCDB
// required). If dbPath is empty, it verifies against every .db under the
// auto-detected account DB root.
func VerifyPassphraseAgainstDBs(passphraseHex, dbPath string) (*VerifyResult, error) {
	pass, err := hex.DecodeString(strings.TrimSpace(passphraseHex))
	if err != nil {
		return nil, fmt.Errorf("passphrase is not valid hex: %w", err)
	}
	res := &VerifyResult{
		Material:              "passphrase",
		PassphraseValidLength: len(pass) == wcdb.PassphraseLength,
		Method:                "sqlcipher4-page1-hmac",
	}
	if !res.PassphraseValidLength {
		return nil, fmt.Errorf("passphrase must be %d bytes (%d hex chars), got %d bytes",
			wcdb.PassphraseLength, wcdb.PassphraseLength*2, len(pass))
	}

	var dbs []string
	if dbPath != "" {
		dbs = []string{dbPath}
	} else {
		root, _, derr := config.AutoDetectDBRoot()
		if derr != nil {
			return nil, fmt.Errorf("auto-detect DB root: %w (or pass --db)", derr)
		}
		dbs = findDBFiles(root)
		if len(dbs) == 0 {
			return nil, fmt.Errorf("no .db files under %s", root)
		}
	}

	for _, d := range dbs {
		entry := VerifyDBEntry{Path: d}
		salt, serr := readDBSaltHex(d)
		if serr != nil {
			entry.Error = serr.Error()
			res.DBs = append(res.DBs, entry)
			continue
		}
		entry.Salt = salt

		encKey, derr := wcdb.DeriveEncKey(passphraseHex, salt)
		if derr != nil {
			entry.Error = derr.Error()
			res.DBs = append(res.DBs, entry)
			continue
		}
		ok, verr := wcdb.VerifyEncKeyPage1(d, encKey)
		if verr != nil {
			entry.Error = verr.Error()
			res.DBs = append(res.DBs, entry)
			continue
		}
		entry.Match = ok
		if entry.Match {
			res.Matched++
		}
		res.DBs = append(res.DBs, entry)
	}
	res.Total = len(res.DBs)
	return res, nil
}

// VerifyEncKeyAgainstDBs verifies a raw per-DB enc_key (64 hex chars) against
// one or more local databases via SQLCipher page-1 HMAC. No derivation is
// involved: the key either matches the DB with the same salt or it does not.
func VerifyEncKeyAgainstDBs(encKeyHex, dbPath string) (*VerifyResult, error) {
	key, err := normalizeEncKeyHex(encKeyHex)
	if err != nil {
		return nil, err
	}
	res := &VerifyResult{
		Material:              "enc_key",
		PassphraseValidLength: true, // not applicable; enc_key shape already enforced
		Method:                "sqlcipher4-page1-hmac",
	}

	var dbs []string
	if dbPath != "" {
		dbs = []string{dbPath}
	} else {
		root, _, derr := config.AutoDetectDBRoot()
		if derr != nil {
			return nil, fmt.Errorf("auto-detect DB root: %w (or pass --db)", derr)
		}
		dbs = findDBFiles(root)
		if len(dbs) == 0 {
			return nil, fmt.Errorf("no .db files under %s", root)
		}
	}

	for _, d := range dbs {
		entry := VerifyDBEntry{Path: d}
		salt, serr := readDBSaltHex(d)
		if serr != nil {
			entry.Error = serr.Error()
			res.DBs = append(res.DBs, entry)
			continue
		}
		entry.Salt = salt
		ok, verr := wcdb.VerifyEncKeyPage1(d, key)
		if verr != nil {
			entry.Error = verr.Error()
			res.DBs = append(res.DBs, entry)
			continue
		}
		entry.Match = ok
		if entry.Match {
			res.Matched++
		}
		res.DBs = append(res.DBs, entry)
	}
	res.Total = len(res.DBs)
	return res, nil
}

// ConfigKeyInfo summarizes what key material the loaded config holds, without
// revealing any of it.
func ConfigKeyInfo() (map[string]any, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	path, _ := config.Path()
	return map[string]any{
		"config_path":       path,
		"schema_version":    cfg.SchemaVersion,
		"wxid":              cfg.Wxid,
		"db_root":           cfg.DBRoot,
		"cached_key_count":  len(cfg.Keys),
		"key_entry_count":   len(cfg.KeyEntries),
		"has_passphrase":    cfg.HasPassphrase(),
		"kdf":               cfg.KDF,
		"passphrase_source": cfg.PassphraseSource,
	}, nil
}

// findDBFiles walks root for *.db files (WeChat stores them under db_storage).
func findDBFiles(root string) []string {
	var out []string
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".db") {
			out = append(out, p)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// readDBSaltHex reads the 16-byte SQLCipher salt from a DB header.
func readDBSaltHex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	buf := make([]byte, 16)
	if _, err := f.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
