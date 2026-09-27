//go:build windows

// Read-only toward WeChat. Explicit snapshot mode uses private ephemeral
// local plaintext. Only separate explicit cache mode writes protected config;
// no setup, material output or changes to WeChat.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"weixin-key/internal/wxkey"
)

type diagnosticOptions struct {
	inspect        bool
	snapshot       bool
	scratch        string
	cache          bool
	cacheConfig    string
	cacheConfigSHA string
	verifyCache    bool
	schemaCheck    bool
	semanticCheck  bool
	probe          wxkey.MaterialProbeOptions
}

func parseDiagnostic(args []string) (diagnosticOptions, error) {
	var out diagnosticOptions
	f := flag.NewFlagSet("material-probe-diagnostic", flag.ContinueOnError)
	f.SetOutput(io.Discard) // never echo unknown arguments or their values
	f.BoolVar(&out.inspect, "inspect", false, "inspect the pinned disk module only")
	f.BoolVar(&out.snapshot, "snapshot-check", false, "consume verified keys for one ephemeral full database check")
	f.StringVar(&out.scratch, "scratch", "", "explicit existing scratch directory outside the source")
	f.BoolVar(&out.cache, "cache-verified", false, "explicitly persist verified keys with current-user protection after full snapshot check")
	f.BoolVar(&out.verifyCache, "verify-cache", false, "verify protected cached keys without process access or writes")
	f.BoolVar(&out.schemaCheck, "schema-check", false, "report only bounded message column shapes during an offline private snapshot")
	f.BoolVar(&out.semanticCheck, "semantic-check", false, "report bounded message value-type and marker counts without message contents")
	f.StringVar(&out.cacheConfig, "cache-config", "", "explicit existing account configuration")
	f.StringVar(&out.cacheConfigSHA, "cache-config-sha256", "", "expected current configuration SHA256")
	f.StringVar(&out.probe.ModulePath, "module", "", "absolute pinned module path")
	f.StringVar(&out.probe.ModuleSHA256, "sha256", "", "expected full module SHA256")
	pid := f.Uint64("pid", 0, "observed target PID")
	created := f.String("created", "", "observed process creation time in RFC3339Nano")
	f.StringVar(&out.probe.DBRoot, "root", "", "explicit account root")
	f.StringVar(&out.probe.ExePath, "exe", "", "observed executable path")
	f.StringVar(&out.probe.PrimaryDB, "primary-db", "", "explicit db_storage-relative database")
	bad := errors.New("invalid diagnostic arguments")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return out, flag.ErrHelp
		}
		return out, bad
	}
	if f.NArg() != 0 {
		return out, bad
	}
	// Repeated flags are rejected instead of silently choosing a target.
	seen := map[string]bool{}
	for _, arg := range args {
		if len(arg) < 2 || arg[0] != '-' {
			continue
		}
		name := arg[1:]
		if len(name) > 0 && name[0] == '-' {
			name = name[1:]
		}
		for i, c := range name {
			if c == '=' {
				name = name[:i]
				break
			}
		}
		if seen[name] {
			return out, bad
		}
		seen[name] = true
	}
	if out.verifyCache {
		invalid := false
		f.Visit(func(v *flag.Flag) {
			switch v.Name {
			case "verify-cache", "root", "cache-config", "cache-config-sha256", "snapshot-check", "scratch", "primary-db", "schema-check", "semantic-check":
			default:
				invalid = true
			}
		})
		h, err := hex.DecodeString(out.cacheConfigSHA)
		if invalid || !filepath.IsAbs(out.probe.DBRoot) || !filepath.IsAbs(out.cacheConfig) || err != nil || len(h) != 32 {
			return out, bad
		}
		if out.snapshot {
			if !filepath.IsAbs(out.scratch) || !filepath.IsLocal(out.probe.PrimaryDB) || out.probe.PrimaryDB == "." {
				return out, bad
			}
		} else if out.scratch != "" || out.probe.PrimaryDB != "" || out.schemaCheck || out.semanticCheck {
			return out, bad
		}
		return out, nil
	}
	if out.schemaCheck || out.semanticCheck {
		return out, bad
	} // structural inspection is offline only
	hash, err := hex.DecodeString(out.probe.ModuleSHA256)
	if err != nil || len(hash) != 32 || !filepath.IsAbs(out.probe.ModulePath) {
		return out, bad
	}
	if out.inspect {
		liveFlag := false
		f.Visit(func(v *flag.Flag) {
			if v.Name != "inspect" && v.Name != "module" && v.Name != "sha256" {
				liveFlag = true
			}
		})
		if liveFlag {
			return out, bad
		}
		return out, nil
	}
	if out.snapshot != (out.scratch != "") || (out.snapshot && !filepath.IsAbs(out.scratch)) {
		return out, bad
	}
	if out.cache {
		h, err := hex.DecodeString(out.cacheConfigSHA)
		if !out.snapshot || !filepath.IsAbs(out.cacheConfig) || err != nil || len(h) != 32 {
			return out, bad
		}
	} else if out.cacheConfig != "" || out.cacheConfigSHA != "" {
		return out, bad
	}
	if *pid == 0 || *pid > 0xffffffff || !filepath.IsAbs(out.probe.DBRoot) || !filepath.IsAbs(out.probe.ExePath) ||
		!filepath.IsLocal(out.probe.PrimaryDB) || out.probe.PrimaryDB == "." {
		return out, bad
	}
	when, err := time.Parse(time.RFC3339Nano, *created)
	if err != nil || when.IsZero() {
		return out, bad
	}
	out.probe.PID, out.probe.CreatedUTC = uint32(*pid), when
	return out, nil
}

type diagnosticService struct {
	inspect      func(context.Context, string, string) (wxkey.MaterialProfileReport, error)
	probe        func(context.Context, wxkey.MaterialProbeOptions) (wxkey.PassiveProbeReport, error)
	consume      func(context.Context, wxkey.MaterialProbeOptions, func(*wxkey.VerifiedPassiveKeys) error) (wxkey.PassiveProbeReport, error)
	snapshot     func(context.Context, diagnosticOptions, string, []byte) (materialSnapshotReport, error)
	cache        func(context.Context, *wxkey.VerifiedPassiveKeys, string, string) (wxkey.MaterialCacheReport, error)
	verifyCache  func(context.Context, string, string, string) (wxkey.MaterialCacheVerification, error)
	consumeCache func(context.Context, string, string, string, func(*wxkey.VerifiedPassiveKeys) error) (wxkey.MaterialCacheVerification, error)
}

func runDiagnostic(ctx context.Context, args []string, stdout io.Writer, service diagnosticService) int {
	opts, err := parseDiagnostic(args)
	if errors.Is(err, flag.ErrHelp) {
		_, err = io.WriteString(stdout, "Disk only: --inspect --module ABS_PATH --sha256 HASH\nLive: --module ABS_PATH --sha256 HASH --pid PID --created RFC3339Nano --root ABS_ACCOUNT --exe ABS_EXE --primary-db RELATIVE_DB\nOptional: --snapshot-check --scratch ABS_EXISTING_DIRECTORY (private ephemeral plaintext, then removed; no chat output)\nExplicit persistence additionally requires: --cache-verified --cache-config ABS_EXISTING_CONFIG --cache-config-sha256 CURRENT_HASH (current-user DPAPI; protected backup; no key output)\nOffline only: --verify-cache --root ABS_ACCOUNT --cache-config ABS_CONFIG --cache-config-sha256 CURRENT_HASH (no process access or config writes)\nOptional offline snapshot: --snapshot-check --scratch ABS_EXISTING_DIRECTORY --primary-db RELATIVE_DB; add --schema-check for bounded message column shapes or --semantic-check for bounded value-type/marker counts, never row contents or table identities.\nDefault modes never write config. Scan deadline: cooperative 90s; snapshot/cache overall deadline: 8min.\n")
		if err != nil {
			return 1
		}
		return 0
	}
	out := struct {
		Static            *wxkey.MaterialProfileReport     `json:"static,omitempty"`
		Report            *wxkey.PassiveProbeReport        `json:"report,omitempty"`
		Snapshot          *materialSnapshotReport          `json:"snapshot,omitempty"`
		Cache             *wxkey.MaterialCacheReport       `json:"cache,omitempty"`
		CacheVerification *wxkey.MaterialCacheVerification `json:"cache_verification,omitempty"`
		ErrorCode         string                           `json:"error_code,omitempty"`
	}{}
	code := 0
	if err != nil {
		out.ErrorCode, code = "invalid-arguments", 2
	} else {
		if opts.verifyCache {
			var r wxkey.MaterialCacheVerification
			var e error
			if opts.snapshot {
				r, e = service.consumeCache(ctx, opts.probe.DBRoot, opts.cacheConfig, opts.cacheConfigSHA,
					func(keys *wxkey.VerifiedPassiveKeys) error {
						return keys.WithSource(ctx, opts.probe.PrimaryDB, func(source string, key []byte) error {
							result, err := service.snapshot(ctx, opts, source, key)
							out.Snapshot = &result
							return err
						})
					})
			} else {
				r, e = service.verifyCache(ctx, opts.probe.DBRoot, opts.cacheConfig, opts.cacheConfigSHA)
			}
			out.CacheVerification, err = &r, e
		} else if opts.inspect {
			r, e := service.inspect(ctx, opts.probe.ModulePath, opts.probe.ModuleSHA256)
			out.Static, err = &r, e
		} else if opts.snapshot {
			r, e := service.consume(ctx, opts.probe, func(keys *wxkey.VerifiedPassiveKeys) error {
				err := keys.WithSource(ctx, opts.probe.PrimaryDB, func(source string, key []byte) error {
					result, err := service.snapshot(ctx, opts, source, key)
					out.Snapshot = &result
					return err
				})
				if err != nil {
					return err
				}
				out.Cache, err = cacheAfterSnapshot(ctx, opts, keys, out.Snapshot, service.cache)
				return err
			})
			out.Report, err = &r, e
		} else {
			r, e := service.probe(ctx, opts.probe)
			out.Report, err = &r, e
		}
		if err != nil {
			code, out.ErrorCode = 1, "diagnostic-failed"
			switch {
			case errors.Is(err, wxkey.ErrPassiveProbeBudget):
				out.ErrorCode = "budget-exhausted"
			case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
				out.ErrorCode = "cancelled-or-timeout"
			}
		}
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if enc.Encode(out) != nil {
		return 1
	}
	return code
}

func cacheAfterSnapshot(ctx context.Context, opts diagnosticOptions, keys *wxkey.VerifiedPassiveKeys,
	snapshot *materialSnapshotReport,
	cache func(context.Context, *wxkey.VerifiedPassiveKeys, string, string) (wxkey.MaterialCacheReport, error),
) (*wxkey.MaterialCacheReport, error) {
	if !opts.cache {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if snapshot == nil || snapshot.Status != "full-database-verified-and-queryable" ||
		!snapshot.IntegrityOK || !snapshot.PrivateDirectoryVerified || !snapshot.EphemeralDirectoryRemoved {
		return nil, errors.New("full private snapshot verification and cleanup are required before persistence")
	}
	result, err := cache(ctx, keys, opts.cacheConfig, opts.cacheConfigSHA)
	return &result, err // retain Applied proof even when post-commit checks fail
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	duration := 90 * time.Second
	if opts, err := parseDiagnostic(os.Args[1:]); err == nil && opts.snapshot {
		duration = 8 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	os.Exit(runDiagnostic(ctx, os.Args[1:], os.Stdout,
		diagnosticService{inspect: wxkey.InspectPassiveMaterialProfile, probe: wxkey.ProbePassiveMaterialObjects,
			consume: wxkey.WithVerifiedPassiveKeys, snapshot: checkMaterialSnapshot, cache: wxkey.CacheVerifiedPassiveKeys,
			verifyCache: wxkey.VerifyProtectedMaterialCache, consumeCache: wxkey.WithVerifiedProtectedCache}))
}
