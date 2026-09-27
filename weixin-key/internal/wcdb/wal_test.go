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

// walFrame describes one synthetic WAL frame for buildEncryptedWAL.
type walFrame struct {
	pgno    uint32
	plain   []byte
	commit  bool
	dbSize  uint32
	badMAC  bool
	oldSalt bool
}

// buildEncryptedWAL writes a synthetic SQLCipher-encrypted WAL whose frames
// carry valid per-page HMACs (unless badMAC). It is the local, cycle-free
// equivalent of a test fixture builder (testutil imports wcdb, so wcdb tests
// cannot use testutil).
func buildEncryptedWAL(t *testing.T, dir, name string, encKey, salt []byte, frames []walFrame) string {
	t.Helper()
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

	var buf bytes.Buffer
	hdr := make([]byte, 32)
	binary.BigEndian.PutUint32(hdr[0:4], 0x377f0682)
	binary.BigEndian.PutUint32(hdr[4:8], 3007000)
	binary.BigEndian.PutUint32(hdr[8:12], 4096)
	binary.BigEndian.PutUint32(hdr[16:20], 0xA5A5A5A5)
	binary.BigEndian.PutUint32(hdr[20:24], 0x5A5A5A5A)
	s0, s1 := fixtureWALChecksum(hdr[:24], 0, 0)
	binary.BigEndian.PutUint32(hdr[24:28], s0)
	binary.BigEndian.PutUint32(hdr[28:32], s1)
	buf.Write(hdr)

	for _, fr := range frames {
		fh := make([]byte, 24)
		binary.BigEndian.PutUint32(fh[0:4], fr.pgno)
		if fr.commit {
			binary.BigEndian.PutUint32(fh[4:8], fr.dbSize)
		}
		salt1, salt2 := uint32(0xA5A5A5A5), uint32(0x5A5A5A5A)
		if fr.oldSalt {
			salt1, salt2 = 0x11111111, 0x22222222
		}
		binary.BigEndian.PutUint32(fh[8:12], salt1)
		binary.BigEndian.PutUint32(fh[12:16], salt2)

		page := make([]byte, SQLCipherPageSize)
		iv := make([]byte, SQLCipherIVSize)
		if _, err := rand.Read(iv); err != nil {
			t.Fatal(err)
		}
		plain := make([]byte, ivOff)
		copy(plain, fr.plain)
		start := 0
		if fr.pgno == 1 {
			copy(page, salt)
			start = SaltLength
		}
		cipher.NewCBCEncrypter(block, iv).CryptBlocks(page[start:ivOff], plain[start:ivOff])
		copy(page[ivOff:ivOff+SQLCipherIVSize], iv)

		var pg [4]byte
		binary.LittleEndian.PutUint32(pg[:], fr.pgno)
		mac := hmac.New(sha512.New, macKey)
		mac.Write(page[start : SQLCipherPageSize-SQLCipherHMACSize])
		mac.Write(pg[:])
		sum := mac.Sum(nil)
		if fr.badMAC {
			sum[0] ^= 0xFF
		}
		copy(page[SQLCipherPageSize-SQLCipherHMACSize:], sum)
		s0, s1 = fixtureWALChecksum(fh[:8], s0, s1)
		s0, s1 = fixtureWALChecksum(page, s0, s1)
		binary.BigEndian.PutUint32(fh[16:20], s0)
		binary.BigEndian.PutUint32(fh[20:24], s1)
		buf.Write(fh)
		buf.Write(page)
	}

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// Latest messages live only in the WAL: replaying committed frames must make
// them appear in the decrypted snapshot, while uncommitted tail frames are
// dropped.
func TestDecryptDBWithWALAppliesCommittedFrames(t *testing.T) {
	encKey := make([]byte, EncKeyLength)
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(encKey); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	page1 := make([]byte, SQLCipherPageSize-SQLCipherReserve)
	copy(page1, sqliteHeader)
	oldPage2 := bytes.Repeat([]byte{0xAA}, SQLCipherPageSize-SQLCipherReserve)
	encPath := buildEncryptedTestDB(t, dir, encKey, salt, [][]byte{page1, oldPage2})

	newPage2 := bytes.Repeat([]byte{0xBB}, SQLCipherPageSize-SQLCipherReserve)
	page3 := bytes.Repeat([]byte{0xCC}, SQLCipherPageSize-SQLCipherReserve)
	uncommitted := bytes.Repeat([]byte{0xEE}, SQLCipherPageSize-SQLCipherReserve)
	walPath := buildEncryptedWAL(t, dir, "encrypted.db-wal", encKey, salt, []walFrame{
		{pgno: 2, plain: newPage2},
		{pgno: 3, plain: page3, commit: true, dbSize: 3},
		{pgno: 2, plain: uncommitted}, // after the commit: must NOT apply
	})

	out := filepath.Join(dir, "plain.db")
	rep, err := DecryptDBWithWAL(encPath, walPath, out, encKey)
	if err != nil {
		t.Fatalf("DecryptDBWithWAL: %v", err)
	}
	if rep.MainPages != 2 || rep.WALFrames != 3 || rep.WALCommits != 1 || rep.WALFramesApplied != 2 {
		t.Fatalf("report = %+v", rep)
	}

	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3*SQLCipherPageSize {
		t.Fatalf("output size = %d, want 3 pages", len(got))
	}
	ivOff := SQLCipherPageSize - SQLCipherReserve
	if !bytes.Equal(got[SQLCipherPageSize:SQLCipherPageSize+ivOff], newPage2[:ivOff]) {
		t.Fatal("page 2 does not reflect the committed WAL frame")
	}
	if !bytes.Equal(got[2*SQLCipherPageSize:2*SQLCipherPageSize+ivOff], page3[:ivOff]) {
		t.Fatal("page 3 (only in WAL) missing from snapshot")
	}
	if bytes.Equal(got[SQLCipherPageSize:SQLCipherPageSize+ivOff], uncommitted[:ivOff]) {
		t.Fatal("uncommitted tail frame was applied")
	}
	if got[18] != 1 || got[19] != 1 {
		t.Fatalf("journal mode bytes = %d,%d, want 1,1 (rollback)", got[18], got[19])
	}
}

// A torn/corrupt frame stops the replay at the cut, keeping everything
// committed before it, and the report says so.
func TestDecryptDBWithWALStopsAtCorruptFrame(t *testing.T) {
	encKey := make([]byte, EncKeyLength)
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(encKey); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	page1 := make([]byte, SQLCipherPageSize-SQLCipherReserve)
	copy(page1, sqliteHeader)
	encPath := buildEncryptedTestDB(t, dir, encKey, salt, [][]byte{page1})

	good := bytes.Repeat([]byte{0x42}, SQLCipherPageSize-SQLCipherReserve)
	after := bytes.Repeat([]byte{0x43}, SQLCipherPageSize-SQLCipherReserve)
	walPath := buildEncryptedWAL(t, dir, "encrypted.db-wal", encKey, salt, []walFrame{
		{pgno: 2, plain: good, commit: true, dbSize: 2},
		{pgno: 3, plain: after, commit: true, dbSize: 3, badMAC: true},
	})

	out := filepath.Join(dir, "plain.db")
	rep, err := DecryptDBWithWAL(encPath, walPath, out, encKey)
	if err != nil {
		t.Fatalf("DecryptDBWithWAL: %v", err)
	}
	if rep.WALFramesApplied != 1 || rep.Note == "" {
		t.Fatalf("report = %+v, want 1 applied frame with a note", rep)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 2*SQLCipherPageSize {
		t.Fatalf("output size = %d, want 2 pages (corrupt commit ignored)", info.Size())
	}
}

// A WAL from a previous checkpoint generation (foreign salts) is ignored.
func TestDecryptDBWithWALRejectsForeignGeneration(t *testing.T) {
	encKey := make([]byte, EncKeyLength)
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(encKey); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	page1 := make([]byte, SQLCipherPageSize-SQLCipherReserve)
	copy(page1, sqliteHeader)
	encPath := buildEncryptedTestDB(t, dir, encKey, salt, [][]byte{page1})

	walPath := buildEncryptedWAL(t, dir, "encrypted.db-wal", encKey, salt, []walFrame{
		{pgno: 2, plain: bytes.Repeat([]byte{0x55}, SQLCipherPageSize-SQLCipherReserve), commit: true, dbSize: 2, oldSalt: true},
	})

	out := filepath.Join(dir, "plain.db")
	rep, err := DecryptDBWithWAL(encPath, walPath, out, encKey)
	if err != nil {
		t.Fatalf("DecryptDBWithWAL: %v", err)
	}
	if rep.WALFramesApplied != 0 || rep.Note == "" {
		t.Fatalf("report = %+v, want 0 applied with note", rep)
	}
}

// Fixture implementation deliberately does not call the production checksum.
func fixtureWALChecksum(b []byte, a, c uint32) (uint32, uint32) {
	words := make([]uint32, len(b)/4)
	for i := range words {
		words[i] = binary.LittleEndian.Uint32(b[i*4:])
	}
	for i := 0; i < len(words); i += 2 {
		a = a + words[i] + c
		c = c + words[i+1] + a
	}
	return a, c
}
