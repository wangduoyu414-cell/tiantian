package wxkey

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"weixin-key/internal/config"
	"weixin-key/internal/wcdb"
)

// ErrNoKeyMaterial means no candidate material exists at all (nothing cached,
// nothing configured): the only way forward is a capture run (setup).
var ErrNoKeyMaterial = errors.New("no key material available")

// ErrKeyMaterialRejected means material was present but failed verification
// against the target DB (stale cache, wrong account, rotated passphrase). The
// caller should re-capture rather than retry the same material.
var ErrKeyMaterialRejected = errors.New("key material failed verification")

// ResolveOptions carries explicit key material. Explicit choices always win
// over the config cache.
type ResolveOptions struct {
	// EncKeyHex is an explicit per-DB raw enc_key (64 hex chars).
	EncKeyHex string
	// PassphraseHex is an explicit account passphrase (64 hex chars).
	PassphraseHex string
	// NoConfig disables the config cache + stored passphrase fallback.
	NoConfig bool
	// NoEnvironment disables ambient passphrase selection for explicitly
	// bound offline diagnostics. Existing callers keep their default order.
	NoEnvironment bool
}

// ResolvedKey is the verified per-DB enc_key plus provenance. It never
// carries the passphrase itself.
type ResolvedKey struct {
	DBPath    string
	SaltHex   string
	EncKeyHex string
	// Kind describes the source material (enc_key or passphrase).
	Kind MaterialKind
	// Source: "--enc-key", "--passphrase", "env:WECHAT_CLI_PASSPHRASE_HEX",
	// "config:cache", "config:passphrase".
	Source string
}

// KeyResolver is the single key-resolution service shared by decrypt/export
// (and the GUI later). It never starts, stops, or scans any process: all
// paths are offline. Resolution order per DB:
//
//  1. explicit --enc-key
//  2. explicit --passphrase (derive per-DB enc_key)
//  3. WECHAT_CLI_PASSPHRASE_HEX
//  4. config cache: per-salt enc_key, re-verified against the current file
//  5. config passphrase (derive per-DB enc_key)
//
// Cached values are always re-verified against the live DB page 1, so a
// rotated passphrase or a rewritten DB is detected instead of trusted.
type KeyResolver struct {
	cfg    *config.Config
	cfgErr error
	// memo deduplicates PBKDF2 derivations within one job: one derivation per
	// (passphrase, salt) pair, shared by every consumer of this resolver.
	memo map[string]string
	// deriveN counts real PBKDF2 runs (test instrumentation).
	deriveN int
}

// NewKeyResolver returns a resolver that loads the config lazily on first use.
func NewKeyResolver() *KeyResolver {
	return &KeyResolver{memo: map[string]string{}}
}

func (r *KeyResolver) loadConfig() (*config.Config, error) {
	if r.cfg == nil && r.cfgErr == nil {
		r.cfg, r.cfgErr = config.Load()
	}
	return r.cfg, r.cfgErr
}

func (r *KeyResolver) derive(passphraseHex, saltHex string) (string, error) {
	k := passphraseHex + "|" + saltHex
	if v, ok := r.memo[k]; ok {
		return v, nil
	}
	enc, err := wcdb.DeriveEncKey(passphraseHex, saltHex)
	if err != nil {
		return "", err
	}
	r.deriveN++
	r.memo[k] = enc
	return enc, nil
}

// ResolveForDB returns the verified per-DB enc_key for dbPath, or a typed
// error: ErrNoKeyMaterial (capture needed) vs ErrKeyMaterialRejected (stale
// material, re-capture advised).
func (r *KeyResolver) ResolveForDB(dbPath string, opts ResolveOptions) (*ResolvedKey, error) {
	saltHex, err := readDBSaltHex(dbPath)
	if err != nil {
		return nil, fmt.Errorf("read salt from %s: %w", dbPath, err)
	}
	saltHex = strings.ToLower(saltHex)

	// 1. Explicit enc_key: fail fast when wrong (explicit choice, no fallback).
	if raw := strings.TrimSpace(opts.EncKeyHex); raw != "" {
		k, nerr := normalizeEncKeyHex(raw)
		if nerr != nil {
			return nil, fmt.Errorf("--enc-key: %w", nerr)
		}
		ok, verr := wcdb.VerifyEncKeyPage1(dbPath, k)
		if verr != nil {
			return nil, fmt.Errorf("verify --enc-key against %s: %w", dbPath, verr)
		}
		if !ok {
			return nil, fmt.Errorf("%w: --enc-key does not match %s", ErrKeyMaterialRejected, dbPath)
		}
		return &ResolvedKey{DBPath: dbPath, SaltHex: saltHex, EncKeyHex: k, Kind: MaterialEncKey, Source: "--enc-key"}, nil
	}

	// 2. Explicit passphrase: fail fast when wrong.
	if raw := strings.TrimSpace(opts.PassphraseHex); raw != "" {
		return r.resolvePassphrase(dbPath, saltHex, raw, "--passphrase")
	}

	var rejected []string

	// 3. Env passphrase.
	if raw := strings.TrimSpace(os.Getenv("WECHAT_CLI_PASSPHRASE_HEX")); raw != "" && !opts.NoEnvironment {
		res, rerr := r.resolvePassphrase(dbPath, saltHex, raw, "env:WECHAT_CLI_PASSPHRASE_HEX")
		if rerr == nil {
			return res, nil
		}
		rejected = append(rejected, rerr.Error())
	}

	if !opts.NoConfig {
		cfg, cerr := r.loadConfig()
		if cerr != nil {
			return nil, fmt.Errorf("load material configuration: %w", cerr)
		}
		if cerr == nil && cfg != nil {
			// 4. Cached per-salt enc_key (schema-4 key_entries, schema-2 keys
			// fallback), always re-verified against the current file.
			if k, ok := lookupCachedKey(cfg, saltHex); ok {
				okv, verr := wcdb.VerifyEncKeyPage1(dbPath, k)
				if verr != nil {
					return nil, fmt.Errorf("verify cached key against %s: %w", dbPath, verr)
				}
				if okv {
					return &ResolvedKey{DBPath: dbPath, SaltHex: saltHex, EncKeyHex: k, Kind: MaterialEncKey, Source: "config:cache"}, nil
				}
				rejected = append(rejected, "cached enc_key for salt "+saltHex+" failed verification (stale cache)")
			}
			// 5. Config passphrase.
			if cfg.HasPassphrase() {
				res, rerr := r.resolvePassphrase(dbPath, saltHex, cfg.Passphrase, "config:passphrase")
				if rerr == nil {
					return res, nil
				}
				rejected = append(rejected, rerr.Error())
			}
		}
	}

	if len(rejected) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrKeyMaterialRejected, strings.Join(rejected, "; "))
	}
	return nil, fmt.Errorf("%w for %s (salt %s): pass --enc-key/--passphrase, set WECHAT_CLI_PASSPHRASE_HEX, or run `weixin-key setup`",
		ErrNoKeyMaterial, dbPath, saltHex)
}

func (r *KeyResolver) resolvePassphrase(dbPath, saltHex, raw, source string) (*ResolvedKey, error) {
	pp, err := normalizePassphraseHex(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}
	enc, err := r.derive(pp, saltHex)
	if err != nil {
		return nil, err
	}
	ok, err := wcdb.VerifyEncKeyPage1(dbPath, enc)
	if err != nil {
		return nil, fmt.Errorf("verify passphrase from %s against %s: %w", source, dbPath, err)
	}
	if !ok {
		return nil, fmt.Errorf("%w: passphrase from %s does not match %s", ErrKeyMaterialRejected, source, dbPath)
	}
	return &ResolvedKey{DBPath: dbPath, SaltHex: saltHex, EncKeyHex: enc, Kind: MaterialPassphrase, Source: source}, nil
}

// lookupCachedKey prefers schema-4 key entries and falls back to the legacy
// schema-2 keys map.
func lookupCachedKey(cfg *config.Config, saltHex string) (string, bool) {
	if cfg == nil {
		return "", false
	}
	if e, ok := cfg.KeyEntries[saltHex]; ok && e.EncKey != "" {
		return e.EncKey, true
	}
	if k, ok := cfg.Keys[saltHex]; ok && k != "" {
		return k, true
	}
	return "", false
}
