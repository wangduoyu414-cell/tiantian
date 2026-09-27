//go:build windows

package wxkey

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
	"testing"

	"weixin-key/internal/config"
	"weixin-key/internal/testutil"
)

func TestVerifiedMaterialCacheProtectedAndResolverReusable(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "db_storage")
	if err := os.Mkdir(base, 0o700); err != nil {
		t.Fatal(err)
	}
	first, pass, salt1, key1 := testutil.MustNewPassphraseDB(base, "first.db", 1)
	second, salt2, key2 := testutil.MustNewPassphraseDBWith(base, "second.db", 1, pass)
	source, err := openProbeRoot(base, os.OpenRoot)
	if err != nil {
		t.Fatal(err)
	}
	keys := &VerifiedPassiveKeys{source: source, base: base, ready: true, provenance: "synthetic-material"}
	defer keys.clear()
	b1, _ := hex.DecodeString(key1)
	b2, _ := hex.DecodeString(key2)
	keys.put("first.db", b1)
	keys.put("second.db", b2)
	path := filepath.Join(t.TempDir(), "config.json")
	raw, _ := json.Marshal(config.Config{SchemaVersion: 4, Wxid: "wxid_synthetic", DBRoot: root})
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(raw)
	t.Setenv("WECHAT_CLI_CONFIG", path)
	report, err := CacheVerifiedPassiveKeys(context.Background(), keys, path, hex.EncodeToString(hash[:]))
	if err != nil || report.VerifiedDBs != 2 || report.UniqueKeys != 2 || !report.AccountBound ||
		!report.Update.Applied || !report.Update.ProtectedReadback {
		t.Fatal("protected cache failed", err)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil || bytes.Contains(onDisk, []byte(key1)) || bytes.Contains(onDisk, []byte(key2)) || bytes.Contains(onDisk, []byte(pass)) {
		t.Fatal("plaintext material escaped to configuration")
	}
	keys.clear() // resolver must now succeed without live material or a process
	resolver := NewKeyResolver()
	for i, db := range []string{first, second} {
		resolved, err := resolver.ResolveForDB(db, ResolveOptions{})
		if err != nil || resolved.Source != "config:cache" || resolved.Kind != MaterialEncKey ||
			resolved.SaltHex != []string{salt1, salt2}[i] || resolved.EncKeyHex != []string{key1, key2}[i] {
			t.Fatal("protected cache did not support offline resolver reuse")
		}
	}
	t.Setenv("WECHAT_CLI_PASSPHRASE_HEX", strings.Repeat("ff", 32))
	t.Setenv("WECHAT_CLI_CONFIG", filepath.Join(t.TempDir(), "poison.json"))
	t.Setenv("WECHAT_CLI_DB_ROOT", t.TempDir())
	offline, err := VerifyProtectedMaterialCache(context.Background(), root, path, report.Update.SHA256)
	if err != nil || offline.Status != "all-source-dbs-verified-from-protected-cache" ||
		offline.CacheHits != 2 || offline.Derivations != 0 || !offline.SourceStable {
		t.Fatal("cache-only diagnostic failed or used ambient material", err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(onDisk, after) {
		t.Fatal("read-only cache check changed configuration")
	}
	var escaped *VerifiedPassiveKeys
	calls := 0
	consumed, err := WithVerifiedProtectedCache(context.Background(), root, path, report.Update.SHA256, func(k *VerifiedPassiveKeys) error {
		calls++
		escaped = k
		if err := os.Rename(base, base+"-moved"); err == nil {
			t.Fatal("offline consumer failed to retain its verified source Root")
		}
		return k.WithSource(context.Background(), "first.db", func(source string, raw []byte) error {
			if source != first || hex.EncodeToString(raw) != key1 {
				t.Fatal("offline consumer got wrong source/material")
			}
			return nil
		})
	})
	if err != nil || calls != 1 || consumed.CacheHits != 2 {
		t.Fatal("offline material consumer failed", err)
	}
	if err := escaped.WithKey("first.db", func([]byte) error { t.Fatal("offline keys escaped callback"); return nil }); err == nil {
		t.Fatal("offline consumer remained usable")
	}
	if err := os.Rename(base, base+"-moved"); err != nil {
		t.Fatal("offline Root remained held", err)
	}
	if err := os.Rename(base+"-moved", base); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	_, err = WithVerifiedProtectedCache(ctx, root, path, report.Update.SHA256, func(k *VerifiedPassiveKeys) error { escaped = k; cancel(); return nil })
	if !errors.Is(err, context.Canceled) || escaped.ready || escaped.source != nil {
		t.Fatal("offline cancellation/cleanup failed")
	}
	for _, kind := range []string{"account", "hash", "cancel", "source-corruption"} {
		t.Run("offline-"+kind, func(t *testing.T) {
			selected, expected := root, report.Update.SHA256
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "account":
				selected = t.TempDir()
			case "hash":
				expected = strings.Repeat("00", 32)
			case "cancel":
				cancel()
			case "source-corruption":
				b, _ := os.ReadFile(second)
				b[100] ^= 1
				if err := os.WriteFile(second, b, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			r, err := VerifyProtectedMaterialCache(ctx, selected, path, expected)
			if err == nil || r.Status != "failed" {
				t.Fatal("unsafe offline verification accepted")
			}
		})
	}
}

func TestVerifiedMaterialCacheRejectsInactiveWrongAccountAndChangedDB(t *testing.T) {
	for _, kind := range []string{"inactive", "wrong-account", "changed-db"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			base := filepath.Join(root, "db_storage")
			if err := os.Mkdir(base, 0o700); err != nil {
				t.Fatal(err)
			}
			db, _, _, key := testutil.MustNewPassphraseDB(base, "db.db", 1)
			source, err := openProbeRoot(base, os.OpenRoot)
			if err != nil {
				t.Fatal(err)
			}
			keys := &VerifiedPassiveKeys{source: source, base: base, ready: true, provenance: "synthetic"}
			defer keys.clear()
			k, _ := hex.DecodeString(key)
			keys.put("db.db", k)
			cfgRoot := root
			if kind == "wrong-account" {
				cfgRoot = t.TempDir()
			}
			path := filepath.Join(t.TempDir(), "config.json")
			raw, _ := json.Marshal(config.Config{SchemaVersion: 4, Wxid: "synthetic", DBRoot: cfgRoot})
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			h := sha256.Sum256(raw)
			if kind == "inactive" {
				keys.ready = false
			}
			if kind == "changed-db" {
				b, _ := os.ReadFile(db)
				b[100] ^= 1
				if err := os.WriteFile(db, b, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			report, err := CacheVerifiedPassiveKeys(context.Background(), keys, path, hex.EncodeToString(h[:]))
			if err == nil || report.Update.Applied {
				t.Fatal("unsafe material persisted")
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(raw, after) {
				t.Fatal("failed cache changed configuration")
			}
		})
	}
}

func TestVerifiedMaterialCacheRejectsLegacyFallbackBehindEmptyTypedEntry(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "db_storage")
	if err := os.Mkdir(base, 0o700); err != nil {
		t.Fatal(err)
	}
	_, _, salt, key := testutil.MustNewPassphraseDB(base, "db.db", 1)
	path := filepath.Join(t.TempDir(), "config.json")
	raw, _ := json.Marshal(config.Config{SchemaVersion: 4, Wxid: "synthetic", DBRoot: root})
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(raw)
	proof, err := config.UpdateProtectedAtPath(context.Background(), path, hex.EncodeToString(hash[:]), func(cfg *config.Config) error {
		cfg.KDF = "pbkdf2-sha512-256000"
		cfg.SetVerifiedKey(salt, key, "synthetic", 123)
		entry := cfg.KeyEntries[salt]
		entry.EncKey = "" // legacy Keys is still correct; typed evidence is incomplete
		cfg.KeyEntries[salt] = entry
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	r, err := VerifyProtectedMaterialCache(context.Background(), root, path, proof.SHA256)
	if err == nil || r.Status != "failed" || r.CacheHits != 0 {
		t.Fatal("legacy fallback was falsely accepted as typed cache evidence")
	}
}
