package wxkey

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"weixin-key/internal/wcdb"
)

// MaterialKind identifies what a verified candidate actually is, so a derived
// enc_key and a captured passphrase are never stored in each other's place.
type MaterialKind string

const (
	// MaterialEncKey is a raw 32-byte SQLCipher per-DB encryption key.
	MaterialEncKey MaterialKind = "enc_key"
	// MaterialPassphrase is the 32-byte account-level passphrase; the per-DB
	// enc_key must be derived with PBKDF2-HMAC-SHA512 against the DB salt.
	MaterialPassphrase MaterialKind = "passphrase"
)

// KeyCandidateVerification is the typed outcome of testing one candidate
// against one database.
type KeyCandidateVerification struct {
	Match   bool
	Kind    MaterialKind
	SaltHex string
	// EncKeyHex is the per-DB enc_key that must be persisted for SaltHex. When
	// Kind is MaterialPassphrase this is the DERIVED key, not the candidate.
	EncKeyHex string
	// PassphraseHex is set only when Kind is MaterialPassphrase: the verified
	// account passphrase itself (so it can be cached for future/new DBs).
	PassphraseHex string
	// Source identifies where the candidate came from (route label).
	Source string
}

// VerifyKeyCandidate tests candidateHex against the DB at dbPath, first as a
// raw enc_key (cheap page-1 HMAC), then as an account passphrase (PBKDF2
// derive + page-1 HMAC). The returned EncKeyHex is always the value that must
// be stored for the DB - never the untyped candidate.
//
// Verification is pure Go. The optional native WCDB open() was removed: it
// crashed (access violation) when the library was not bootstrapped, and the
// page-1 HMAC is the authoritative check anyway.
func VerifyKeyCandidate(dbPath, candidateHex, saltHex string) (KeyCandidateVerification, error) {
	v := KeyCandidateVerification{SaltHex: strings.ToLower(strings.TrimSpace(saltHex))}
	cand, err := normalizeKeyMaterialHex(candidateHex)
	if err != nil {
		// Not even shaped like 32-byte key material: no match, not an error.
		return v, nil
	}

	// 1. Raw enc_key: one page-1 HMAC, microseconds.
	ok, err := wcdb.VerifyEncKeyPage1(dbPath, cand)
	if err != nil {
		return v, err
	}
	if ok {
		v.Match = true
		v.Kind = MaterialEncKey
		v.EncKeyHex = cand
		return v, nil
	}

	// 2. Account passphrase: derive against this DB's salt, then verify.
	encKeyHex, err := wcdb.DeriveEncKey(cand, v.SaltHex)
	if err != nil {
		return v, fmt.Errorf("derive enc_key: %w", err)
	}
	ok, err = wcdb.VerifyEncKeyPage1(dbPath, encKeyHex)
	if err != nil {
		return v, err
	}
	if ok {
		v.Match = true
		v.Kind = MaterialPassphrase
		v.EncKeyHex = encKeyHex
		v.PassphraseHex = cand
	}
	return v, nil
}

// normalizeKeyMaterialHex lower-cases, strips an optional 0x prefix, and
// requires exactly 32 bytes of hex. Shared by enc_key and passphrase inputs.
func normalizeKeyMaterialHex(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(strings.ToLower(raw), "0x")
	b, err := hex.DecodeString(raw)
	if err != nil {
		return "", errors.New("not valid hex")
	}
	if len(b) != 32 {
		return "", fmt.Errorf("must be 32 bytes (64 hex chars), got %d bytes", len(b))
	}
	return hex.EncodeToString(b), nil
}

// normalizePassphraseHex validates and lower-cases a hex-encoded 32-byte
// passphrase. It accepts strings with or without an optional "0x" prefix.
func normalizePassphraseHex(raw string) (string, error) {
	v, err := normalizeKeyMaterialHex(raw)
	if err != nil {
		return "", fmt.Errorf("passphrase %s", err)
	}
	return v, nil
}

// normalizeEncKeyHex validates and lower-cases a hex-encoded 32-byte enc_key.
func normalizeEncKeyHex(raw string) (string, error) {
	v, err := normalizeKeyMaterialHex(raw)
	if err != nil {
		return "", fmt.Errorf("enc_key %s", err)
	}
	return v, nil
}
