//go:build windows

package wxkey

import (
	"testing"
	"time"

	"weixin-key/internal/testutil"
)

// Regression test for the verified type-confusion bug: a candidate that only
// verifies as a *passphrase* (via PBKDF2 derivation) must have the DERIVED
// enc_key recorded for the salt - storing the passphrase itself as the
// per-DB enc_key makes later opens fail even though "setup verified OK".
func TestVerifyCandidateKeysStoresDerivedEncKeyNotPassphrase(t *testing.T) {
	dir := t.TempDir()
	dbPath, passHex, saltHex, wantEncKey := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)

	dbs := []windowsSourceDB{{rel: "msg_0.db", path: dbPath, salt: saltHex}}
	salts := map[string]bool{saltHex: true}
	scan := newSetupScan()

	if err := windowsVerifyCandidateKeys([]string{passHex}, dbs, salts, scan, time.Time{}); err != nil {
		t.Fatalf("windowsVerifyCandidateKeys: %v", err)
	}
	got, ok := scan.found[saltHex]
	if !ok {
		t.Fatal("passphrase candidate did not verify against its own DB")
	}
	if got != wantEncKey {
		t.Fatalf("found[%s] = %q, want derived enc_key %q (passphrase stored as enc_key)", saltHex, got, wantEncKey)
	}
	if scan.passphraseHex != passHex {
		t.Fatal("verified passphrase was not captured for config caching")
	}
}

// A candidate that IS the raw enc_key must be stored unchanged.
func TestVerifyCandidateKeysStoresRawEncKeyAsIs(t *testing.T) {
	dir := t.TempDir()
	dbPath, _, saltHex, encKeyHex := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)

	dbs := []windowsSourceDB{{rel: "msg_0.db", path: dbPath, salt: saltHex}}
	salts := map[string]bool{saltHex: true}
	scan := newSetupScan()

	if err := windowsVerifyCandidateKeys([]string{encKeyHex}, dbs, salts, scan, time.Time{}); err != nil {
		t.Fatalf("windowsVerifyCandidateKeys: %v", err)
	}
	if got := scan.found[saltHex]; got != encKeyHex {
		t.Fatalf("found[%s] = %q, want raw enc_key %q", saltHex, got, encKeyHex)
	}
}

// A wrong candidate must match nothing and record nothing.
func TestVerifyCandidateKeysRejectsWrongMaterial(t *testing.T) {
	dir := t.TempDir()
	dbPath, _, saltHex, _ := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)

	dbs := []windowsSourceDB{{rel: "msg_0.db", path: dbPath, salt: saltHex}}
	salts := map[string]bool{saltHex: true}
	scan := newSetupScan()

	wrong := "00"
	for len(wrong) < 64 {
		wrong += "11"
	}
	if err := windowsVerifyCandidateKeys([]string{wrong}, dbs, salts, scan, time.Time{}); err != nil {
		t.Fatalf("windowsVerifyCandidateKeys: %v", err)
	}
	if len(scan.found) != 0 {
		t.Fatalf("wrong candidate recorded %d entries, want 0", len(scan.found))
	}
}
