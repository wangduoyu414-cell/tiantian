package wcdb

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"

	"golang.org/x/crypto/pbkdf2"
)

// SQLCipher 4 page layout used by WeChat 4.1+ local DBs.
const (
	SQLCipherPageSize  = 4096
	SQLCipherHMACSize  = 64
	SQLCipherReserve   = 80 // IV(16) + HMAC(64)
	SQLCipherIVSize    = 16 // per-page AES-CBC IV stored in the reserve area
	SQLCipherFastKDF   = 2
	SQLCipherSaltMask  = 0x3a
	SQLCipherMACKeyLen = 32 // critical: must be 32, not HMAC size 64
)

// VerifyEncKeyPage1 checks whether encKeyHex is the SQLCipher raw encryption key
// for dbPath by recomputing page-1 HMAC entirely in pure Go. This does not need
// the native WCDB/SQLCipher library and is the preferred offline verifier.
func VerifyEncKeyPage1(dbPath, encKeyHex string) (bool, error) {
	key, err := hex.DecodeString(encKeyHex)
	if err != nil {
		return false, fmt.Errorf("decode enc_key hex: %w", err)
	}
	if len(key) != EncKeyLength {
		return false, fmt.Errorf("enc_key must be %d bytes (got %d)", EncKeyLength, len(key))
	}

	page, err := readPage1(dbPath)
	if err != nil {
		return false, err
	}
	return page1MACMatches(key, page), nil
}

// VerifyPassphrasePage1 derives enc_key from passphraseHex + page-1 salt, then
// verifies the page-1 MAC. Returns the derived enc_key hex on success.
func VerifyPassphrasePage1(dbPath, passphraseHex string) (encKeyHex string, ok bool, err error) {
	page, err := readPage1(dbPath)
	if err != nil {
		return "", false, err
	}
	saltHex := hex.EncodeToString(page[:SaltLength])
	encKeyHex, err = DeriveEncKey(passphraseHex, saltHex)
	if err != nil {
		return "", false, err
	}
	key, err := hex.DecodeString(encKeyHex)
	if err != nil {
		return "", false, err
	}
	if !page1MACMatches(key, page) {
		return encKeyHex, false, nil
	}
	return encKeyHex, true, nil
}

func readPage1(dbPath string) ([]byte, error) {
	f, err := os.Open(dbPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	page := make([]byte, SQLCipherPageSize)
	n, err := f.Read(page)
	if err != nil {
		return nil, err
	}
	if n < SQLCipherPageSize {
		return nil, fmt.Errorf("%s: short page1 (%d bytes)", dbPath, n)
	}
	return page, nil
}

// page1MACMatches implements SQLCipher 4 page-1 HMAC verification:
//
//	mac_key = PBKDF2-HMAC-SHA512(enc_key, salt^0x3a, 2, dklen=32)
//	HMAC-SHA512(mac_key, page[16:4032] || LE32(1)) == page[4032:4096]
func page1MACMatches(encKey, page []byte) bool {
	if len(encKey) != EncKeyLength || len(page) < SQLCipherPageSize {
		return false
	}
	return pageMACValid(deriveMACKey(encKey, page[:SaltLength]), page, 1, SaltLength)
}

// deriveMACKey computes the SQLCipher 4 MAC key:
// PBKDF2-HMAC-SHA512(enc_key, salt^0x3a, fast_kdf_iter=2, dklen=32).
func deriveMACKey(encKey, salt []byte) []byte {
	hmacSalt := make([]byte, SaltLength)
	for i := range salt {
		hmacSalt[i] = salt[i] ^ SQLCipherSaltMask
	}
	return pbkdf2.Key(encKey, hmacSalt, SQLCipherFastKDF, SQLCipherMACKeyLen, sha512.New)
}

// pageMACValid verifies HMAC-SHA512(mac_key, page[start:4032] || LE32(pgno))
// against the stored MAC at page[4032:4096]. start is SaltLength for page 1
// (the salt bytes are not covered by the MAC) and 0 for every other page.
func pageMACValid(macKey, page []byte, pgno, start int) bool {
	regionEnd := SQLCipherPageSize - SQLCipherHMACSize
	var pg [4]byte
	binary.LittleEndian.PutUint32(pg[:], uint32(pgno))
	mac := hmac.New(sha512.New, macKey)
	mac.Write(page[start:regionEnd])
	mac.Write(pg[:])
	return hmac.Equal(mac.Sum(nil), page[regionEnd:SQLCipherPageSize])
}
