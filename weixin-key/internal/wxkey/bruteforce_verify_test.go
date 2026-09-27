//go:build windows

package wxkey

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// TestSQLCipherVerifierAcceptsCorrectKey builds a synthetic SQLCipher 4 page-1
// using the same parameters WeChat 4.x uses (mac_key derived with dklen=32) and
// asserts that verify() accepts the real enc_key and rejects a wrong key. This
// guards the mac_key dklen regression (64 vs 32) that previously caused every
// correct candidate to be rejected.
func TestSQLCipherVerifierAcceptsCorrectKey(t *testing.T) {
	encKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(0xA0 + i)
	}
	salt := make([]byte, 16)
	for i := range salt {
		salt[i] = byte(0x11 + i)
	}

	// Construct a page: [0:16]=salt, [16:4032]=ciphertext+IV, [4032:4096]=MAC.
	page := make([]byte, sqlcipherPageSize)
	copy(page[:16], salt)
	for i := 16; i < sqlcipherPageSize-sqlcipherHMACSize; i++ {
		page[i] = byte(i * 7)
	}

	// Compute the MAC exactly as SQLCipher 4 does: mac_key = PBKDF2(enc_key,
	// salt^0x3a, 2, dklen=32); HMAC over page[16:4032] || LE32(pgno=1).
	hmacSalt := make([]byte, 16)
	for i := range salt {
		hmacSalt[i] = salt[i] ^ sqlcipherSaltMask
	}
	macKey := pbkdf2SHA512(encKey, hmacSalt, sqlcipherFastKDFIter, sqlcipherMACKeySize)
	if len(macKey) != sqlcipherMACKeySize {
		t.Fatalf("mac_key len = %d, want %d", len(macKey), sqlcipherMACKeySize)
	}
	region := page[16 : sqlcipherPageSize-sqlcipherHMACSize]
	hmacInput := make([]byte, 0, len(region)+4)
	hmacInput = append(hmacInput, region...)
	var pgno [4]byte
	binary.LittleEndian.PutUint32(pgno[:], 1)
	hmacInput = append(hmacInput, pgno[:]...)
	mac := hmac.New(sha512.New, macKey)
	mac.Write(hmacInput)
	copy(page[sqlcipherPageSize-sqlcipherHMACSize:], mac.Sum(nil))

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	if err := os.WriteFile(dbPath, page, 0o600); err != nil {
		t.Fatal(err)
	}

	v := newSQLCipherVerifier(dbPath, "dummy-salt-hex")
	if v == nil {
		t.Fatal("newSQLCipherVerifier returned nil for a well-formed page")
	}
	if !v.verify(encKey) {
		t.Fatal("verify rejected the correct enc_key (mac_key dklen regression?)")
	}

	wrong := make([]byte, 32)
	for i := range wrong {
		wrong[i] = byte(0x40 + i)
	}
	if v.verify(wrong) {
		t.Fatal("verify accepted a wrong enc_key")
	}
}
