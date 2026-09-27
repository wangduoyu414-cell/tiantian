package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"weixin-key/internal/testutil"
)

// isolateConfig points the config at an empty temp file so tests never touch
// the real user config, and neutralizes the env passphrase override.
func isolateConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	t.Setenv("WECHAT_CLI_CONFIG", path)
	t.Setenv("WECHAT_CLI_PASSPHRASE_HEX", "")
	return path
}

// runCmd runs the CLI in-process and captures streams.
func runCmd(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// decodeSingleJSON asserts stdout holds exactly one JSON value.
func decodeSingleJSON(t *testing.T, stdout string, v any) {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(stdout))
	if err := dec.Decode(v); err != nil {
		t.Fatalf("stdout is not one JSON value: %v\nstdout: %s", err, stdout)
	}
	var extra any
	if err := dec.Decode(&extra); err == nil {
		t.Fatalf("stdout holds a second JSON value (double dispatch?): %s", stdout)
	}
}

func TestRunVerifyWithoutMaterialFails(t *testing.T) {
	isolateConfig(t)
	code, _, stderr := runCmd(t, "verify")
	if code != exitError {
		t.Fatalf("verify without material: exit = %d, want %d (silent success regression)", code, exitError)
	}
	if !strings.Contains(stderr, "passphrase") {
		t.Fatalf("stderr should name the missing material: %q", stderr)
	}
}

func TestRunVerifyRejectsUnknownFlagAndMissingValue(t *testing.T) {
	isolateConfig(t)
	if code, _, _ := runCmd(t, "verify", "--bogus"); code != exitUsage {
		t.Fatalf("unknown flag: exit = %d, want %d", code, exitUsage)
	}
	if code, _, _ := runCmd(t, "verify", "--db"); code != exitUsage {
		t.Fatalf("missing value: exit = %d, want %d", code, exitUsage)
	}
	if code, _, _ := runCmd(t, "decrypt", "--db", "x", "--out", "y", "--bogus"); code != exitUsage {
		t.Fatalf("decrypt unknown flag: exit = %d, want %d", code, exitUsage)
	}
	if code, _, _ := runCmd(t, "setup", "unexpected-positional"); code != exitUsage {
		t.Fatalf("setup positional: exit = %d, want %d", code, exitUsage)
	}
}

func TestRunUnknownCommandAndHelp(t *testing.T) {
	isolateConfig(t)
	if code, _, _ := runCmd(t, "frobnicate"); code != exitUsage {
		t.Fatalf("unknown command: exit = %d, want %d", code, exitUsage)
	}
	code, stdout, _ := runCmd(t, "--help")
	if code != exitOK || !strings.Contains(stdout, "Usage:") {
		t.Fatalf("help: exit = %d, stdout missing usage", code)
	}
	if code := func() int { c, _, _ := runCmd(t); return c }(); code != exitUsage {
		t.Fatalf("no args: exit = %d, want %d", code, exitUsage)
	}
}

// The dispatch regression: a successful decrypt must print exactly one result
// and exit 0, not chain into the verify command afterwards.
func TestRunDecryptSucceedsOnce(t *testing.T) {
	isolateConfig(t)
	dir := t.TempDir()
	dbPath, _, _, encKey := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)
	out := filepath.Join(dir, "plain.db")

	code, stdout, stderr := runCmd(t, "decrypt", "--db", dbPath, "--out", out, "--enc-key", encKey)
	if code != exitOK {
		t.Fatalf("decrypt: exit = %d, stderr = %s", code, stderr)
	}
	var res struct {
		OutPath   string `json:"out_path"`
		KeySource string `json:"key_source"`
		Verified  bool   `json:"verified"`
		Pages     int    `json:"pages"`
	}
	decodeSingleJSON(t, stdout, &res)
	if !res.Verified || res.KeySource != "--enc-key" || res.Pages != 2 {
		t.Fatalf("result = %+v", res)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("output missing: %v", err)
	}
}

// P1 acceptance: with a previously verified config cache, decrypt runs fully
// offline with NO flags beyond paths.
func TestRunDecryptUsesConfigCache(t *testing.T) {
	cfgPath := isolateConfig(t)
	dir := t.TempDir()
	dbPath, _, salt, encKey := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)
	if err := os.WriteFile(cfgPath, []byte(`{"schema_version":2,"keys":{"`+salt+`":"`+encKey+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "plain.db")

	code, stdout, stderr := runCmd(t, "decrypt", "--db", dbPath, "--out", out)
	if code != exitOK {
		t.Fatalf("decrypt via config cache: exit = %d, stderr = %s", code, stderr)
	}
	var res struct {
		KeySource string `json:"key_source"`
	}
	decodeSingleJSON(t, stdout, &res)
	if res.KeySource != "config:cache" {
		t.Fatalf("key_source = %q, want config:cache", res.KeySource)
	}
}

// --no-config must force a failure when only the config could have provided
// material.
func TestRunDecryptNoConfigSkipsCache(t *testing.T) {
	cfgPath := isolateConfig(t)
	dir := t.TempDir()
	dbPath, _, salt, encKey := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)
	if err := os.WriteFile(cfgPath, []byte(`{"schema_version":2,"keys":{"`+salt+`":"`+encKey+`"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "plain.db")

	code, _, stderr := runCmd(t, "decrypt", "--db", dbPath, "--out", out, "--no-config")
	if code != exitError {
		t.Fatalf("decrypt --no-config: exit = %d, want %d", code, exitError)
	}
	if !strings.Contains(stderr, "no key material") {
		t.Fatalf("stderr should report missing material: %q", stderr)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("output file must not exist after failed decrypt")
	}
}

func TestRunVerifyMatchesSyntheticDB(t *testing.T) {
	isolateConfig(t)
	dir := t.TempDir()
	dbPath, passHex, _, _ := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)

	code, stdout, _ := runCmd(t, "verify", "--passphrase", passHex, "--db", dbPath)
	if code != exitOK {
		t.Fatalf("verify correct passphrase: exit = %d", code)
	}
	var res struct {
		Material string `json:"material"`
		Matched  int    `json:"matched"`
		Total    int    `json:"total"`
	}
	decodeSingleJSON(t, stdout, &res)
	if res.Material != "passphrase" || res.Matched != 1 || res.Total != 1 {
		t.Fatalf("result = %+v", res)
	}
}

func TestRunVerifyEncKeyFlag(t *testing.T) {
	isolateConfig(t)
	dir := t.TempDir()
	dbPath, _, _, encKey := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)

	code, stdout, _ := runCmd(t, "verify", "--enc-key", encKey, "--db", dbPath)
	if code != exitOK {
		t.Fatalf("verify --enc-key: exit = %d", code)
	}
	var res struct {
		Material string `json:"material"`
		Matched  int    `json:"matched"`
	}
	decodeSingleJSON(t, stdout, &res)
	if res.Material != "enc_key" || res.Matched != 1 {
		t.Fatalf("result = %+v", res)
	}
}

func TestRunVerifyWrongMaterialExitsOne(t *testing.T) {
	isolateConfig(t)
	dir := t.TempDir()
	dbPath, _, _, _ := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)
	wrong := strings.Repeat("ef", 32)

	code, stdout, _ := runCmd(t, "verify", "--passphrase", wrong, "--db", dbPath)
	if code != exitError {
		t.Fatalf("verify wrong passphrase: exit = %d, want %d", code, exitError)
	}
	var res struct {
		Matched int `json:"matched"`
	}
	decodeSingleJSON(t, stdout, &res)
	if res.Matched != 0 {
		t.Fatalf("matched = %d, want 0", res.Matched)
	}
}

func TestRunVerifyConflictingMaterialIsUsageError(t *testing.T) {
	isolateConfig(t)
	code, _, _ := runCmd(t, "verify", "--passphrase", strings.Repeat("aa", 32), "--enc-key", strings.Repeat("bb", 32), "--db", "x")
	if code != exitUsage {
		t.Fatalf("conflicting material: exit = %d, want %d", code, exitUsage)
	}
}

// Wrong key must not leave a partial plaintext file behind.
func TestRunDecryptWrongKeyLeavesNoOutput(t *testing.T) {
	isolateConfig(t)
	dir := t.TempDir()
	dbPath, _, _, _ := testutil.MustNewPassphraseDB(dir, "msg_0.db", 2)
	out := filepath.Join(dir, "plain.db")
	wrong := strings.Repeat("99", 32)

	code, _, _ := runCmd(t, "decrypt", "--db", dbPath, "--out", out, "--enc-key", wrong)
	if code != exitError {
		t.Fatalf("decrypt wrong key: exit = %d, want %d", code, exitError)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("partial output must not exist after failed decrypt")
	}
}
