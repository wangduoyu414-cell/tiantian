package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"weixin-key/internal/config"
)

func onboardingSource(t *testing.T) string {
	t.Helper()
	source := filepath.Join(t.TempDir(), "wxid_fixture_0001")
	if err := os.MkdirAll(filepath.Join(source, "db_storage"), 0o700); err != nil {
		t.Fatal(err)
	}
	return source
}

func TestDoctorMissingSelectionNeverClaimsReady(t *testing.T) {
	for _, name := range []string{"WECHAT_CLI_CONFIG", "WX_MCP_CONFIG", "WECHAT_CLI_DB_ROOT", "WX_MCP_DB_ROOT"} {
		t.Setenv(name, "ENVIRONMENT_VALUE_MUST_NOT_APPEAR")
	}
	code, stdout, stderr := runCmd(t, "doctor", "--pretty")
	var rep doctorReport
	decodeSingleJSON(t, stdout, &rep)
	if code != exitOK || stderr != "" || !rep.ReadOnly || rep.ReadyForExport || rep.CacheVerified ||
		rep.Source.Status != "not-selected" || rep.Config.Status != "not-selected" ||
		strings.Contains(stdout, "ENVIRONMENT_VALUE_MUST_NOT_APPEAR") {
		t.Fatal("doctor incorrectly inherited identity or claimed readiness")
	}
}

func TestDoctorFreshMachineDoesNotCreateFiles(t *testing.T) {
	source, work := onboardingSource(t), t.TempDir()
	cfg, out := filepath.Join(work, "config.json"), filepath.Join(work, "output")
	code, stdout, stderr := runCmd(t, "doctor", "--db-root", source, "--config", cfg, "--out", out)
	var rep doctorReport
	decodeSingleJSON(t, stdout, &rep)
	if code != exitOK || stderr != "" || rep.Config.Status != "absent" ||
		rep.Source.Status != "selected-not-authenticated" || rep.Output.Status != "path-only-checked" {
		t.Fatal("fresh-machine diagnosis wrong", stdout, stderr)
	}
	if entries, err := os.ReadDir(work); err != nil || len(entries) != 0 {
		t.Fatal("doctor wrote output/config/lock files")
	}
	if entries, err := os.ReadDir(filepath.Join(source, "db_storage")); err != nil || len(entries) != 0 {
		t.Fatal("doctor wrote into the source")
	}
}

func TestDoctorExistingConfigNotParsedOrDisplayed(t *testing.T) {
	source := onboardingSource(t)
	cfg := filepath.Join(t.TempDir(), "config.json")
	secret := []byte("not JSON; synthetic secret must never be printed")
	if err := os.WriteFile(cfg, secret, 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ := runCmd(t, "doctor", "--db-root", source, "--config", cfg)
	var rep doctorReport
	decodeSingleJSON(t, stdout, &rep)
	if code != exitOK || rep.Config.Status != "present-not-read-or-verified" || strings.Contains(stdout, string(secret)) {
		t.Fatal("config existence was confused with valid material")
	}
}

func TestDoctorRejectsUnsafeOrInvalidSelectedPaths(t *testing.T) {
	source := onboardingSource(t)
	for _, args := range [][]string{
		{"doctor", "--db-root", source, "--out", filepath.Join(source, "export")},
		{"doctor", "--db-root", source, "--config", filepath.Join(source, "config.json")},
		{"doctor", "--db-root", t.TempDir()},
	} {
		if code, _, _ := runCmd(t, args...); code != exitError {
			t.Fatal("invalid selected path not reported", args)
		}
	}
}

func TestOnboardingStrictArguments(t *testing.T) {
	for _, args := range [][]string{
		{"doctor", "--config"}, {"doctor", "--config", ""}, {"doctor", "--out", "relative"},
		{"doctor", "--unknown"}, {"doctor", "extra"}, {"init-account"},
		{"init-account", "--config", "relative", "--db-root", t.TempDir(), "--account", "fixture"},
		{"init-account", "--db-root", t.TempDir(), "--config", filepath.Join(t.TempDir(), "config.json"), "--account", ""},
	} {
		if code, _, _ := runCmd(t, args...); code != exitUsage {
			t.Fatal("malformed onboarding command accepted", args)
		}
	}
}

func TestInitAccountMissingValueDoesNotWriteOptionAsAccount(t *testing.T) {
	source, state := onboardingSource(t), t.TempDir()
	code, _, _ := runCmd(t, "init-account", "--db-root", source,
		"--config", filepath.Join(state, "config.json"), "--account", "--pretty")
	if code != exitUsage {
		t.Errorf("missing account value: got exit %d, want usage error", code)
	}
	if entries, err := os.ReadDir(state); err != nil || len(entries) != 0 {
		t.Error("missing value created configuration artifacts")
	}
}

func TestInitAccountCommandInterruptWaitsForCleanupAndEmitsProof(t *testing.T) {
	for _, applied := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-publication", true: "after-publication"}[applied], func(t *testing.T) {
			signals := make(chan os.Signal, 2)
			signals <- os.Interrupt
			signals <- os.Interrupt
			var out, stderr bytes.Buffer
			cleaned := false
			code := cmdInitAccountWithSignals([]string{
				"--db-root", onboardingSource(t), "--config", filepath.Join(t.TempDir(), "config.json"), "--account", "fixture",
			}, &out, &stderr, signals, func(ctx context.Context, _, _, _ string) (config.MetadataInitProof, error) {
				<-ctx.Done()
				cleaned = true
				return config.MetadataInitProof{Applied: applied, MetadataVerified: applied, PrivateFile: applied}, ctx.Err()
			})
			var proof config.MetadataInitProof
			decodeSingleJSON(t, out.String(), &proof)
			if code != exitError || !cleaned || proof.Applied != applied || stderr.Len() == 0 {
				t.Fatal("CLI interrupt lost cleanup/proof")
			}
		})
	}
}
