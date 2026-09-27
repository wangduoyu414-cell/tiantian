package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"weixin-key/internal/config"
	"weixin-key/internal/pathguard"
)

type doctorCheck struct {
	Status string `json:"status"`
	Path   string `json:"path,omitempty"`
}

type doctorReport struct {
	SchemaVersion   int         `json:"schema_version"`
	OS              string      `json:"os"`
	Arch            string      `json:"arch"`
	ReadOnly        bool        `json:"read_only"`
	Source          doctorCheck `json:"source"`
	Config          doctorCheck `json:"config"`
	Output          doctorCheck `json:"output"`
	CacheVerified   bool        `json:"cache_verified"`
	ReadyForExport  bool        `json:"ready_for_export"`
	Acquisition     string      `json:"acquisition"`
	ExportReadiness string      `json:"export_readiness"`
	NextActions     []string    `json:"next_actions"`
}

// Doctor intentionally does not load/decrypt config, read a DB, enumerate
// accounts or open a process. An unselected path is not guessed from a prior
// machine or from WECHAT_CLI_* / WX_MCP_* environment overrides.
func cmdDoctor(args []string, stdout, stderr io.Writer) int {
	f, err := parseFlags("doctor", args, flagSpec{
		values: map[string]bool{"db-root": true, "config": true, "out": true},
		bools:  map[string]bool{"pretty": true},
	})
	if err != nil {
		return usageError("doctor", err, stderr)
	}
	if len(f.positionals) != 0 {
		return usageError("doctor", errors.New("no positional arguments accepted"), stderr)
	}
	for _, name := range []string{"db-root", "config", "out"} {
		if f.flag(name) && !filepath.IsAbs(f.str(name)) {
			return usageError("doctor", errors.New("selected paths must be absolute and nonempty"), stderr)
		}
	}
	rep := inspectOnboarding(f.str("db-root"), f.str("config"), f.str("out"))
	if err := printJSON(stdout, rep, f.flag("pretty")); err != nil {
		return fail("doctor", err, stderr)
	}
	if rep.Source.Status == "invalid" || rep.Config.Status == "invalid" || rep.Output.Status == "invalid" {
		return exitError
	}
	return exitOK // inspection completed, NOT permission/readiness to export
}

func inspectOnboarding(source, cfg, output string) doctorReport {
	rep := doctorReport{
		SchemaVersion: 1, OS: runtime.GOOS, Arch: runtime.GOARCH, ReadOnly: true,
		Source: doctorCheck{Status: "not-selected"}, Config: doctorCheck{Status: "not-selected"},
		Output:          doctorCheck{Status: "not-selected"},
		Acquisition:     "unsupported-platform",
		ExportReadiness: "real-parser-and-private-staging-not-yet-accepted",
		NextActions:     []string{"confirm-own-account-and-local-authorization"},
	}
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		rep.Acquisition = "bounded-passive-diagnostic-requires-local-validation"
	}
	if source != "" {
		rep.Source = doctorCheck{Status: "invalid", Path: filepath.Clean(source)}
		if resolved, err := filepath.EvalSymlinks(source); err == nil {
			if root, err := os.OpenRoot(resolved); err == nil {
				info, statErr := root.Lstat("db_storage")
				_ = root.Close()
				if statErr == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
					rep.Source = doctorCheck{Status: "selected-not-authenticated", Path: resolved}
				}
			}
		}
	}
	switch rep.Source.Status {
	case "not-selected":
		rep.NextActions = append(rep.NextActions, "select-explicit-account-root-containing-db_storage")
	case "invalid":
		rep.NextActions = append(rep.NextActions, "correct-source-path-without-scanning-unrelated-accounts")
	}
	if cfg != "" {
		rep.Config = doctorCheck{Status: "invalid", Path: filepath.Clean(cfg)}
		if info, err := os.Lstat(cfg); errors.Is(err, os.ErrNotExist) {
			rep.Config.Status = "absent"
		} else if err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0 {
			rep.Config.Status = "present-not-read-or-verified"
		}
		if rep.Source.Status == "selected-not-authenticated" {
			if _, err := pathguard.ResolveOutput(filepath.Dir(cfg), rep.Source.Path); err != nil {
				rep.Config.Status = "invalid"
			}
		}
	}
	if rep.Config.Status == "absent" {
		rep.NextActions = append(rep.NextActions, "init-account-at-new-explicit-path-then-use-reviewed-material-diagnostic")
	} else if rep.Config.Status == "present-not-read-or-verified" {
		rep.NextActions = append(rep.NextActions, "verify-selected-cache-before-any-acquisition")
	} else {
		rep.NextActions = append(rep.NextActions, "select-valid-local-config-path-never-overwrite-copied-cache")
	}
	if output != "" {
		rep.Output = doctorCheck{Status: "source-required", Path: filepath.Clean(output)}
		if rep.Source.Status == "selected-not-authenticated" {
			rep.Output.Status = "invalid"
			if resolved, err := pathguard.ResolveOutput(output, rep.Source.Path); err == nil {
				rep.Output = doctorCheck{Status: "path-only-checked", Path: resolved}
			}
		}
	}
	rep.NextActions = append(rep.NextActions,
		"check-private-staging-free-space-and-real-message-parser-before-export",
		"read-docs-AGENT_WORKFLOW.md-do-not-use-legacy-setup-as-fallback")
	return rep
}

func cmdInitAccount(args []string, stdout, stderr io.Writer) int {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals) // after worker cleanup AND proof output
	return cmdInitAccountWithSignals(args, stdout, stderr, signals, config.InitializeMetadataAtPath)
}

type initializeAccountCall func(context.Context, string, string, string) (config.MetadataInitProof, error)

func cmdInitAccountWithSignals(args []string, stdout, stderr io.Writer, signals <-chan os.Signal, initialize initializeAccountCall) int {
	f, err := parseFlags("init-account", args, flagSpec{
		values: map[string]bool{"db-root": true, "config": true, "account": true},
		bools:  map[string]bool{"pretty": true},
	})
	if err != nil {
		return usageError("init-account", err, stderr)
	}
	if len(f.positionals) != 0 || !filepath.IsAbs(f.str("db-root")) ||
		!filepath.IsAbs(f.str("config")) || f.str("account") == "" || strings.HasPrefix(f.str("account"), "-") {
		return usageError("init-account", errors.New("explicit absolute --db-root, --config and --account are required"), stderr)
	}
	proof, initErr := initializeAccountWithSignals(signals, f.str("config"), f.str("db-root"), f.str("account"), initialize)
	// A failed operation may already have committed. Always emit the proof,
	// and use the exit code rather than JSON presence as success evidence.
	if err := printJSON(stdout, proof, f.flag("pretty")); err != nil {
		return fail("init-account", err, stderr)
	}
	if initErr != nil {
		return fail("init-account", initErr, stderr)
	}
	return exitOK
}

func initializeAccountWithSignals(signals <-chan os.Signal, path, source, account string, initialize initializeAccountCall) (config.MetadataInitProof, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type outcome struct {
		proof config.MetadataInitProof
		err   error
	}
	done := make(chan outcome, 1)
	go func() {
		proof, err := initialize(ctx, path, source, account)
		done <- outcome{proof, err}
	}()
	for {
		select {
		case result := <-done:
			return result.proof, errors.Join(result.err, ctx.Err())
		case _, ok := <-signals:
			if !ok {
				signals = nil
				continue
			}
			cancel() // repeated interrupts still wait for this one worker
		}
	}
}
