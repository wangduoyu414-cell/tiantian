//go:build windows

package wxkey

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/pbkdf2"
	"weixin-key/internal/wcdb"
)

// buildEncryptedDB builds a 2-page SQLCipher 4 file (WeChat layout) encrypted
// with encKey, for testing DecryptDatabase end to end.
func buildEncryptedDB(t *testing.T, dir string, encKey, salt []byte) (dbPath string, plaintext []byte) {
	t.Helper()
	block, err := aes.NewCipher(encKey)
	if err != nil {
		t.Fatal(err)
	}
	hmacSalt := make([]byte, wcdb.SaltLength)
	for i := range salt {
		hmacSalt[i] = salt[i] ^ wcdb.SQLCipherSaltMask
	}
	macKey := pbkdf2.Key(encKey, hmacSalt, wcdb.SQLCipherFastKDF, wcdb.SQLCipherMACKeyLen, sha512.New)
	ivOff := wcdb.SQLCipherPageSize - wcdb.SQLCipherReserve

	plaintext = make([]byte, 2*ivOff)
	for i := range plaintext {
		plaintext[i] = byte(i*13 + 7)
	}

	var buf bytes.Buffer
	for pgno := 1; pgno <= 2; pgno++ {
		page := make([]byte, wcdb.SQLCipherPageSize)
		iv := make([]byte, wcdb.SQLCipherIVSize)
		if _, err := rand.Read(iv); err != nil {
			t.Fatal(err)
		}
		start := 0
		if pgno == 1 {
			copy(page, salt)
			start = wcdb.SaltLength
		}
		src := plaintext[(pgno-1)*ivOff : pgno*ivOff]
		cipher.NewCBCEncrypter(block, iv).CryptBlocks(page[start:ivOff], src[start:ivOff])
		copy(page[ivOff:ivOff+wcdb.SQLCipherIVSize], iv)

		var pg [4]byte
		binary.LittleEndian.PutUint32(pg[:], uint32(pgno))
		mac := hmac.New(sha512.New, macKey)
		mac.Write(page[start : wcdb.SQLCipherPageSize-wcdb.SQLCipherHMACSize])
		mac.Write(pg[:])
		copy(page[wcdb.SQLCipherPageSize-wcdb.SQLCipherHMACSize:], mac.Sum(nil))
		buf.Write(page)
	}

	dbPath = filepath.Join(dir, "msg_0.db")
	if err := os.WriteFile(dbPath, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return dbPath, plaintext
}

func TestDecryptDatabaseEndToEnd(t *testing.T) {
	pass := make([]byte, wcdb.PassphraseLength)
	salt := make([]byte, wcdb.SaltLength)
	if _, err := rand.Read(pass); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}
	passHex := hex.EncodeToString(pass)
	encKeyHex, err := wcdb.DeriveEncKey(passHex, hex.EncodeToString(salt))
	if err != nil {
		t.Fatal(err)
	}
	encKey, _ := hex.DecodeString(encKeyHex)

	dir := t.TempDir()
	dbPath, plaintext := buildEncryptedDB(t, dir, encKey, salt)
	outPath := filepath.Join(dir, "plain.db")

	// Path 1: --enc-key
	res, err := DecryptDatabase(dbPath, outPath, DecryptOptions{EncKeyHex: encKeyHex})
	if err != nil {
		t.Fatalf("decrypt via --enc-key: %v", err)
	}
	if res.KeySource != "--enc-key" || res.Pages != 2 {
		t.Fatalf("unexpected result %+v", res)
	}
	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	ivOff := wcdb.SQLCipherPageSize - wcdb.SQLCipherReserve
	if !bytes.Equal(got[:wcdb.SaltLength], []byte("SQLite format 3\x00")) {
		t.Fatal("page 1 missing SQLite header after decrypt")
	}
	if !bytes.Equal(got[wcdb.SaltLength:ivOff], plaintext[wcdb.SaltLength:ivOff]) {
		t.Fatal("page 1 content mismatch")
	}
	if !bytes.Equal(got[wcdb.SQLCipherPageSize:wcdb.SQLCipherPageSize+ivOff], plaintext[ivOff:2*ivOff]) {
		t.Fatal("page 2 content mismatch")
	}

	// Path 2: --passphrase (derives the same enc_key from the file's salt)
	outPath2 := filepath.Join(dir, "plain2.db")
	res2, err := DecryptDatabase(dbPath, outPath2, DecryptOptions{PassphraseHex: passHex})
	if err != nil {
		t.Fatalf("decrypt via --passphrase: %v", err)
	}
	if res2.KeySource != "--passphrase" {
		t.Fatalf("source = %q", res2.KeySource)
	}

	// Path 3: wrong passphrase must fail before writing output
	bad := make([]byte, wcdb.PassphraseLength)
	if _, err := rand.Read(bad); err != nil {
		t.Fatal(err)
	}
	outPath3 := filepath.Join(dir, "plain3.db")
	if _, err := DecryptDatabase(dbPath, outPath3, DecryptOptions{PassphraseHex: hex.EncodeToString(bad)}); err == nil {
		t.Fatal("accepted wrong passphrase")
	}
	if _, err := os.Stat(outPath3); !os.IsNotExist(err) {
		t.Fatal("partial output written on key mismatch")
	}
}
