package wcdb

import (
	"encoding/hex"
	"testing"
)

func TestDeriveEncKey(t *testing.T) {
	passphrase := make([]byte, PassphraseLength)
	for i := range passphrase {
		passphrase[i] = byte(i)
	}
	salt := make([]byte, SaltLength)
	for i := range salt {
		salt[i] = byte(255 - i)
	}

	got, err := DeriveEncKey(hex.EncodeToString(passphrase), hex.EncodeToString(salt))
	if err != nil {
		t.Fatalf("DeriveEncKey failed: %v", err)
	}
	if len(got) != 64 {
		t.Fatalf("expected 64-hex enc key, got %d chars", len(got))
	}

	// Same inputs must produce the same output.
	got2, err := DeriveEncKey(hex.EncodeToString(passphrase), hex.EncodeToString(salt))
	if err != nil {
		t.Fatalf("DeriveEncKey second call failed: %v", err)
	}
	if got != got2 {
		t.Fatalf("DeriveEncKey is not deterministic: %s vs %s", got, got2)
	}
}

func TestDeriveEncKeyInvalidInputs(t *testing.T) {
	validPP := hex.EncodeToString(make([]byte, PassphraseLength))
	validSalt := hex.EncodeToString(make([]byte, SaltLength))

	cases := []struct {
		name string
		pp   string
		salt string
	}{
		{"short passphrase", "abcd", validSalt},
		{"long passphrase", validPP + "00", validSalt},
		{"short salt", validPP, "abcd"},
		{"long salt", validPP, validSalt + "00"},
		{"bad hex passphrase", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz", validSalt},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := DeriveEncKey(c.pp, c.salt); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
