//go:build windows

package wxkey

import (
	"encoding/hex"
	"testing"
	"time"

	"weixin-key/internal/testutil"
)

// The pure chunk scanner finds an 8-aligned key window.
func TestScanChunkForKeysFindsAlignedKey(t *testing.T) {
	dir := t.TempDir()
	dbPath, _, saltHex, encKeyHex := testutil.MustNewPassphraseDB(dir, "msg_0.db", 1)
	v := newSQLCipherVerifier(dbPath, saltHex)
	if v == nil {
		t.Fatal("verifier nil")
	}
	key, _ := hex.DecodeString(encKeyHex)

	buf := make([]byte, 4096)
	for i := range buf {
		buf[i] = byte(i * 31)
	}
	copy(buf[256:288], key) // 256 is 8-aligned

	var n uint64
	got := scanChunkForKeys(buf, []*sqlcipherVerifier{v}, &n)
	if got == nil || hex.EncodeToString(got) != encKeyHex {
		t.Fatal("aligned key not found")
	}
	if n == 0 {
		t.Fatal("candidate counter not incremented")
	}
}

// A key not on the 8-byte stride must NOT match (documented limitation: keys
// are pointer-aligned in WeChat's allocations).
func TestScanChunkForKeysSkipsUnaligned(t *testing.T) {
	dir := t.TempDir()
	dbPath, _, saltHex, encKeyHex := testutil.MustNewPassphraseDB(dir, "msg_0.db", 1)
	v := newSQLCipherVerifier(dbPath, saltHex)
	key, _ := hex.DecodeString(encKeyHex)

	buf := make([]byte, 4096)
	for i := range buf {
		buf[i] = byte(i * 31)
	}
	copy(buf[252:284], key) // 252 % 8 = 4: off-stride
	if got := scanChunkForKeys(buf, []*sqlcipherVerifier{v}, nil); got != nil {
		t.Fatal("off-stride key matched (document the stride change if intentional)")
	}
}

// The scan is a pure function of the input bytes: same buffer, same verdict,
// regardless of who calls it (worker-count independence).
func TestScanChunkForKeysDeterministic(t *testing.T) {
	dir := t.TempDir()
	dbPath, _, saltHex, encKeyHex := testutil.MustNewPassphraseDB(dir, "msg_0.db", 1)
	v := newSQLCipherVerifier(dbPath, saltHex)
	key, _ := hex.DecodeString(encKeyHex)
	buf := make([]byte, 8192)
	for i := range buf {
		buf[i] = byte(i * 17)
	}
	copy(buf[1024:1056], key)
	a := scanChunkForKeys(buf, []*sqlcipherVerifier{v}, nil)
	b := scanChunkForKeys(buf, []*sqlcipherVerifier{v}, nil)
	if hex.EncodeToString(a) != hex.EncodeToString(b) {
		t.Fatal("non-deterministic scan")
	}
}

func TestWindowsScanWorkerCount(t *testing.T) {
	t.Setenv("WECHAT_CLI_SCAN_WORKERS", "7")
	if n := windowsScanWorkerCount(); n != 7 {
		t.Fatalf("workers = %d, want 7", n)
	}
	t.Setenv("WECHAT_CLI_SCAN_WORKERS", "bogus")
	if n := windowsScanWorkerCount(); n < 1 || n > 64 {
		t.Fatalf("fallback workers = %d", n)
	}
}

func TestRouteSharesAndDeadlineCapping(t *testing.T) {
	if d := windowsRouteShare("d2-heap"); d != 60*time.Second {
		t.Fatalf("d2 share = %s", d)
	}
	t.Setenv("WECHAT_CLI_BUDGET_D2_HEAP", "5s")
	if d := windowsRouteShare("d2-heap"); d != 5*time.Second {
		t.Fatalf("env override share = %s", d)
	}
	global := time.Now().Add(10 * time.Second)
	dl := windowsRouteDeadline(global, time.Minute)
	if !dl.Equal(global) {
		t.Fatal("route deadline must cap at global deadline")
	}
	uncapped := windowsRouteDeadline(time.Time{}, time.Minute)
	if uncapped.IsZero() || time.Until(uncapped) < 59*time.Second || time.Until(uncapped) > time.Minute {
		t.Fatal("zero global deadline disables global capping, not the finite per-route share")
	}
}
