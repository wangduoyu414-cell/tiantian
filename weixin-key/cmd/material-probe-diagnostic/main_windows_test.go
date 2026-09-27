//go:build windows

package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"weixin-key/internal/wxkey"
)

func diagnosticTestArgs() []string {
	return []string{"--module", `C:\synthetic\Weixin.dll`, "--sha256", strings.Repeat("ab", 32)}
}
func TestDiagnosticExplicitArgumentsAndDispatch(t *testing.T) {
	args := append(diagnosticTestArgs(), "--pid", "123", "--created", "2026-09-12T12:00:00Z",
		"--root", `C:\synthetic\account`, "--exe", `C:\synthetic\Weixin.exe`, "--primary-db", `message\message_0.db`)
	opts, err := parseDiagnostic(args)
	if err != nil || opts.inspect || opts.probe.PID != 123 {
		t.Fatal("valid explicit live arguments rejected")
	}
	var out bytes.Buffer
	calls := 0
	code := runDiagnostic(context.Background(), args, &out, diagnosticService{
		probe: func(_ context.Context, p wxkey.MaterialProbeOptions) (wxkey.PassiveProbeReport, error) {
			calls++
			if p.PID != 123 || p.PrimaryDB != `message\message_0.db` {
				t.Fatal("wrong target")
			}
			return wxkey.PassiveProbeReport{Status: "no-match-in-observed-regions"}, nil
		},
	})
	if code != 0 || calls != 1 || !strings.Contains(out.String(), "no-match") {
		t.Fatal("live dispatch failed")
	}
	out.Reset()
	code = runDiagnostic(context.Background(), append(diagnosticTestArgs(), "--inspect"), &out, diagnosticService{
		inspect: func(context.Context, string, string) (wxkey.MaterialProfileReport, error) {
			return wxkey.MaterialProfileReport{Status: "static-hypothesis-only"}, nil
		},
	})
	if code != 0 || !strings.Contains(out.String(), "static-hypothesis-only") {
		t.Fatal("static dispatch failed")
	}
}
func TestDiagnosticRejectsAmbiguityBeforeIO(t *testing.T) {
	live := append(diagnosticTestArgs(), "--pid", "123", "--created", "2026-09-12T12:00:00Z",
		"--root", `C:\synthetic\account`, "--exe", `C:\synthetic\Weixin.exe`, "--primary-db", `message\message_0.db`)
	overflow := append(diagnosticTestArgs(), "--pid", "4294967296", "--created", "2026-09-12T12:00:00Z",
		"--root", `C:\synthetic\account`, "--exe", `C:\synthetic\Weixin.exe`, "--primary-db", `message\message_0.db`)
	for _, args := range [][]string{
		nil, append(diagnosticTestArgs(), "--inspect", "extra"), append(diagnosticTestArgs(), "--inspect", "--pid", "0"),
		append(diagnosticTestArgs(), "--inspect", "--unknown", "SECRET_SENTINEL"), overflow,
		append(append([]string(nil), live...), "--pid", "456"), append(append([]string(nil), live...), "--primary-db", `..\escape.db`),
		append(diagnosticTestArgs(), "--inspect", "--root"),
	} {
		var out bytes.Buffer
		// Nil services make any dispatch on invalid input a test failure.
		code := runDiagnostic(context.Background(), args, &out, diagnosticService{})
		if code != 2 || strings.Contains(out.String(), "SECRET_SENTINEL") {
			t.Fatal("unsafe arguments or output")
		}
	}
}
func TestDiagnosticErrorsDoNotReflectSensitiveDetails(t *testing.T) {
	var out bytes.Buffer
	code := runDiagnostic(context.Background(), append(diagnosticTestArgs(), "--inspect"), &out, diagnosticService{
		inspect: func(context.Context, string, string) (wxkey.MaterialProfileReport, error) {
			return wxkey.MaterialProfileReport{}, errors.New("SECRET_SENTINEL")
		},
	})
	if code != 1 || strings.Contains(out.String(), "SECRET_SENTINEL") {
		t.Fatal("internal error reflected")
	}
}

func TestDiagnosticSnapshotRequiresExplicitOptInAndScratch(t *testing.T) {
	base := append(diagnosticTestArgs(), "--pid", "123", "--created", "2026-09-12T12:00:00Z",
		"--root", `C:\synthetic\account`, "--exe", `C:\synthetic\Weixin.exe`, "--primary-db", `message\message_0.db`)
	valid := append(append([]string(nil), base...), "--snapshot-check", "--scratch", `C:\synthetic\scratch`)
	opts, err := parseDiagnostic(valid)
	if err != nil || !opts.snapshot || opts.scratch != `C:\synthetic\scratch` {
		t.Fatal("explicit snapshot mode rejected")
	}
	for _, extra := range [][]string{
		{"--snapshot-check"}, {"--scratch", `C:\synthetic\scratch`}, {"--snapshot-check", "--scratch", "relative"},
		{"--inspect", "--snapshot-check", "--scratch", `C:\synthetic\scratch`},
	} {
		if _, err := parseDiagnostic(append(append([]string(nil), base...), extra...)); err == nil {
			t.Fatal("ambiguous snapshot mode accepted")
		}
	}
	var out bytes.Buffer
	calls := 0
	code := runDiagnostic(context.Background(), valid, &out, diagnosticService{
		consume: func(context.Context, wxkey.MaterialProbeOptions, func(*wxkey.VerifiedPassiveKeys) error) (wxkey.PassiveProbeReport, error) {
			calls++
			return wxkey.PassiveProbeReport{Status: "failed"}, errors.New("SECRET_SENTINEL")
		},
	})
	if code != 1 || calls != 1 || strings.Contains(out.String(), "SECRET_SENTINEL") {
		t.Fatal("snapshot dispatch/error redaction failed")
	}
}

func TestDiagnosticCacheRequiresSeparateExplicitConsentAndBinding(t *testing.T) {
	base := append(diagnosticTestArgs(), "--pid", "123", "--created", "2026-09-12T12:00:00Z",
		"--root", `C:\synthetic\account`, "--exe", `C:\synthetic\Weixin.exe`, "--primary-db", "db.db",
		"--snapshot-check", "--scratch", `C:\synthetic\scratch`)
	valid := append(append([]string(nil), base...), "--cache-verified", "--cache-config", `C:\synthetic\config.json`,
		"--cache-config-sha256", strings.Repeat("12", 32))
	opts, err := parseDiagnostic(valid)
	if err != nil || !opts.cache {
		t.Fatal("explicit bound cache mode rejected")
	}
	for _, extra := range [][]string{
		{"--cache-verified"}, {"--cache-config", `C:\synthetic\config.json`},
		{"--cache-verified", "--cache-config", "relative", "--cache-config-sha256", strings.Repeat("12", 32)},
		{"--cache-verified", "--cache-config", `C:\synthetic\config.json`, "--cache-config-sha256", "bad"},
	} {
		if _, err := parseDiagnostic(append(append([]string(nil), base...), extra...)); err == nil {
			t.Fatal("ambiguous cache mode accepted")
		}
	}
}

func TestDiagnosticCacheOnlyAfterCompletePrivateSnapshot(t *testing.T) {
	for _, kind := range []string{"default", "nil", "status", "integrity", "privacy", "cleanup", "cancel", "success", "applied-error"} {
		t.Run(kind, func(t *testing.T) {
			opts := diagnosticOptions{cache: kind != "default", cacheConfig: `C:\synthetic\config.json`, cacheConfigSHA: "synthetic"}
			report := &materialSnapshotReport{Status: "full-database-verified-and-queryable", IntegrityOK: true,
				PrivateDirectoryVerified: true, EphemeralDirectoryRemoved: true}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "nil":
				report = nil
			case "status":
				report.Status = "failed"
			case "integrity":
				report.IntegrityOK = false
			case "privacy":
				report.PrivateDirectoryVerified = false
			case "cleanup":
				report.EphemeralDirectoryRemoved = false
			case "cancel":
				cancel()
			}
			calls := 0
			sentinel := errors.New("synthetic post-commit failure")
			out, err := cacheAfterSnapshot(ctx, opts, nil, report,
				func(_ context.Context, _ *wxkey.VerifiedPassiveKeys, path, hash string) (wxkey.MaterialCacheReport, error) {
					calls++
					if path != opts.cacheConfig || hash != opts.cacheConfigSHA {
						t.Fatal("explicit config binding was changed")
					}
					var result wxkey.MaterialCacheReport
					result.Update.Applied = true
					if kind == "applied-error" {
						return result, sentinel
					}
					return result, nil
				})
			switch kind {
			case "default":
				if calls != 0 || err != nil || out != nil {
					t.Fatal("default mode reached persistence")
				}
			case "success":
				if calls != 1 || err != nil || out == nil {
					t.Fatal("complete snapshot failed to reach persistence")
				}
			case "applied-error":
				if calls != 1 || !errors.Is(err, sentinel) || out == nil || !out.Update.Applied {
					t.Fatal("applied proof was lost")
				}
			default:
				if calls != 0 || err == nil || out != nil {
					t.Fatal("incomplete snapshot reached persistence")
				}
			}
		})
	}
}

func TestDiagnosticOfflineCacheHasNoLiveDispatch(t *testing.T) {
	args := []string{"--verify-cache", "--root", `C:\synthetic\account`,
		"--cache-config", `C:\synthetic\config.json`, "--cache-config-sha256", strings.Repeat("ab", 32)}
	var out bytes.Buffer
	calls := 0
	code := runDiagnostic(context.Background(), args, &out, diagnosticService{
		verifyCache: func(_ context.Context, root, path, hash string) (wxkey.MaterialCacheVerification, error) {
			calls++
			if root != args[2] || path != args[4] || hash != args[6] {
				t.Fatal("offline binding changed")
			}
			return wxkey.MaterialCacheVerification{Status: "all-source-dbs-verified-from-protected-cache"}, nil
		},
	}) // all live services nil: an accidental live call fails the test
	if code != 0 || calls != 1 {
		t.Fatal("offline dispatch failed")
	}
	for _, extra := range [][]string{{"--pid", "123"}, {"--inspect"}, {"--cache-verified"}, {"--snapshot-check"}, {"--module", `C:\x.dll`}} {
		if _, err := parseDiagnostic(append(append([]string(nil), args...), extra...)); err == nil {
			t.Fatal("offline mode accepted live/persistence arguments")
		}
	}
}

func TestDiagnosticOfflineSnapshotNeedsExplicitSourceAndScratch(t *testing.T) {
	base := []string{"--verify-cache", "--root", `C:\synthetic\account`,
		"--cache-config", `C:\synthetic\config.json`, "--cache-config-sha256", strings.Repeat("ab", 32)}
	valid := append(append([]string(nil), base...), "--snapshot-check", "--scratch", `C:\synthetic\scratch`, "--primary-db", "db.db", "--schema-check")
	opts, err := parseDiagnostic(valid)
	if err != nil || !opts.verifyCache || !opts.snapshot || !opts.schemaCheck {
		t.Fatal("explicit offline snapshot rejected")
	}
	for _, extra := range [][]string{
		{"--schema-check"}, {"--primary-db", "db.db"}, {"--scratch", `C:\synthetic\scratch`},
		{"--snapshot-check", "--scratch", `C:\synthetic\scratch`},
		{"--snapshot-check", "--scratch", `C:\synthetic\scratch`, "--primary-db", `..\escape.db`},
		{"--snapshot-check", "--scratch", "relative", "--primary-db", "db.db"},
	} {
		if _, err := parseDiagnostic(append(append([]string(nil), base...), extra...)); err == nil {
			t.Fatal("ambiguous offline snapshot accepted")
		}
	}
	var out bytes.Buffer
	calls := 0
	code := runDiagnostic(context.Background(), valid, &out, diagnosticService{
		consumeCache: func(context.Context, string, string, string, func(*wxkey.VerifiedPassiveKeys) error) (wxkey.MaterialCacheVerification, error) {
			calls++
			return wxkey.MaterialCacheVerification{Status: "failed"}, errors.New("SENSITIVE_SENTINEL")
		},
	})
	if code != 1 || calls != 1 || strings.Contains(out.String(), "SENSITIVE_SENTINEL") {
		t.Fatal("offline snapshot dispatch/redaction failed")
	}
}
