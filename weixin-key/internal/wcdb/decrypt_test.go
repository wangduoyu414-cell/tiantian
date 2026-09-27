package wcdb

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/pbkdf2"
)

// buildEncryptedTestDB encrypts plaintext pages with the SQLCipher 4 / WeChat
// 4.1+ layout so DecryptDB can be tested round-trip.
func buildEncryptedTestDB(t *testing.T, dir string, encKey, salt []byte, plaintextPages [][]byte) string {
	t.Helper()
	if len(salt) != SaltLength {
		t.Fatalf("test salt must be %d bytes", SaltLength)
	}
	var buf bytes.Buffer
	block, err := aes.NewCipher(encKey)
	if err != nil {
		t.Fatal(err)
	}
	hmacSalt := make([]byte, SaltLength)
	for i := range salt {
		hmacSalt[i] = salt[i] ^ SQLCipherSaltMask
	}
	macKey := pbkdf2.Key(encKey, hmacSalt, SQLCipherFastKDF, SQLCipherMACKeyLen, sha512.New)
	ivOff := SQLCipherPageSize - SQLCipherReserve

	for pgno, plain := range plaintextPages {
		if len(plain) > ivOff {
			t.Fatalf("plaintext page %d too large", pgno+1)
		}
		page := make([]byte, SQLCipherPageSize)
		iv := make([]byte, SQLCipherIVSize)
		if _, err := rand.Read(iv); err != nil {
			t.Fatal(err)
		}
		start := 0
		if pgno == 0 {
			// Page 1: salt replaces the SQLite header in the encrypted file.
			copy(page, salt)
			start = SaltLength
		}
		cipher.NewCBCEncrypter(block, iv).CryptBlocks(page[start:ivOff], plain[start:ivOff])
		copy(page[ivOff:ivOff+SQLCipherIVSize], iv)

		var pg [4]byte
		binary.LittleEndian.PutUint32(pg[:], uint32(pgno+1))
		mac := hmac.New(sha512.New, macKey)
		mac.Write(page[start : SQLCipherPageSize-SQLCipherHMACSize])
		mac.Write(pg[:])
		copy(page[SQLCipherPageSize-SQLCipherHMACSize:], mac.Sum(nil))
		buf.Write(page)
	}

	path := filepath.Join(dir, "encrypted.db")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDecryptDBRoundTrip(t *testing.T) {
	encKey := make([]byte, EncKeyLength)
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(encKey); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}

	// Page 1 plaintext starts after the (decrypted) SQLite header.
	page1 := make([]byte, SQLCipherPageSize-SQLCipherReserve)
	copy(page1, sqliteHeader)
	for i := SaltLength; i < len(page1); i++ {
		page1[i] = byte(i * 3)
	}
	page2 := make([]byte, SQLCipherPageSize-SQLCipherReserve)
	for i := range page2 {
		page2[i] = byte(i * 5)
	}
	page3 := make([]byte, SQLCipherPageSize-SQLCipherReserve)
	for i := range page3 {
		page3[i] = byte(i * 11)
	}

	dir := t.TempDir()
	encPath := buildEncryptedTestDB(t, dir, encKey, salt, [][]byte{page1, page2, page3})
	outPath := filepath.Join(dir, "plain.db")

	pages, err := DecryptDB(encPath, outPath, encKey)
	if err != nil {
		t.Fatalf("DecryptDB: %v", err)
	}
	if pages != 3 {
		t.Fatalf("pages = %d, want 3", pages)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3*SQLCipherPageSize {
		t.Fatalf("output size = %d, want %d", len(got), 3*SQLCipherPageSize)
	}
	// Page 1 must carry the restored SQLite header.
	if !bytes.Equal(got[:SaltLength], sqliteHeader) {
		t.Fatalf("page 1 header = %q, want SQLite magic", got[:SaltLength])
	}
	ivOff := SQLCipherPageSize - SQLCipherReserve
	for pg, want := range [][]byte{page1, page2, page3} {
		off := pg * SQLCipherPageSize
		if !bytes.Equal(got[off:off+ivOff], want[:ivOff]) {
			t.Fatalf("page %d content mismatch", pg+1)
		}
	}
}

func TestDecryptDBRejectsWrongKey(t *testing.T) {
	encKey := make([]byte, EncKeyLength)
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(encKey); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}
	page1 := make([]byte, SQLCipherPageSize-SQLCipherReserve)
	copy(page1, sqliteHeader)

	dir := t.TempDir()
	encPath := buildEncryptedTestDB(t, dir, encKey, salt, [][]byte{page1})
	outPath := filepath.Join(dir, "plain.db")

	wrong := make([]byte, EncKeyLength)
	if _, err := rand.Read(wrong); err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptDB(encPath, outPath, wrong); err == nil {
		t.Fatal("DecryptDB accepted a wrong key (expected HMAC mismatch)")
	}
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		t.Fatal("partial output was published on failure")
	}
}

func TestDecryptDBRejectsTruncatedFile(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "short.db")
	if err := os.WriteFile(bad, make([]byte, 100), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := DecryptDB(bad, filepath.Join(dir, "out.db"), make([]byte, EncKeyLength)); err == nil {
		t.Fatal("DecryptDB accepted a non-page-multiple file")
	}
}
