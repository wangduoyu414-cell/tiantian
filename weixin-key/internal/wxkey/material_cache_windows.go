//go:build windows

package wxkey

import (
	"context"
	"encoding/hex"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"weixin-key/internal/config"
	"weixin-key/internal/pathguard"
	"weixin-key/internal/wcdb"
)

type MaterialCacheReport struct {
	VerifiedDBs  int                         `json:"verified_dbs"`
	UniqueKeys   int                         `json:"unique_keys"`
	AccountBound bool                        `json:"account_bound"`
	Update       config.ProtectedUpdateProof `json:"update"`
}

// CacheVerifiedPassiveKeys is explicit persistence, separate from observation.
// Only an active, fully validated consumer can call it. Every current DB is
// re-authenticated through its retained source identity before config mutation.
// No passphrase is persisted; stored entries are typed per-salt raw enc_keys.
func CacheVerifiedPassiveKeys(ctx context.Context, keys *VerifiedPassiveKeys, configPath, expectedConfigSHA string) (report MaterialCacheReport, retErr error) {
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if keys == nil || !keys.ready || keys.source == nil || len(keys.keys) == 0 || len(keys.keys) > 64 || keys.provenance == "" {
		return report, errors.New("an active fully verified material consumer is required")
	}
	accountRoot := filepath.Dir(keys.base)
	if !filepath.IsAbs(configPath) {
		return report, errors.New("cache configuration path must be explicit and absolute")
	}
	if _, err := pathguard.ResolveOutput(filepath.Dir(configPath), accountRoot); err != nil {
		return report, err
	}
	names := make([]string, 0, len(keys.keys))
	for name := range keys.keys {
		names = append(names, name)
	}
	sort.Strings(names)
	verified := map[string]string{}
	defer clear(verified)
	for _, name := range names {
		err := keys.WithSource(ctx, name, func(path string, key []byte) error {
			salt, err := readDBSaltHex(path)
			if err != nil {
				return err
			}
			raw := hex.EncodeToString(key)
			ok, err := wcdb.VerifyEncKeyPage1(path, raw)
			if err != nil || !ok {
				return errors.New("a current source database no longer authenticates")
			}
			if previous, ok := verified[salt]; ok && previous != raw {
				return errors.New("conflicting material for one database salt")
			}
			verified[salt] = raw
			report.VerifiedDBs++
			return nil
		})
		if err != nil {
			return report, err
		}
	}
	report.UniqueKeys = len(verified)
	report.Update, retErr = config.UpdateProtectedAtPath(ctx, configPath, expectedConfigSHA, func(cfg *config.Config) error {
		if !strings.EqualFold(filepath.Clean(cfg.DBRoot), accountRoot) || cfg.Wxid == "" {
			return errors.New("configuration belongs to a different account")
		}
		report.AccountBound = true
		if cfg.KDF == "" {
			cfg.KDF = "pbkdf2-sha512-256000"
		}
		now := time.Now().Unix()
		for salt, raw := range verified {
			cfg.SetVerifiedKey(salt, raw, keys.provenance, now)
		}
		return ctx.Err()
	})
	return report, retErr
}
