package wxkey

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"weixin-key/internal/testutil"
)

// writeConfig writes a raw config JSON and points the resolver at it.
func writeConfig(t *testing.T, json string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(json), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WECHAT_CLI_CONFIG", path)
	return path
}

// Acceptance anchor: one passphrase derives the correct, DIFFERENT enc_key
// for each DB salt, and both resolve offline (no WeChat process involved).
func TestResolverConfigPassphraseDerivesPerSalt(t *testing.T) {
	dir := t.TempDir()
	db1, pp, salt1, enc1 := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)
	db2, salt2, enc2 := testutil.MustNewPassphraseDBWith(dir, "msg_1.db", 2, pp)
	if salt1 == salt2 || enc1 == enc2 {
		t.Fatal("fixture error: expected distinct salts and enc_keys")
	}
	writeConfig(t, fmt.Sprintf(`{"schema_version":3,"passphrase":%q,"kdf":"pbkdf2-sha512-256000"}`, pp))

	r := NewKeyResolver()
	k1, err := r.ResolveForDB(db1, ResolveOptions{})
	if err != nil {
		t.Fatalf("resolve db1: %v", err)
	}
	if k1.EncKeyHex != enc1 || k1.Source != "config:passphrase" || k1.Kind != MaterialPassphrase {
		t.Fatalf("db1 resolved %+v", k1)
	}
	k2, err := r.ResolveForDB(db2, ResolveOptions{})
	if err != nil {
		t.Fatalf("resolve db2: %v", err)
	}
	if k2.EncKeyHex != enc2 {
		t.Fatalf("db2 enc_key = %q, want %q", k2.EncKeyHex, enc2)
	}
	// Two salts -> exactly two derivations; resolving again must hit the memo.
	if r.deriveN != 2 {
		t.Fatalf("deriveN = %d, want 2", r.deriveN)
	}
	if _, err := r.ResolveForDB(db1, ResolveOptions{}); err != nil {
		t.Fatalf("re-resolve db1: %v", err)
	}
	if r.deriveN != 2 {
		t.Fatalf("deriveN = %d after memo hit, want 2", r.deriveN)
	}
}

// Cached per-salt enc_key is preferred and verified offline; no derivation.
func TestResolverUsesCachedKeyEntry(t *testing.T) {
	dir := t.TempDir()
	db, _, salt, enc := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)
	writeConfig(t, fmt.Sprintf(`{"schema_version":4,"keys":{%q:%q},"key_entries":{%q:{"enc_key":%q,"source":"test"}}}`, salt, enc, salt, enc))

	r := NewKeyResolver()
	k, err := r.ResolveForDB(db, ResolveOptions{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if k.EncKeyHex != enc || k.Source != "config:cache" || k.Kind != MaterialEncKey {
		t.Fatalf("resolved %+v", k)
	}
	if r.deriveN != 0 {
		t.Fatalf("cache hit should not derive, deriveN = %d", r.deriveN)
	}
}

// Legacy schema-2 configs (keys map only, no key_entries) still resolve.
func TestResolverReadsLegacyKeysMap(t *testing.T) {
	dir := t.TempDir()
	db, _, salt, enc := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)
	writeConfig(t, fmt.Sprintf(`{"schema_version":2,"keys":{%q:%q}}`, salt, enc))

	r := NewKeyResolver()
	k, err := r.ResolveForDB(db, ResolveOptions{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if k.EncKeyHex != enc || k.Source != "config:cache" {
		t.Fatalf("resolved %+v", k)
	}
}

// A stale cache entry (key no longer matches the file) must not be trusted;
// the resolver falls through to the config passphrase.
func TestResolverStaleCacheFallsBackToPassphrase(t *testing.T) {
	dir := t.TempDir()
	db, pp, salt, _ := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)
	stale := "00"
	for len(stale) < 64 {
		stale += "11"
	}
	writeConfig(t, fmt.Sprintf(`{"schema_version":3,"passphrase":%q,"kdf":"pbkdf2-sha512-256000","keys":{%q:%q}}`, pp, salt, stale))

	r := NewKeyResolver()
	k, err := r.ResolveForDB(db, ResolveOptions{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if k.Source != "config:passphrase" || k.Kind != MaterialPassphrase {
		t.Fatalf("expected fallback to config passphrase, got %+v", k)
	}
}

// Material present but wrong = rejected (re-capture), not "no material".
func TestResolverStaleMaterialIsRejectedNotMissing(t *testing.T) {
	dir := t.TempDir()
	db, _, salt, _ := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)
	stale := "22"
	for len(stale) < 64 {
		stale += "33"
	}
	writeConfig(t, fmt.Sprintf(`{"schema_version":2,"keys":{%q:%q}}`, salt, stale))

	r := NewKeyResolver()
	_, err := r.ResolveForDB(db, ResolveOptions{})
	if !errors.Is(err, ErrKeyMaterialRejected) {
		t.Fatalf("err = %v, want ErrKeyMaterialRejected", err)
	}
	if errors.Is(err, ErrNoKeyMaterial) {
		t.Fatal("stale cache must not be reported as missing material")
	}
}

// No config at all = ErrNoKeyMaterial (the capture-needed signal).
func TestResolverNoMaterial(t *testing.T) {
	dir := t.TempDir()
	db, _, _, _ := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)
	writeConfig(t, `{}`)

	r := NewKeyResolver()
	_, err := r.ResolveForDB(db, ResolveOptions{})
	if !errors.Is(err, ErrNoKeyMaterial) {
		t.Fatalf("err = %v, want ErrNoKeyMaterial", err)
	}
}

func TestResolverProtectedConfigFailureIsNotMissingMaterial(t *testing.T) {
	dir := t.TempDir()
	db, _, _, _ := testutil.MustNewPassphraseDB(dir, "msg_0.db", 1)
	writeConfig(t, `{"schema_version":5,"protected_secrets":{"scheme":"dpapi-user-v1","ciphertext":"AQID"}}`)
	_, err := NewKeyResolver().ResolveForDB(db, ResolveOptions{})
	if err == nil || errors.Is(err, ErrNoKeyMaterial) || errors.Is(err, ErrKeyMaterialRejected) {
		t.Fatalf("protected configuration failure would trigger capture: %v", err)
	}
}

// Explicit flags beat config; a wrong explicit enc_key fails fast.
func TestResolverExplicitEncKeyWinsAndFails(t *testing.T) {
	dir := t.TempDir()
	db, pp, salt, enc := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)
	writeConfig(t, fmt.Sprintf(`{"schema_version":3,"passphrase":%q,"kdf":"pbkdf2-sha512-256000"}`, pp))

	r := NewKeyResolver()
	k, err := r.ResolveForDB(db, ResolveOptions{EncKeyHex: enc})
	if err != nil || k.Source != "--enc-key" {
		t.Fatalf("explicit enc_key: %+v err=%v", k, err)
	}
	wrong := "ab"
	for len(wrong) < 64 {
		wrong += "cd"
	}
	if _, err := r.ResolveForDB(db, ResolveOptions{EncKeyHex: wrong}); !errors.Is(err, ErrKeyMaterialRejected) {
		t.Fatalf("wrong explicit enc_key: err = %v, want ErrKeyMaterialRejected", err)
	}
	_ = salt
}

// NoConfig disables the config fallback entirely.
func TestResolverNoConfigSkipsCache(t *testing.T) {
	dir := t.TempDir()
	db, _, salt, enc := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)
	writeConfig(t, fmt.Sprintf(`{"schema_version":2,"keys":{%q:%q}}`, salt, enc))

	r := NewKeyResolver()
	if _, err := r.ResolveForDB(db, ResolveOptions{NoConfig: true}); !errors.Is(err, ErrNoKeyMaterial) {
		t.Fatalf("err = %v, want ErrNoKeyMaterial with NoConfig", err)
	}
}
