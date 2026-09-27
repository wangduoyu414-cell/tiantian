//go:build windows

package config

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func checkedConfigFixture(t *testing.T) (string, []byte, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	raw := []byte(`{"schema_version":4,"wxid":"wxid_synthetic","db_root":"C:\\synthetic\\account"}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(raw)
	return path, raw, hex.EncodeToString(h[:])
}
func addCheckedSyntheticKey(cfg *Config) error {
	cfg.KDF = "pbkdf2-sha512-256000"
	cfg.SetVerifiedKey(strings.Repeat("12", 16), strings.Repeat("ab", 32), "synthetic", 123)
	return nil
}
func TestProtectedExplicitUpdateIgnoresEnvAndReadbacks(t *testing.T) {
	path, before, hash := checkedConfigFixture(t)
	poison := filepath.Join(t.TempDir(), "must-not-write.json")
	t.Setenv("WECHAT_CLI_CONFIG", poison)
	t.Setenv("WECHAT_CLI_DB_ROOT", `C:\wrong-account`)
	proof, err := UpdateProtectedAtPath(context.Background(), path, hash, addCheckedSyntheticKey)
	if err != nil || !proof.Applied || !proof.ProtectedReadback || proof.SHA256 == hash {
		t.Fatal("protected update failed", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(after, []byte(strings.Repeat("ab", 32))) {
		t.Fatal("material persisted in plaintext")
	}
	cfg, err := decodeConfig(after)
	if err != nil || cfg.DBRoot != `C:\synthetic\account` || len(cfg.Keys) != 1 {
		t.Fatal("wrong account or missing protected key")
	}
	if _, err := os.Stat(poison); !os.IsNotExist(err) {
		t.Fatal("environment-selected path was written")
	}
	backup, err := os.ReadFile(path + ".schema4-backup.protected")
	if err != nil {
		t.Fatal(err)
	}
	var pv protectedValue
	if err := json.Unmarshal(backup, &pv); err != nil {
		t.Fatal(err)
	}
	restored, err := unprotectConfigBytes(&pv)
	if err != nil || !bytes.Equal(restored, before) {
		t.Fatal("original metadata backup not preserved")
	}
	clear(restored)
}
func TestProtectedExplicitUpdateRejectsMismatchCancelAndAccountSwitch(t *testing.T) {
	for _, kind := range []string{"hash", "cancel-before", "cancel-during", "account", "callback-error", "external-edit"} {
		t.Run(kind, func(t *testing.T) {
			path, before, hash := checkedConfigFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if kind == "hash" {
				hash = strings.Repeat("00", 32)
			}
			if kind == "cancel-before" {
				cancel()
			}
			sentinel := errors.New("synthetic callback failure")
			external := []byte(`{"schema_version":4,"wxid":"external","db_root":"C:\\synthetic\\account"}`)
			proof, err := UpdateProtectedAtPath(ctx, path, hash, func(c *Config) error {
				if kind == "hash" || kind == "cancel-before" {
					t.Fatal("invalid preflight reached mutation")
				}
				_ = addCheckedSyntheticKey(c)
				switch kind {
				case "cancel-during":
					cancel()
				case "account":
					c.Wxid = "different"
				case "callback-error":
					return sentinel
				case "external-edit":
					return os.WriteFile(path, external, 0o600)
				}
				return nil
			})
			if err == nil || proof.Applied {
				t.Fatal("unsafe config update accepted")
			}
			after, readErr := os.ReadFile(path)
			want := before
			if kind == "external-edit" {
				want = external
			}
			if readErr != nil || !bytes.Equal(want, after) {
				t.Fatal("failed update changed existing config")
			}
		})
	}
}
func TestProtectedExplicitUpdateLockIsFailFast(t *testing.T) {
	path, before, hash := checkedConfigFixture(t)
	f, err := os.OpenFile(filepath.Join(filepath.Dir(path), ".config.json.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	unlock, err := lockConfigFile(f)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	proof, err := UpdateProtectedAtPath(context.Background(), path, hash, func(*Config) error { t.Fatal("busy lock reached mutation"); return nil })
	if !errors.Is(err, ErrConfigBusy) || proof.Applied {
		t.Fatal("busy lock did not fail fast")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("busy update changed config")
	}
}

func TestProtectedExplicitUpdateReusesMetadataBackup(t *testing.T) {
	path, _, hash := checkedConfigFixture(t)
	first, err := UpdateProtectedAtPath(context.Background(), path, hash, addCheckedSyntheticKey)
	if err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(path + ".schema4-backup.protected")
	if err != nil {
		t.Fatal(err)
	}
	second, err := UpdateProtectedAtPath(context.Background(), path, first.SHA256, addCheckedSyntheticKey)
	if err != nil || !second.Applied || !second.BackupVerified {
		t.Fatal("a valid metadata backup must not block subsequent cache updates", err)
	}
	after, _ := os.ReadFile(path + ".schema4-backup.protected")
	if !bytes.Equal(backup, after) {
		t.Fatal("existing recovery backup was replaced")
	}
}

func TestProtectedExplicitUpdateReportsAppliedWhenReadbackFails(t *testing.T) {
	path, before, hash := checkedConfigFixture(t)
	sentinel := errors.New("synthetic post-commit verification error")
	proof, err := updateProtectedAtPath(context.Background(), path, hash, addCheckedSyntheticKey,
		func(root *os.Root, base string, cfg *Config, p *ProtectedUpdateProof) error {
			if !p.Applied {
				t.Fatal("commit must be recorded before readback")
			}
			return sentinel
		})
	if !errors.Is(err, sentinel) || !proof.Applied || proof.ProtectedReadback {
		t.Fatal("post-commit failure lost applied/uncertain status")
	}
	after, _ := os.ReadFile(path)
	if bytes.Equal(after, before) {
		t.Fatal("test did not actually commit before readback failure")
	}
	// Reusing the original digest must fail, not silently overwrite the commit.
	retry, err := UpdateProtectedAtPath(context.Background(), path, hash, addCheckedSyntheticKey)
	if !errors.Is(err, ErrConfigChanged) || retry.Applied {
		t.Fatal("stale retry was not rejected")
	}
}

func TestProtectedExplicitUpdateKeepsCancellationAfterCommit(t *testing.T) {
	for _, failReadback := range []bool{false, true} {
		t.Run(fmt.Sprint(failReadback), func(t *testing.T) {
			path, _, hash := checkedConfigFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			sentinel := errors.New("synthetic readback failure")
			proof, err := updateProtectedAtPath(ctx, path, hash, addCheckedSyntheticKey,
				func(root *os.Root, base string, cfg *Config, p *ProtectedUpdateProof) error {
					cancel() // interruption after the atomic commit
					if failReadback {
						return sentinel
					}
					return verifyProtectedReadback(root, base, cfg, p)
				})
			if !errors.Is(err, context.Canceled) || !proof.Applied {
				t.Fatal("post-commit cancellation was swallowed or applied status lost")
			}
			if failReadback {
				if !errors.Is(err, sentinel) || proof.ProtectedReadback {
					t.Fatal("readback failure lost")
				}
			} else if !proof.ProtectedReadback {
				t.Fatal("cancellation must not skip committed-state reconciliation")
			}
		})
	}
}
