//go:build windows

package wxkey

import (
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"weixin-key/internal/testutil"
)

// A structure-backed candidate that is the account PASSPHRASE resolves every
// unresolved salt via derivation, records the passphrase for config caching,
// and consumes budget.
func TestD0DualModeResolvesPassphraseCandidate(t *testing.T) {
	dir := t.TempDir()
	db1, pp, salt1, _ := testutil.MustNewPassphraseDB(dir, "msg_0.db", 1)
	db2, salt2, _ := testutil.MustNewPassphraseDBWith(dir, "msg_1.db", 1, pp)

	dbs := []windowsSourceDB{
		{rel: "msg_0.db", path: db1, salt: salt1},
		{rel: "msg_1.db", path: db2, salt: salt2},
	}
	salts := map[string]bool{salt1: true, salt2: true}
	scan := newSetupScan()
	budget := 1

	if !windowsD0DualModeCandidate(pp, dbs, salts, scan, time.Time{}, &budget) {
		t.Fatal("passphrase candidate did not resolve")
	}
	if budget != 0 {
		t.Fatalf("budget = %d, want 0", budget)
	}
	if scan.lenFound() != 2 {
		t.Fatalf("resolved salts = %d, want 2", scan.lenFound())
	}
	if scan.passphraseHex != pp {
		t.Fatal("passphrase not retained for config caching")
	}
}

// Budget exhaustion: with budget 0 nothing is attempted.
func TestD0DualModeBudgetExhausted(t *testing.T) {
	dir := t.TempDir()
	db1, pp, salt1, _ := testutil.MustNewPassphraseDB(dir, "msg_0.db", 1)
	dbs := []windowsSourceDB{{rel: "msg_0.db", path: db1, salt: salt1}}
	salts := map[string]bool{salt1: true}
	scan := newSetupScan()
	budget := 0
	if windowsD0DualModeCandidate(pp, dbs, salts, scan, time.Time{}, &budget) {
		t.Fatal("budget-0 attempt resolved")
	}
	if scan.lenFound() != 0 {
		t.Fatal("resolved with exhausted budget")
	}
}

// A raw enc_key candidate resolves without touching the passphrase path.
func TestD0DualModeRawKeyCandidate(t *testing.T) {
	dir := t.TempDir()
	db1, _, salt1, encKey := testutil.MustNewPassphraseDB(dir, "msg_0.db", 1)
	dbs := []windowsSourceDB{{rel: "msg_0.db", path: db1, salt: salt1}}
	salts := map[string]bool{salt1: true}
	scan := newSetupScan()
	budget := 4
	if !windowsD0DualModeCandidate(encKey, dbs, salts, scan, time.Time{}, &budget) {
		t.Fatal("rawkey candidate did not resolve")
	}
	if scan.found[salt1] != encKey {
		t.Fatalf("stored %q, want %q", scan.found[salt1], encKey)
	}
	if scan.passphraseHex != "" {
		t.Fatal("rawkey hit must not populate the passphrase slot")
	}
}

// A wrong candidate resolves nothing.
func TestD0DualModeWrongCandidate(t *testing.T) {
	dir := t.TempDir()
	db1, _, salt1, _ := testutil.MustNewPassphraseDB(dir, "msg_0.db", 1)
	dbs := []windowsSourceDB{{rel: "msg_0.db", path: db1, salt: salt1}}
	salts := map[string]bool{salt1: true}
	scan := newSetupScan()
	budget := 2
	wrong := strings.Repeat("ab", 32)
	if windowsD0DualModeCandidate(wrong, dbs, salts, scan, time.Time{}, &budget) {
		t.Fatal("wrong candidate resolved")
	}
	if budget != 1 {
		t.Fatalf("budget = %d, want 1 (one candidate attempted)", budget)
	}
	if scan.lenFound() != 0 {
		t.Fatal("wrong candidate recorded")
	}
}

func TestD0BudgetEnv(t *testing.T) {
	t.Setenv("WECHAT_CLI_D0_PASSPHRASE_BUDGET", "3")
	if n := windowsD0PassphraseBudget(); n != 3 {
		t.Fatalf("budget = %d, want 3", n)
	}
	t.Setenv("WECHAT_CLI_D0_PASSPHRASE_BUDGET", "")
	if n := windowsD0PassphraseBudget(); n != 8 {
		t.Fatalf("default budget = %d, want 8", n)
	}
}

var _ = hex.EncodeToString
