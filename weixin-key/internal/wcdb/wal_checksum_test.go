package wcdb

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestWALRejectsModifiedCommitMarker(t *testing.T) {
	key, salt := make([]byte, EncKeyLength), make([]byte, SaltLength)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	page1 := make([]byte, SQLCipherPageSize-SQLCipherReserve)
	copy(page1, sqliteHeader)
	src := buildEncryptedTestDB(t, dir, key, salt, [][]byte{page1})
	wal := buildEncryptedWAL(t, dir, "source-wal", key, salt, []walFrame{{pgno: 2, plain: bytes.Repeat([]byte{0x44}, SQLCipherPageSize-SQLCipherReserve)}})
	b, err := os.ReadFile(wal)
	if err != nil {
		t.Fatal(err)
	}
	// Payload HMAC is still valid. Only the transaction's commit marker is
	// corrupted, which must never turn an uncommitted page into committed data.
	binary.BigEndian.PutUint32(b[32+4:32+8], 2)
	if err = os.WriteFile(wal, b, 0o600); err != nil {
		t.Fatal(err)
	}
	rep, err := DecryptDBWithWAL(src, wal, filepath.Join(dir, "out.db"), key)
	if err == nil && rep.WALFramesApplied != 0 {
		t.Fatal("unchecked frame header promoted uncommitted data")
	}
}
