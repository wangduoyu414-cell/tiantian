//go:build windows

package wxkey

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"weixin-key/internal/testutil"
)

func completeMaterialReport() PassiveProbeReport {
	return PassiveProbeReport{Status: "all-page1-hmac-verified", SourceDBs: 1, VerifiedDBs: 1,
		IdentityStable: true, AccountStable: true, SourceStable: true, ModuleStable: true}
}

func consumerSourceFixture(t *testing.T) (*VerifiedPassiveKeys, string) {
	t.Helper()
	base := filepath.Join(t.TempDir(), "db_storage")
	if err := os.Mkdir(base, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(base, "synthetic.db")
	if err := os.WriteFile(path, []byte("synthetic"), 0o600); err != nil {
		t.Fatal(err)
	}
	source, err := openProbeRoot(base, os.OpenRoot)
	if err != nil {
		t.Fatal(err)
	}
	keys := &VerifiedPassiveKeys{source: source, base: base, ready: true}
	keys.put("synthetic.db", bytes.Repeat([]byte{42}, 32))
	t.Cleanup(keys.clear)
	return keys, path
}

func TestVerifiedPassiveSourceRetainsOriginalRootIdentity(t *testing.T) {
	for _, kind := range []string{"replacement", "junction-to-original", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			keys, _ := consumerSourceFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if kind == "cancel" {
				cancel()
			} else {
				// The real retained os.Root prevents rename on this host
				// (separately tested below). Inject a different resolved
				// path to exercise the additional identity/reparse checks.
				original := keys.base
				changed := keys.base + "-changed"
				if kind == "replacement" {
					if err := os.Mkdir(changed, 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(changed, "synthetic.db"), []byte("synthetic"), 0o600); err != nil {
						t.Fatal(err)
					}
				} else {
					// Even a junction back to the original identity is not
					// the real directory contract admitted at observation.
					makeProbeTestJunction(t, changed, original)
				}
				keys.base = changed
			}
			err := keys.WithSource(ctx, "synthetic.db", func(string, []byte) error {
				t.Fatal("changed/redirected source reached consumer")
				return nil
			})
			if err == nil {
				t.Fatal("source boundary change accepted")
			}
			if kind == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation lost")
			}
		})
	}
}

func TestVerifiedPassiveSourceHoldsRootUntilConsumerCleanup(t *testing.T) {
	keys, _ := consumerSourceFixture(t)
	original, moved := keys.base, keys.base+"-moved"
	if err := os.Rename(original, moved); err == nil {
		t.Fatal("root not held between observation and consumer")
	}
	keys.clear()
	if err := os.Rename(original, moved); err != nil {
		t.Fatal("root was not released at material cleanup")
	}
}

func TestVerifiedPassiveSourceAllowsWritesButBlocksReplacement(t *testing.T) {
	keys, path := consumerSourceFixture(t)
	err := keys.WithSource(context.Background(), "synthetic.db", func(got string, b []byte) error {
		if got != path || !bytes.Equal(b, bytes.Repeat([]byte{42}, 32)) {
			t.Fatal("wrong source/key pairing")
		}
		writer, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			t.Fatal("consumer pin blocked ordinary writer")
		}
		writer.Close()
		if err := os.Rename(path, path+".moved"); err == nil {
			t.Fatal("consumer source replaced")
		}
		if err := os.Rename(keys.base, keys.base+".moved"); err == nil {
			t.Fatal("consumer root replaced")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path, path+".moved"); err != nil {
		t.Fatal("file pin not released")
	}
}
func TestVerifiedPassiveKeysDeliveredOnlyAfterStableFullValidation(t *testing.T) {
	path, pass, salt, enc := testutil.MustNewPassphraseDB(t.TempDir(), "db.db", 1)
	candidate, _ := hex.DecodeString(pass)
	want, _ := hex.DecodeString(enc)
	var keys VerifiedPassiveKeys
	var r PassiveProbeReport
	verify := newMaterialProbeVerifierRetaining(context.Background(),
		[]*sqlcipherVerifier{newSQLCipherVerifier(path, salt)}, 0, nil, &r, 8,
		func(_ int, b []byte) { keys.put(`message\db.db`, b) })
	done, err := verify(candidate)
	if !done || err != nil || r.DirectPassphraseDBs != 1 {
		t.Fatal("synthetic capture failed")
	}
	if err := keys.WithKey(`message\db.db`, func([]byte) error { t.Fatal("unsettled keys escaped"); return nil }); err == nil {
		t.Fatal("keys usable before post-verification")
	}
	var borrowed []byte
	sentinel := errors.New("synthetic consumer error")
	err = consumeVerifiedPassiveKeys(context.Background(), completeMaterialReport(), &keys, func(got *VerifiedPassiveKeys) error {
		if err := got.WithKey(`other.db`, func([]byte) error { t.Fatal("wrong DB fallback"); return nil }); err == nil {
			t.Fatal("unknown DB accepted")
		}
		if b, err := json.Marshal(got); err == nil || len(b) != 0 {
			t.Fatal("material container serialized")
		}
		for _, format := range []string{"%v", "%+v", "%#v", "%x"} {
			if fmt.Sprintf(format, got) != "[redacted verified passive keys]" {
				t.Fatal("material formatting leaked")
			}
			if fmt.Sprintf(format, *got) != "[redacted verified passive keys]" {
				t.Fatal("value-copy material formatting leaked")
			}
		}
		if b, err := json.Marshal(*got); err == nil || len(b) != 0 {
			t.Fatal("value-copy material container serialized")
		}
		return got.WithKey(`message\DB.db`, func(b []byte) error {
			borrowed = b
			if !bytes.Equal(b, want) || bytes.Equal(b, candidate) {
				t.Fatal("consumer got passphrase instead of derived key")
			}
			return sentinel
		})
	})
	if !errors.Is(err, sentinel) || keys.ready || len(keys.keys) != 0 {
		t.Fatal("consumer error or cleanup lost")
	}
	if !bytes.Equal(borrowed, make([]byte, 32)) {
		t.Fatal("borrowed key not cleared")
	}
}
func TestVerifiedPassiveKeysRejectPartialUnstableAndCancelled(t *testing.T) {
	for _, kind := range []string{"partial", "identity", "account", "source", "module", "missing-key", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			r := completeMaterialReport()
			var keys VerifiedPassiveKeys
			keys.put("db.db", bytes.Repeat([]byte{42}, 32))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "partial":
				r.VerifiedDBs = 0
			case "identity":
				r.IdentityStable = false
			case "account":
				r.AccountStable = false
			case "source":
				r.SourceStable = false
			case "module":
				r.ModuleStable = false
			case "missing-key":
				keys.clear()
			case "cancel":
				cancel()
			}
			err := consumeVerifiedPassiveKeys(ctx, r, &keys, func(*VerifiedPassiveKeys) error {
				t.Fatal("invalid capture reached consumer")
				return nil
			})
			if err == nil || keys.ready || len(keys.keys) != 0 {
				t.Fatal("invalid result accepted or retained")
			}
		})
	}
}
