//go:build windows

package wxkey

import (
	"context"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
	"weixin-key/internal/config"
)

type MaterialCacheVerification struct {
	Status       string `json:"status"`
	SourceDBs    int    `json:"source_dbs"`
	CacheHits    int    `json:"cache_hits"`
	SourceStable bool   `json:"source_page1_stable"`
	Derivations  int    `json:"derivations"`
}

// VerifyProtectedMaterialCache has no process, capture or persistence path.
// It exercises the existing resolver against every discovered encrypted DB,
// using only an explicitly digest-bound protected config, never env material.
func VerifyProtectedMaterialCache(ctx context.Context, accountRoot, configPath, expectedSHA string) (report MaterialCacheVerification, err error) {
	return verifyProtectedMaterialCache(ctx, accountRoot, configPath, expectedSHA, nil)
}

// WithVerifiedProtectedCache permits synchronous offline consumption only
// after the same complete typed-cache verification. The original source Root
// remains pinned through the callback; no process discovery or persistence.
func WithVerifiedProtectedCache(ctx context.Context, accountRoot, configPath, expectedSHA string,
	consume func(*VerifiedPassiveKeys) error) (MaterialCacheVerification, error) {
	if consume == nil {
		return MaterialCacheVerification{Status: "failed"}, errors.New("cache consumer is required")
	}
	return verifyProtectedMaterialCache(ctx, accountRoot, configPath, expectedSHA, consume)
}

func verifyProtectedMaterialCache(ctx context.Context, accountRoot, configPath, expectedSHA string,
	consume func(*VerifiedPassiveKeys) error) (report MaterialCacheVerification, err error) {
	report.Status = "failed"
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if !filepath.IsAbs(accountRoot) {
		return report, errors.New("explicit account root required")
	}
	cfg, err := config.ReadProtectedAtPath(configPath, expectedSHA)
	if err != nil {
		return report, err
	}
	defer func() {
		clear(cfg.Keys)
		clear(cfg.KeyEntries)
		cfg.Passphrase, cfg.ImageKey, cfg.Key = "", "", ""
	}()
	if !strings.EqualFold(filepath.Clean(cfg.DBRoot), filepath.Clean(accountRoot)) || cfg.Wxid == "" {
		return report, errors.New("cache belongs to a different account")
	}
	// A passphrase is not a valid fallback for this cache-only check.
	cfg.Passphrase = ""
	resolver := &KeyResolver{cfg: cfg, memo: map[string]string{}}
	base := filepath.Join(accountRoot, "db_storage")
	source, err := openProbeRoot(base, os.OpenRoot)
	if err != nil {
		return report, err
	}
	keys := &VerifiedPassiveKeys{source: source, base: base, provenance: "protected-config-cache"}
	defer keys.clear()
	dbs, err := probeSourceDBsInRoot(ctx, source, base)
	if err != nil {
		return report, err
	}
	report.SourceDBs = len(dbs)
	if len(dbs) == 0 || len(dbs) > 64 {
		return report, errors.New("cache check requires 1..64 encrypted databases")
	}
	var before []*sqlcipherVerifier
	for _, db := range dbs {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		v := readProbeVerifier(source, db)
		if v == nil {
			return report, errors.New("source verification page unavailable")
		}
		before = append(before, v)
		entry, ok := cfg.KeyEntries[db.salt]
		if !ok || entry.EncKey == "" || entry.Kind != "enc_key" || entry.WxID != cfg.Wxid || entry.KDF != cfg.KDF ||
			entry.Source == "" || entry.VerifiedAt == 0 {
			return report, errors.New("cache is missing a typed account-bound entry")
		}
		err := func() error {
			f, err := source.Open(db.rel)
			if err != nil {
				return err
			}
			defer f.Close()
			var opened windows.ByHandleFileInformation
			if err := windows.GetFileInformationByHandle(windows.Handle(f.Fd()), &opened); err != nil {
				return err
			}
			pin, current, err := openConsumerSourceIdentity(db.path, false)
			if err != nil {
				return err
			}
			defer windows.CloseHandle(pin)
			if !sameProbeDirectory(opened, current) {
				return errors.New("cache source path changed")
			}
			resolved, err := resolver.ResolveForDB(db.path, ResolveOptions{NoEnvironment: true})
			if err != nil {
				return err
			}
			defer func() { resolved.EncKeyHex = "" }()
			if resolved.Source != "config:cache" || resolved.Kind != MaterialEncKey || resolved.EncKeyHex != entry.EncKey {
				return errors.New("offline check used something other than cached raw material")
			}
			if consume != nil {
				raw, err := hex.DecodeString(resolved.EncKeyHex)
				if err != nil || len(raw) != 32 {
					clear(raw)
					return errors.New("verified cached material has invalid encoding")
				}
				keys.put(db.rel, raw)
				clear(raw)
			}
			report.CacheHits++
			return ctx.Err()
		}()
		if err != nil {
			return report, err
		}
	}
	report.SourceStable, err = recheckProbeSources(ctx, dbs, before, func(db windowsSourceDB) *sqlcipherVerifier {
		return readProbeVerifier(source, db)
	})
	if err != nil {
		return report, err
	}
	report.Derivations = resolver.deriveN
	if report.CacheHits != report.SourceDBs || resolver.deriveN != 0 {
		return report, errors.New("cache coverage incomplete or derivation was unexpectedly used")
	}
	report.Status = "all-source-dbs-verified-from-protected-cache"
	if consume != nil {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		if len(keys.keys) != len(dbs) {
			return report, errors.New("offline consumer source coverage is ambiguous")
		}
		keys.ready = true
		return report, errors.Join(consume(keys), ctx.Err())
	}
	return report, nil
}
