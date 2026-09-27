package wcdb

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// DefaultKDF identifies the SQLCipher 4 key-derivation method used by
	// WeChat 4.1+ when a passphrase is supplied instead of a raw key.
	DefaultKDF = "pbkdf2-sha512-256000"

	// DefaultKDFIters is SQLCipher 4's default PBKDF2 iteration count.
	DefaultKDFIters = 256000

	// PassphraseLength is the length of the raw passphrase material captured
	// from WeChat 4.1+ (32 bytes, used as the PBKDF2 password).
	PassphraseLength = 32

	// SaltLength is the SQLCipher salt size stored in the first bytes of each
	// encrypted .db file.
	SaltLength = 16

	// EncKeyLength is the SQLCipher AES-256 encryption key length.
	EncKeyLength = 32
)

// DeriveEncKey derives the SQLCipher 4 raw encryption key from a 32-byte
// passphrase and a 16-byte database salt. passphraseHex and saltHex must be
// lower/upper-case hexadecimal strings of the required lengths.
func DeriveEncKey(passphraseHex, saltHex string) (string, error) {
	passphrase, err := hex.DecodeString(passphraseHex)
	if err != nil {
		return "", fmt.Errorf("decode passphrase hex: %w", err)
	}
	if len(passphrase) != PassphraseLength {
		return "", fmt.Errorf("passphrase must be %d bytes (got %d)", PassphraseLength, len(passphrase))
	}

	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return "", fmt.Errorf("decode salt hex: %w", err)
	}
	if len(salt) != SaltLength {
		return "", fmt.Errorf("salt must be %d bytes (got %d)", SaltLength, len(salt))
	}

	key := pbkdf2.Key(passphrase, salt, DefaultKDFIters, EncKeyLength, sha512.New)
	return hex.EncodeToString(key), nil
}
