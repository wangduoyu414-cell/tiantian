package wcdb

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/pbkdf2"
)

func TestVerifyEncKeyPage1AcceptsCorrectKey(t *testing.T) {
	encKey := make([]byte, EncKeyLength)
	for i := range encKey {
		encKey[i] = byte(0xA0 + i)
	}
	salt := make([]byte, SaltLength)
	for i := range salt {
		salt[i] = byte(0x11 + i)
	}

	page := make([]byte, SQLCipherPageSize)
	copy(page[:SaltLength], salt)
	for i := SaltLength; i < SQLCipherPageSize-SQLCipherHMACSize; i++ {
		page[i] = byte(i * 7)
	}
	hmacSalt := make([]byte, SaltLength)
	for i := range salt {
		hmacSalt[i] = salt[i] ^ SQLCipherSaltMask
	}
	macKey := pbkdf2.Key(encKey, hmacSalt, SQLCipherFastKDF, SQLCipherMACKeyLen, sha512.New)
	region := page[SaltLength : SQLCipherPageSize-SQLCipherHMACSize]
	var pgno [4]byte
	binary.LittleEndian.PutUint32(pgno[:], 1)
	mac := hmac.New(sha512.New, macKey)
	mac.Write(region)
	mac.Write(pgno[:])
	copy(page[SQLCipherPageSize-SQLCipherHMACSize:], mac.Sum(nil))

	// Guard the dklen=32 requirement: a 64-byte mac_key must NOT verify.
	wrongMACKey := pbkdf2.Key(encKey, hmacSalt, SQLCipherFastKDF, 64, sha512.New)
	wrongMAC := hmac.New(sha512.New, wrongMACKey)
	wrongMAC.Write(region)
	wrongMAC.Write(pgno[:])
	if hmac.Equal(wrongMAC.Sum(nil), page[SQLCipherPageSize-SQLCipherHMACSize:]) {
		t.Fatal("unexpected: dklen=64 MAC matched page built with dklen=32")
	}

	dbPath := filepath.Join(t.TempDir(), "page1.db")
	if err := os.WriteFile(dbPath, page, 0o600); err != nil {
		t.Fatal(err)
	}

	ok, err := VerifyEncKeyPage1(dbPath, hex.EncodeToString(encKey))
	if err != nil {
		t.Fatalf("VerifyEncKeyPage1: %v", err)
	}
	if !ok {
		t.Fatal("VerifyEncKeyPage1 rejected correct enc_key")
	}

	wrong := make([]byte, EncKeyLength)
	for i := range wrong {
		wrong[i] = byte(0x40 + i)
	}
	ok, err = VerifyEncKeyPage1(dbPath, hex.EncodeToString(wrong))
	if err != nil {
		t.Fatalf("VerifyEncKeyPage1 wrong key: %v", err)
	}
	if ok {
		t.Fatal("VerifyEncKeyPage1 accepted wrong enc_key")
	}
}

func TestVerifyPassphrasePage1(t *testing.T) {
	pass := make([]byte, PassphraseLength)
	for i := range pass {
		pass[i] = byte(i + 3)
	}
	salt := make([]byte, SaltLength)
	for i := range salt {
		salt[i] = byte(0x80 + i)
	}
	encKey := pbkdf2.Key(pass, salt, DefaultKDFIters, EncKeyLength, sha512.New)

	page := make([]byte, SQLCipherPageSize)
	copy(page[:SaltLength], salt)
	for i := SaltLength; i < SQLCipherPageSize-SQLCipherHMACSize; i++ {
		page[i] = byte(i)
	}
	hmacSalt := make([]byte, SaltLength)
	for i := range salt {
		hmacSalt[i] = salt[i] ^ SQLCipherSaltMask
	}
	macKey := pbkdf2.Key(encKey, hmacSalt, SQLCipherFastKDF, SQLCipherMACKeyLen, sha512.New)
	region := page[SaltLength : SQLCipherPageSize-SQLCipherHMACSize]
	var pgno [4]byte
	binary.LittleEndian.PutUint32(pgno[:], 1)
	mac := hmac.New(sha512.New, macKey)
	mac.Write(region)
	mac.Write(pgno[:])
	copy(page[SQLCipherPageSize-SQLCipherHMACSize:], mac.Sum(nil))

	dbPath := filepath.Join(t.TempDir(), "pp.db")
	if err := os.WriteFile(dbPath, page, 0o600); err != nil {
		t.Fatal(err)
	}

	got, ok, err := VerifyPassphrasePage1(dbPath, hex.EncodeToString(pass))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("passphrase page1 verify failed")
	}
	if got != hex.EncodeToString(encKey) {
		t.Fatalf("derived enc_key mismatch")
	}
}
