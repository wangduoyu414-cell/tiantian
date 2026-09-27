// Package testutil builds synthetic SQLCipher 4 databases in the WeChat 4.1+
// layout for tests. It contains no real data and no secrets: every byte is
// generated locally at test time.
package testutil

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/pbkdf2"

	"weixin-key/internal/wcdb"
)

// BuildEncryptedDB writes a synthetic SQLCipher 4 encrypted database with the
// given page count, encrypted with encKey under salt. Page payloads are
// deterministic pseudo-content; page 1 carries the salt in place of the
// SQLite header, exactly like a real SQLCipher file.
func BuildEncryptedDB(dir, name string, encKey, salt []byte, pages int) (string, error) {
	if pages < 1 {
		pages = 1
	}
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return "", err
	}
	hmacSalt := make([]byte, wcdb.SaltLength)
	for i := range salt {
		hmacSalt[i] = salt[i] ^ wcdb.SQLCipherSaltMask
	}
	macKey := pbkdf2.Key(encKey, hmacSalt, wcdb.SQLCipherFastKDF, wcdb.SQLCipherMACKeyLen, sha512.New)
	ivOff := wcdb.SQLCipherPageSize - wcdb.SQLCipherReserve

	var buf bytes.Buffer
	for pgno := 1; pgno <= pages; pgno++ {
		page := make([]byte, wcdb.SQLCipherPageSize)
		iv := make([]byte, wcdb.SQLCipherIVSize)
		if _, err := rand.Read(iv); err != nil {
			return "", err
		}
		start := 0
		if pgno == 1 {
			copy(page, salt)
			start = wcdb.SaltLength
		}
		plain := make([]byte, ivOff)
		for i := range plain {
			plain[i] = byte(pgno*31 + i*13 + 7)
		}
		cipher.NewCBCEncrypter(block, iv).CryptBlocks(page[start:ivOff], plain[start:ivOff])
		copy(page[ivOff:ivOff+wcdb.SQLCipherIVSize], iv)

		var pg [4]byte
		binary.LittleEndian.PutUint32(pg[:], uint32(pgno))
		mac := hmac.New(sha512.New, macKey)
		mac.Write(page[start : wcdb.SQLCipherPageSize-wcdb.SQLCipherHMACSize])
		mac.Write(pg[:])
		copy(page[wcdb.SQLCipherPageSize-wcdb.SQLCipherHMACSize:], mac.Sum(nil))
		buf.Write(page)
	}

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// NewPassphraseDB generates a random 32-byte passphrase and salt, derives the
// enc_key exactly like WeChat 4.1+ does, builds a 2-page encrypted DB, and
// returns everything a test needs to exercise passphrase/raw-key paths.
func NewPassphraseDB(dir, name string, pages int) (dbPath, passphraseHex, saltHex, encKeyHex string, err error) {
	pass := make([]byte, wcdb.PassphraseLength)
	salt := make([]byte, wcdb.SaltLength)
	if _, err = rand.Read(pass); err != nil {
		return "", "", "", "", err
	}
	if _, err = rand.Read(salt); err != nil {
		return "", "", "", "", err
	}
	passphraseHex = hex.EncodeToString(pass)
	saltHex = hex.EncodeToString(salt)
	encKeyHex, err = wcdb.DeriveEncKey(passphraseHex, saltHex)
	if err != nil {
		return "", "", "", "", err
	}
	encKey, err := hex.DecodeString(encKeyHex)
	if err != nil {
		return "", "", "", "", err
	}
	dbPath, err = BuildEncryptedDB(dir, name, encKey, salt, pages)
	if err != nil {
		return "", "", "", "", err
	}
	return dbPath, passphraseHex, saltHex, encKeyHex, nil
}

// MustNewPassphraseDB is NewPassphraseDB with fatal-on-error semantics for
// tests; it avoids importing testing in the builder itself.
func MustNewPassphraseDB(dir, name string, pages int) (dbPath, passphraseHex, saltHex, encKeyHex string) {
	dbPath, passphraseHex, saltHex, encKeyHex, err := NewPassphraseDB(dir, name, pages)
	if err != nil {
		panic(fmt.Sprintf("testutil.MustNewPassphraseDB: %v", err))
	}
	return dbPath, passphraseHex, saltHex, encKeyHex
}

// MustNewPassphraseDBWith builds another DB for the SAME account passphrase
// with a fresh random salt, yielding a different enc_key (like a real second
// WeChat DB). Returns the path, the salt, and the derived enc_key.
func MustNewPassphraseDBWith(dir, name string, pages int, passphraseHex string) (dbPath, saltHex, encKeyHex string) {
	salt := make([]byte, wcdb.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		panic(fmt.Sprintf("testutil.MustNewPassphraseDBWith: %v", err))
	}
	saltHex = fmt.Sprintf("%x", salt)
	encKeyHex, err := wcdb.DeriveEncKey(passphraseHex, saltHex)
	if err != nil {
		panic(fmt.Sprintf("testutil.MustNewPassphraseDBWith: %v", err))
	}
	encKey, err := hex.DecodeString(encKeyHex)
	if err != nil {
		panic(fmt.Sprintf("testutil.MustNewPassphraseDBWith: %v", err))
	}
	dbPath, err = BuildEncryptedDB(dir, name, encKey, salt, pages)
	if err != nil {
		panic(fmt.Sprintf("testutil.MustNewPassphraseDBWith: %v", err))
	}
	return dbPath, saltHex, encKeyHex
}

// WALFrame describes one synthetic WAL frame for BuildEncryptedWAL.
type WALFrame struct {
	Pgno    uint32
	Plain   []byte // plaintext content region (pageSize-reserve bytes)
	Commit  bool   // true = commit frame (carries DBSize)
	DBSize  uint32 // DB size in pages after commit (when Commit)
	BadMAC  bool   // test hook: corrupt this frame's HMAC
	OldSalt bool   // test hook: use foreign salts (previous checkpoint generation)
}

// BuildEncryptedWAL writes a synthetic SQLCipher-encrypted WAL file whose
// frames carry valid per-page HMACs (unless BadMAC is set). It exists so WAL
// replay can be tested without a live SQLite instance.
func BuildEncryptedWAL(dir, name string, encKey, salt []byte, frames []WALFrame) (string, error) {
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return "", err
	}
	hmacSalt := make([]byte, wcdb.SaltLength)
	for i := range salt {
		hmacSalt[i] = salt[i] ^ wcdb.SQLCipherSaltMask
	}
	macKey := pbkdf2.Key(encKey, hmacSalt, wcdb.SQLCipherFastKDF, wcdb.SQLCipherMACKeyLen, sha512.New)
	ivOff := wcdb.SQLCipherPageSize - wcdb.SQLCipherReserve

	var buf bytes.Buffer
	// WAL header, big-endian fields.
	hdr := make([]byte, 32)
	binary.BigEndian.PutUint32(hdr[0:4], 0x377f0682)   // magic
	binary.BigEndian.PutUint32(hdr[4:8], 3007000)      // version
	binary.BigEndian.PutUint32(hdr[8:12], 4096)        // page size
	binary.BigEndian.PutUint32(hdr[12:16], 0)          // checkpoint seq
	binary.BigEndian.PutUint32(hdr[16:20], 0xA5A5A5A5) // salt1
	binary.BigEndian.PutUint32(hdr[20:24], 0x5A5A5A5A) // salt2
	buf.Write(hdr)

	for _, fr := range frames {
		fh := make([]byte, 24)
		binary.BigEndian.PutUint32(fh[0:4], fr.Pgno)
		if fr.Commit {
			binary.BigEndian.PutUint32(fh[4:8], fr.DBSize)
		}
		s1, s2 := uint32(0xA5A5A5A5), uint32(0x5A5A5A5A)
		if fr.OldSalt {
			s1, s2 = 0x11111111, 0x22222222
		}
		binary.BigEndian.PutUint32(fh[8:12], s1)
		binary.BigEndian.PutUint32(fh[12:16], s2)
		buf.Write(fh)

		page := make([]byte, wcdb.SQLCipherPageSize)
		iv := make([]byte, wcdb.SQLCipherIVSize)
		if _, err := rand.Read(iv); err != nil {
			return "", err
		}
		plain := make([]byte, ivOff)
		copy(plain, fr.Plain)
		start := 0
		if fr.Pgno == 1 {
			copy(page, salt)
			start = wcdb.SaltLength
		}
		cipher.NewCBCEncrypter(block, iv).CryptBlocks(page[start:ivOff], plain[start:ivOff])
		copy(page[ivOff:ivOff+wcdb.SQLCipherIVSize], iv)

		var pg [4]byte
		binary.LittleEndian.PutUint32(pg[:], fr.Pgno)
		mac := hmac.New(sha512.New, macKey)
		mac.Write(page[start : wcdb.SQLCipherPageSize-wcdb.SQLCipherHMACSize])
		mac.Write(pg[:])
		sum := mac.Sum(nil)
		if fr.BadMAC {
			sum[0] ^= 0xFF
		}
		copy(page[wcdb.SQLCipherPageSize-wcdb.SQLCipherHMACSize:], sum)
		buf.Write(page)
	}

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return "", err
	}
	return path, nil
}
