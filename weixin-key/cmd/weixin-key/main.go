// weixin-key extracts the SQLCipher key material that protects a logged-in
// Windows WeChat 4.1+ local database, so the database can be decrypted offline.
//
// Only operate on your own account on your own machine.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"weixin-key/internal/export"
	"weixin-key/internal/wxkey"
)

const appName = "weixin-key"

// version is injected at build time: -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

// Exit-code contract:
//
//	0 - success
//	1 - runtime failure (verification miss, decrypt failure, setup failure, ...)
//	2 - usage error (unknown command/flag, missing value, invalid combination)
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run dispatches one command and returns its exit code. Every command writes
// exactly one JSON result to stdout on success; errors go to stderr.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return exitUsage
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "doctor":
		return cmdDoctor(rest, stdout, stderr)
	case "init-account":
		return cmdInitAccount(rest, stdout, stderr)
	case "relocate", "find", "locate":
		return cmdRelocate(rest, stdout, stderr)
	case "setup", "capture":
		return cmdSetup(rest, stdout, stderr)
	case "verify":
		return cmdVerify(rest, stdout, stderr)
	case "decrypt":
		return cmdDecrypt(rest, stdout, stderr)
	case "export":
		return cmdExport(rest, stdout, stderr)
	case "info":
		return cmdInfo(rest, stdout, stderr)
	case "-h", "--help", "help":
		usage(stdout)
		return exitOK
	case "-v", "--version", "version":
		fmt.Fprintln(stdout, appName+" "+version)
		return exitOK
	default:
		fmt.Fprintf(stderr, "%s: unknown command %q\n\n", appName, cmd)
		usage(stderr)
		return exitUsage
	}
}

func usage(w io.Writer) {
	fmt.Fprintf(w, `weixin-key %s — WeChat 4.1+ local DB key extraction (own account, own machine)

Usage:
  weixin-key doctor [--db-root <absolute-account-dir>] [--config <absolute-file>]
      [--out <absolute-dir>] [--pretty]
      Read-only new-machine checks. Does not read keys, DB contents or process
      memory. Exit 0 means inspection completed, NOT ready to export.

  weixin-key init-account --db-root <absolute-account-dir> --account <id>
      --config <absolute-new-file> [--pretty]
      Windows-only metadata bootstrap; parent must exist outside the source.
      Never replaces an existing file. Creates no key and runs no capture.
      On failure inspect applied before retrying. See docs/AGENT_WORKFLOW.md.

  weixin-key relocate [--dll <path-to-Weixin.dll>] [--pretty]
      Report static relocation candidates from Weixin.dll. A candidate does
      not establish a safe instruction/material profile or enable capture.

  weixin-key setup [--pretty]
      Verify cached/imported material first; active capture is quarantined.
      Ctrl+C requests cancellation and waits for cleanup. While recovery is
      pending, Ctrl+C requests one recovery retry, not a new capture job.

  weixin-key verify (--passphrase <hex> | --enc-key <hex>) [--db <path>] [--pretty]
      Verify material against local DBs and report per-DB matches.
      With --passphrase, derives per-DB enc_keys (PBKDF2-HMAC-SHA512, 256000
      rounds) and verifies each. With --enc-key, verifies the raw key directly.
      Exit 0 when at least one DB matched, 1 when none matched, 2 on usage
      errors. Prefer WECHAT_CLI_PASSPHRASE_HEX over --passphrase (the command
      line leaks via process lists and shell history).

  weixin-key decrypt --db <encrypted.db> --out <plain.db> [--enc-key <hex>]
      [--passphrase <hex>] [--no-config] [--pretty]
      Decrypt one WeChat 4.1+ SQLCipher DB to a plaintext SQLite file, in
      pure Go (no native WCDB). Key resolution order: --enc-key, --passphrase,
      WECHAT_CLI_PASSPHRASE_HEX, config cached enc_key (re-verified), config
      passphrase. --no-config disables the config fallback.

  weixin-key export --out <dir> [--db-root <account-dir>] [--account <id>]
      [--enc-key <hex> | --passphrase <hex>] [--no-config] [--pretty]
      One-shot export: snapshot every local DB (WAL included, sources stay
      read-only), decrypt what needs decrypting via the shared KeyResolver
      (cached material works fully offline), and write chat-v1 Markdown +
      JSONL + manifest into <dir>. Plaintext DBs are used as-is; encrypted
      DBs without usable material are reported as partial, not hidden.

  weixin-key info [--pretty]
      Show what key material the current config already holds.

Exit codes: 0 success, 1 runtime failure, 2 usage error.

Notes:
  - relocate output func_entry_rva values are code addresses (not secrets).
  - passphrase / enc_key ARE secrets: never commit, upload, or share them.
`, version)
}

// ---------------------------------------------------------------------------
// Unified flag parsing
// ---------------------------------------------------------------------------

// flagSpec declares which flags one subcommand accepts.
type flagSpec struct {
	values map[string]bool // flags that take a value: --db x / --db=x
	bools  map[string]bool // boolean flags: --pretty
}

// parsedFlags is the parse result for one subcommand invocation.
type parsedFlags struct {
	value       map[string]string
	isSet       map[string]bool
	positionals []string
}

// parseFlags accepts -name/--name, -name value/--name value and
// -name=value/--name=value. Unknown flags, missing values, values on boolean
// flags and unexpected positionals are usage errors (exit 2).
func parseFlags(cmd string, args []string, spec flagSpec) (*parsedFlags, error) {
	p := &parsedFlags{value: map[string]string{}, isSet: map[string]bool{}}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			p.positionals = append(p.positionals, args[i+1:]...)
			break
		}
		if len(a) < 2 || a[0] != '-' {
			p.positionals = append(p.positionals, a)
			continue
		}
		name := strings.TrimLeft(a, "-")
		if name == "" {
			return nil, fmt.Errorf("malformed flag %q", a)
		}
		val := ""
		hasVal := false
		if j := strings.IndexByte(name, '='); j >= 0 {
			val, name, hasVal = name[j+1:], name[:j], true
		}
		switch {
		case spec.bools[name]:
			if hasVal {
				return nil, fmt.Errorf("flag --%s does not take a value", name)
			}
			p.isSet[name] = true
		case spec.values[name]:
			if !hasVal {
				if i+1 >= len(args) {
					return nil, fmt.Errorf("flag --%s requires a value", name)
				}
				i++
				val = args[i]
			}
			p.value[name] = val
			p.isSet[name] = true
		default:
			return nil, fmt.Errorf("unknown flag --%s", name)
		}
	}
	return p, nil
}

func (f *parsedFlags) str(name string) string { return f.value[name] }
func (f *parsedFlags) flag(name string) bool  { return f.isSet[name] }

// usageError reports a usage error and returns exitUsage.
func usageError(cmd string, err error, stderr io.Writer) int {
	fmt.Fprintf(stderr, "%s: %s: %v\n", appName, cmd, err)
	fmt.Fprintf(stderr, "Run `%s help` for usage.\n", appName)
	return exitUsage
}

// fail reports a runtime error and returns exitError.
func fail(cmd string, err error, stderr io.Writer) int {
	fmt.Fprintf(stderr, "%s: %s: %v\n", appName, cmd, err)
	return exitError
}

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

func cmdRelocate(args []string, stdout, stderr io.Writer) int {
	f, err := parseFlags("relocate", args, flagSpec{
		values: map[string]bool{"dll": true, "path": true},
		bools:  map[string]bool{"pretty": true},
	})
	if err != nil {
		return usageError("relocate", err, stderr)
	}
	dll := f.str("dll")
	if dll == "" {
		dll = f.str("path")
	}
	if len(f.positionals) > 1 {
		return usageError("relocate", errors.New("accepts at most one positional <dll path>"), stderr)
	}
	if dll == "" && len(f.positionals) == 1 {
		dll = f.positionals[0]
	}
	if dll == "" {
		p, derr := wxkey.DefaultWeixinDLLPath()
		if derr != nil {
			return fail("relocate", derr, stderr)
		}
		dll = p
	}
	rep, err := wxkey.RelocateHookPoints(dll, nil)
	if err != nil {
		return fail("relocate", err, stderr)
	}
	if err := printJSON(stdout, rep, f.flag("pretty")); err != nil {
		return fail("relocate", err, stderr)
	}
	return exitOK
}

func cmdSetup(args []string, stdout, stderr io.Writer) int {
	f, err := parseFlags("setup", args, flagSpec{
		bools: map[string]bool{"pretty": true},
	})
	if err != nil {
		return usageError("setup", err, stderr)
	}
	if len(f.positionals) > 0 {
		return usageError("setup", fmt.Errorf("unexpected argument %q", f.positionals[0]), stderr)
	}
	if !wxkey.SetupSupported() {
		return fail("setup", errors.New(wxkey.UnsupportedSetupMessage()), stderr)
	}
	res, setupStderr, err := runSetupWithSignals(stderr)
	if err != nil {
		if setupStderr != "" {
			fmt.Fprint(stderr, setupStderr)
		}
		return fail("setup", err, stderr)
	}
	// Never print raw key material to stdout. Summarize only.
	out := map[string]any{
		"pid":         res.PID,
		"scan_root":   res.Root,
		"wxid":        res.WxID,
		"config_path": res.ConfigPath,
		"keys_found":  len(res.Keys),
		"note":        "key material written to config; not echoed to stdout",
	}
	if err := printJSON(stdout, out, f.flag("pretty")); err != nil {
		return fail("setup", err, stderr)
	}
	return exitOK
}

func cmdVerify(args []string, stdout, stderr io.Writer) int {
	f, err := parseFlags("verify", args, flagSpec{
		values: map[string]bool{"passphrase": true, "enc-key": true, "db": true},
		bools:  map[string]bool{"pretty": true},
	})
	if err != nil {
		return usageError("verify", err, stderr)
	}
	if len(f.positionals) > 0 {
		return usageError("verify", fmt.Errorf("unexpected argument %q", f.positionals[0]), stderr)
	}
	passphrase := strings.TrimSpace(f.str("passphrase"))
	encKey := strings.TrimSpace(f.str("enc-key"))
	if passphrase != "" && encKey != "" {
		return usageError("verify", errors.New("--passphrase and --enc-key are mutually exclusive"), stderr)
	}
	if passphrase != "" {
		fmt.Fprintln(stderr, appName+": warning: --passphrase on the command line leaks via process lists and shell history; prefer WECHAT_CLI_PASSPHRASE_HEX")
	} else if encKey == "" {
		passphrase = strings.TrimSpace(os.Getenv("WECHAT_CLI_PASSPHRASE_HEX"))
	}
	if passphrase == "" && encKey == "" {
		return fail("verify", errors.New("requires --passphrase <hex>, --enc-key <hex>, or WECHAT_CLI_PASSPHRASE_HEX"), stderr)
	}

	var res *wxkey.VerifyResult
	if encKey != "" {
		res, err = wxkey.VerifyEncKeyAgainstDBs(encKey, f.str("db"))
	} else {
		res, err = wxkey.VerifyPassphraseAgainstDBs(passphrase, f.str("db"))
	}
	if err != nil {
		return fail("verify", err, stderr)
	}
	if err := printJSON(stdout, res, f.flag("pretty")); err != nil {
		return fail("verify", err, stderr)
	}
	if res.Matched == 0 {
		fmt.Fprintln(stderr, appName+": verify: material did not match any target DB")
		return exitError
	}
	return exitOK
}

func cmdDecrypt(args []string, stdout, stderr io.Writer) int {
	f, err := parseFlags("decrypt", args, flagSpec{
		values: map[string]bool{"db": true, "out": true, "enc-key": true, "passphrase": true},
		bools:  map[string]bool{"pretty": true, "config-passphrase": true, "no-config": true},
	})
	if err != nil {
		return usageError("decrypt", err, stderr)
	}
	if len(f.positionals) > 0 {
		return usageError("decrypt", fmt.Errorf("unexpected argument %q", f.positionals[0]), stderr)
	}
	if f.str("db") == "" {
		return usageError("decrypt", errors.New("--db is required"), stderr)
	}
	if f.str("out") == "" {
		return usageError("decrypt", errors.New("--out is required"), stderr)
	}
	if f.str("passphrase") != "" {
		fmt.Fprintln(stderr, appName+": warning: --passphrase on the command line leaks via process lists and shell history; prefer WECHAT_CLI_PASSPHRASE_HEX")
	}
	// --config-passphrase is accepted for backward compatibility; the config
	// fallback (cache first, then passphrase) is now the default.
	opts := wxkey.DecryptOptions{
		EncKeyHex:     f.str("enc-key"),
		PassphraseHex: f.str("passphrase"),
		NoConfig:      f.flag("no-config"),
	}
	res, err := wxkey.DecryptDatabase(f.str("db"), f.str("out"), opts)
	if err != nil {
		return fail("decrypt", err, stderr)
	}
	if err := printJSON(stdout, res, f.flag("pretty")); err != nil {
		return fail("decrypt", err, stderr)
	}
	return exitOK
}

func cmdInfo(args []string, stdout, stderr io.Writer) int {
	f, err := parseFlags("info", args, flagSpec{
		bools: map[string]bool{"pretty": true},
	})
	if err != nil {
		return usageError("info", err, stderr)
	}
	if len(f.positionals) > 0 {
		return usageError("info", fmt.Errorf("unexpected argument %q", f.positionals[0]), stderr)
	}
	res, err := wxkey.ConfigKeyInfo()
	if err != nil {
		return fail("info", err, stderr)
	}
	if err := printJSON(stdout, res, f.flag("pretty")); err != nil {
		return fail("info", err, stderr)
	}
	return exitOK
}

func cmdExport(args []string, stdout, stderr io.Writer) int {
	f, err := parseFlags("export", args, flagSpec{
		values: map[string]bool{"out": true, "db-root": true, "account": true, "enc-key": true, "passphrase": true, "img-key": true},
		bools:  map[string]bool{"pretty": true, "no-config": true, "full": true},
	})
	if err != nil {
		return usageError("export", err, stderr)
	}
	if len(f.positionals) > 0 {
		return usageError("export", fmt.Errorf("unexpected argument %q", f.positionals[0]), stderr)
	}
	if f.str("out") == "" {
		return usageError("export", errors.New("--out is required"), stderr)
	}
	if f.str("enc-key") != "" && f.str("passphrase") != "" {
		return usageError("export", errors.New("--enc-key and --passphrase are mutually exclusive"), stderr)
	}
	if f.str("passphrase") != "" {
		fmt.Fprintln(stderr, appName+": warning: --passphrase on the command line leaks via process lists and shell history; prefer WECHAT_CLI_PASSPHRASE_HEX")
	}
	imgKeyHex := strings.TrimSpace(f.str("img-key"))
	if imgKeyHex == "" {
		imgKeyHex = strings.TrimSpace(os.Getenv("WECHAT_CLI_IMGKEY_HEX"))
	}
	rep, err := export.Run(export.Options{
		DBRoot:    f.str("db-root"),
		OutDir:    f.str("out"),
		AccountID: f.str("account"),
		Full:      f.flag("full"),
		ImgKeyHex: imgKeyHex,
		Resolve: wxkey.ResolveOptions{
			EncKeyHex:     f.str("enc-key"),
			PassphraseHex: f.str("passphrase"),
			NoConfig:      f.flag("no-config"),
		},
	})
	if rep != nil {
		// Surface the report JSON on stdout even for partial/failed runs;
		// the exit code carries the verdict.
		if perr := printJSON(stdout, rep, f.flag("pretty")); perr != nil {
			return fail("export", perr, stderr)
		}
	}
	if err != nil {
		return fail("export", err, stderr)
	}
	switch rep.Status {
	case "complete":
		return exitOK
	case "partial":
		fmt.Fprintln(stderr, appName+": export: partial result (see manifest.json)")
		return exitError
	default:
		return exitError
	}
}

func printJSON(stdout io.Writer, v any, pretty bool) error {
	enc := json.NewEncoder(stdout)
	if pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode result: %w", err)
	}
	return nil
}
