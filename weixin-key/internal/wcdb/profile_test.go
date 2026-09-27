package wcdb

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectProfileAcceptsEncryptedShape(t *testing.T) {
	dir := t.TempDir()
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
	path := buildEncryptedTestDB(t, dir, encKey, salt, [][]byte{page1, page1})
	p, err := DetectProfile(path)
	if err != nil {
		t.Fatalf("encrypted fixture rejected: %v", err)
	}
	if p.Name != "sqlcipher4-wechat4.1" || p.KDFIters != 256000 || p.PageSize != 4096 {
		t.Fatalf("profile = %+v", p)
	}
}

func TestDetectProfileRejectsPlaintext(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain.db")
	content := make([]byte, SQLCipherPageSize)
	copy(content, sqliteHeader) // plaintext SQLite magic
	if err := os.WriteFile(plain, content, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := DetectProfile(plain)
	if !errors.Is(err, ErrProfileUnknown) {
		t.Fatalf("err = %v, want ErrProfileUnknown", err)
	}
	if !strings.Contains(err.Error(), "plaintext") {
		t.Fatalf("error should say plaintext: %v", err)
	}
}

func TestDetectProfileRejectsOddSize(t *testing.T) {
	dir := t.TempDir()
	odd := filepath.Join(dir, "odd.db")
	if err := os.WriteFile(odd, make([]byte, 1000), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := DetectProfile(odd); !errors.Is(err, ErrProfileUnknown) {
		t.Fatalf("err = %v, want ErrProfileUnknown", err)
	}
}
