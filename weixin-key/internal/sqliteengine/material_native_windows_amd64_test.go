//go:build windows && amd64

package sqliteengine

import (
	"bytes"
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"

	"weixin-key/internal/wcdb"
	"weixin-key/internal/wxkey"
)

// Test-only fixture creator: the pinned native engine does the KDF, salt
// generation, encryption and HMAC, not the Go synthetic cipher builder.
func nativeBinaryPassphraseFixture(t *testing.T, pass []byte) (string, string) {
	t.Helper()
	if err := initialize(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "native-binary-passphrase.db")
	var h uintptr
	if rc := api.open(path, &h, 2|4|0x10000, 0); rc != 0 {
		if h != 0 {
			api.close(h)
		}
		t.Fatalf("synthetic native open rc=%d", rc)
	}
	defer api.close(h)
	if api.config(h, "cipher", 4) < 0 {
		t.Fatal("native SQLCipher adapter unavailable")
	}
	for _, p := range []struct {
		name  string
		value int32
	}{
		{"legacy", 4}, {"legacy_page_size", 4096}, {"kdf_iter", 256000}, {"fast_kdf_iter", 2},
		{"hmac_use", 1}, {"hmac_pgno", 1}, {"hmac_salt_mask", 0x3a}, {"kdf_algorithm", 2}, {"hmac_algorithm", 2},
	} {
		if api.cipherConfig(h, "sqlcipher", p.name, p.value) < 0 {
			t.Fatal("native cipher setting rejected")
		}
	}
	// Explicit byte length, containing NUL and high-bit bytes: neither a
	// C-string nor hex text nor the native raw-key prefix.
	rc := api.key(h, "main", unsafe.Pointer(&pass[0]), int32(len(pass)))
	runtime.KeepAlive(pass)
	if rc != 0 {
		t.Fatalf("synthetic native binary key rc=%d", rc)
	}
	nativeExec(t, h, "PRAGMA journal_mode=DELETE; CREATE TABLE messages(id INTEGER PRIMARY KEY, body TEXT); INSERT INTO messages VALUES(1,'synthetic binary passphrase')")
	b, err := os.ReadFile(path)
	if err != nil || len(b) < 8192 || bytes.Equal(b[:16], []byte("SQLite format 3\x00")) {
		t.Fatal("native fixture not encrypted or not persisted")
	}
	return path, hex.EncodeToString(b[:16])
}

func TestNativeBinaryPassphraseIndependentNormalization(t *testing.T) {
	pass, mask := make([]byte, 32), make([]byte, 32)
	for i := range pass {
		pass[i], mask[i] = byte(i*11), byte(255-i*3)
	}
	observed, err := wxkey.NormalizeObservedPassphrase(pass, mask)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := wxkey.NormalizeObservedPassphrase(observed[:], mask)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(recovered[:])
	first, salt1 := nativeBinaryPassphraseFixture(t, pass)
	second, salt2 := nativeBinaryPassphraseFixture(t, pass)
	if salt1 == salt2 {
		t.Fatal("native engine did not generate distinct salts")
	}
	var previousKey string
	for i, path := range []string{first, second} {
		salt := []string{salt1, salt2}[i]
		before := digest(t, path)
		v, err := wxkey.VerifyKeyCandidate(path, hex.EncodeToString(recovered[:]), salt)
		if err != nil || !v.Match || v.Kind != wxkey.MaterialPassphrase || v.EncKeyHex == v.PassphraseHex {
			t.Fatal("Go verification disagrees with native binary-passphrase fixture")
		}
		if previousKey == v.EncKeyHex {
			t.Fatal("different salts reused derived key")
		}
		previousKey = v.EncKeyHex
		// A full native read-transaction backup and Go all-page verification,
		// then an independent plain SQLite query, not just a header check.
		out := filepath.Join(t.TempDir(), "plain.db")
		if err := Backup(context.Background(), path, out, v.EncKeyHex, salt); err != nil {
			t.Fatal(err)
		}
		if queryCount(t, out) != 1 {
			t.Fatal("native fixture content missing")
		}
		raw, err := wxkey.VerifyKeyCandidate(path, v.EncKeyHex, salt)
		if err != nil || !raw.Match || raw.Kind != wxkey.MaterialEncKey || raw.PassphraseHex != "" {
			t.Fatal("derived key type confused with passphrase")
		}
		if ok, err := wcdb.VerifyEncKeyPage1(path, hex.EncodeToString(recovered[:])); err != nil || ok {
			t.Fatal("binary passphrase accepted as raw encryption key")
		}
		badMask := append([]byte(nil), mask...)
		badMask[0] ^= 1
		wrong, _ := wxkey.NormalizeObservedPassphrase(observed[:], badMask)
		for _, input := range [][]byte{observed[:], wrong[:]} {
			if v, err := wxkey.VerifyKeyCandidate(path, hex.EncodeToString(input), salt); err != nil || v.Match {
				t.Fatal("wrapped or wrongly normalized candidate accepted")
			}
		}
		if digest(t, path) != before {
			t.Fatal("verification changed native encrypted source")
		}
		corrupt, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		corrupt[120] ^= 1
		bad := filepath.Join(t.TempDir(), "corrupt.db")
		if err := os.WriteFile(bad, corrupt, 0o600); err != nil {
			t.Fatal(err)
		}
		if v, err := wxkey.VerifyKeyCandidate(bad, hex.EncodeToString(recovered[:]), salt); err != nil || v.Match {
			t.Fatal("corrupt native page accepted")
		}
	}
}
