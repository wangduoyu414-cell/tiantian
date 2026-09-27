//go:build windows

package wxkey

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"weixin-key/internal/config"
	"weixin-key/internal/wcdb"
)

type setupDeps struct {
	processes func() ([]windowsProcess, error)
	restart   func(context.Context, SetupOptions, []windowsSourceDB, map[string]bool, *setupScan) (uint32, error)
	routes    func([]windowsProcess, []windowsSourceDB, map[string]bool, *setupScan, *windowsSetupStats, *uint32, func(string)) error
}

func defaultSetupDeps() setupDeps {
	return setupDeps{processes: windowsTargetProcesses, restart: windowsRestartAccount, routes: runWindowsRoutes}
}

func runSetup() (*SetupResult, string, error) {
	return runSetupContext(context.Background(), defaultSetupOptions())
}

func defaultSetupOptions() SetupOptions {
	return SetupOptions{
		Restart: windowsRestartCaptureEnabled(),
		ExePath: firstEnv("WECHAT_CLI_WECHAT_EXE", "WX_MCP_WECHAT_EXE"),
	}
}

func runSetupContext(ctx context.Context, opts SetupOptions) (*SetupResult, string, error) {
	return runSetupWithDeps(ctx, opts, defaultSetupDeps())
}

func runSetupWithDeps(ctx context.Context, opts SetupOptions, deps setupDeps) (*SetupResult, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, "", err
	}
	initialRoot, initialWxid := cfg.DBRoot, cfg.Wxid
	if opts.DBRoot != "" {
		cfg.DBRoot = opts.DBRoot
		cfg.Wxid = wxidFromAccountDir(opts.DBRoot)
	}
	if cfg.DBRoot == "" {
		cfg.DBRoot, cfg.Wxid, err = config.AutoDetectDBRoot()
		if err != nil {
			return nil, "", err
		}
	}
	if cfg.Wxid == "" {
		cfg.Wxid = wxidFromAccountDir(cfg.DBRoot)
	}
	opts.DBRoot = cfg.DBRoot
	dbs, salts, err := windowsListSourceDBs(cfg.DBRoot)
	if err != nil {
		return nil, "", err
	}
	if len(dbs) == 0 {
		return nil, "", errors.New("no encrypted databases found for selected account")
	}
	scan := newSetupScan()
	scan.ctx = ctx
	scan.recovery = opts.Recovery
	progress := func(s string) {
		if opts.Progress != nil {
			opts.Progress(s)
		}
	}
	progress("验证所选账号已有材料（不操作微信）")
	resolver := NewKeyResolver()
	for _, db := range dbs {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		key, err := resolver.ResolveForDB(db.path, opts.Resolve)
		if err == nil {
			scan.noteRawKey(db.salt, key.EncKeyHex, key.Source)
		} else if !errors.Is(err, ErrNoKeyMaterial) && !errors.Is(err, ErrKeyMaterialRejected) {
			return nil, "", fmt.Errorf("preflight %s: %w", db.rel, err)
		} else if opts.Resolve.EncKeyHex != "" || opts.Resolve.PassphraseHex != "" {
			return nil, "", err // explicit invalid material must not trigger capture
		}
	}
	stats := windowsSetupStats{SourceDBs: len(dbs), TargetSalts: len(salts)}
	var firstPID uint32
	var captureErr error

	// An explicitly supplied local capture file is an offline route. It must
	// run before process discovery/restart, and file errors are not permission
	// to silently replace an import with active process control.
	if scan.lenFound() < len(salts) && firstEnv("WECHAT_CLI_CAPTURE_KEY_FILE", "WX_MCP_CAPTURE_KEY_FILE") != "" {
		start, before := time.Now(), scan.lenFound()
		progress("验证已授权导入材料（不操作微信）")
		firstPID, err = windowsNativeCaptureScan(dbs, salts, scan, windowsRouteDeadline(time.Time{}, windowsRouteShare("capture-file")))
		exit := "completed"
		if err != nil {
			exit = "error"
		}
		stats.Routes = append(stats.Routes, windowsRouteStat{Name: "capture-file", DurationMs: time.Since(start).Milliseconds(), NewCoverage: scan.lenFound() - before, Exit: exit})
		if err != nil {
			return nil, scan.diagString(), err
		}
	}

	// A restart is a separate lifecycle phase, before any computation budget.
	// It is reachable with NO process running. The controller only closes
	// processes whose open files prove association with the selected account.
	if scan.lenFound() < len(salts) && opts.Restart {
		if opts.LoginTimeout <= 0 {
			opts.LoginTimeout = windowsLoginTimeout()
		}
		loginCtx, cancel := context.WithTimeout(ctx, opts.LoginTimeout)
		start, before := time.Now(), scan.lenFound()
		scan.ctx = loginCtx
		firstPID, captureErr = deps.restart(loginCtx, opts, dbs, salts, scan)
		scan.ctx = ctx
		cancel()
		exit := "completed"
		if captureErr != nil {
			exit = "error"
		}
		stats.Routes = append(stats.Routes, windowsRouteStat{Name: "restart-capture",
			DurationMs: time.Since(start).Milliseconds(), NewCoverage: scan.lenFound() - before, Exit: exit})
		if ctx.Err() != nil {
			return nil, scan.diagString(), errors.Join(ctx.Err(), captureErr)
		}
		// Do not hide launch/permission/timeout errors by starting more routes.
		if captureErr != nil {
			return nil, scan.diagString(), captureErr
		}
	}

	if scan.lenFound() < len(salts) && captureErr == nil {
		procs, err := deps.processes()
		if err != nil {
			return nil, scan.diagString(), err
		}
		if len(procs) == 0 {
			return nil, scan.diagString(), errors.New("no running WeChat process; approve account-scoped startup or provide valid offline material")
		}
		captureErr = deps.routes(procs, dbs, salts, scan, &stats, &firstPID, progress)
		if captureErr != nil {
			return nil, scan.diagString(), captureErr
		}
	}
	if ctx.Err() != nil {
		return nil, scan.diagString(), ctx.Err()
	}
	var results []ResultEntry
	verified := map[string]string{}
	for _, db := range dbs {
		if err := ctx.Err(); err != nil {
			return nil, scan.diagString(), err
		}
		key, ok := scan.found[db.salt]
		if !ok || !verifyEncKeyStrict(db.path, key) {
			continue
		}
		verified[db.salt] = key
		results = append(results, ResultEntry{DBRel: filepath.ToSlash(db.rel), DBPath: db.path,
			SaltHex: db.salt, KeyHex: key, VerifyAs: scan.foundSource[db.salt]})
	}
	if len(verified) == 0 {
		if captureErr == nil {
			captureErr = errors.New("no usable material found; cause not established")
		}
		return nil, scan.diagString(), captureErr
	}
	epoch := time.Now().Unix()
	if err := config.Update(func(current *config.Config) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if current.DBRoot != initialRoot || current.Wxid != initialWxid {
			return errors.New("account configuration changed during capture; refusing to commit")
		}
		current.DBRoot, current.Wxid = cfg.DBRoot, cfg.Wxid
		if scan.passphraseHex != "" {
			current.Passphrase = scan.passphraseHex
			current.PassphraseSource = scan.passphraseSource
			current.PassphraseEpoch = epoch
			current.KDF = wcdb.DefaultKDF
		} else if initialRoot != cfg.DBRoot {
			// An old account's passphrase must not follow a new selection.
			current.Passphrase, current.PassphraseSource, current.KDF = "", "", ""
			current.PassphraseEpoch = 0
		}
		for salt, key := range verified {
			current.SetVerifiedKey(salt, key, scan.foundSource[salt], epoch)
		}
		current.KeyEpoch, current.KeyPID = epoch, int(firstPID)
		return nil
	}); err != nil {
		return nil, scan.diagString(), err
	}
	stats.MatchedSalts, stats.VerifiedDBs = len(verified), len(results)
	statsJSON, _ := json.Marshal(stats)
	cfgPath, _ := config.Path()
	res := &SetupResult{PID: int(firstPID), Root: cfg.DBRoot, WxID: cfg.Wxid,
		ConfigPath: cfgPath, Stats: statsJSON, Results: results, Keys: verified}
	msg := fmt.Sprintf("verified %d/%d database files", len(results), len(dbs))
	if len(results) != len(dbs) {
		return res, scan.diagString(), fmt.Errorf("%s; incomplete coverage: %w", msg, errors.Join(captureErr, errors.New("required databases remain unreadable")))
	}
	return res, msg, nil
}

func windowsLoginTimeout() time.Duration {
	if d, err := time.ParseDuration(firstEnv("WECHAT_CLI_LOGIN_TIMEOUT")); err == nil && d > 0 {
		return d
	}
	return 10 * time.Minute
}

type setupRoute struct {
	name string
	run  func(uint32, time.Time) error
}

func runWindowsRoutes(procs []windowsProcess, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan,
	stats *windowsSetupStats, firstPID *uint32, progress func(string)) error {
	// An explicitly requested but unsafe route must fail before passive memory
	// work, not silently fall back. Neither environment flag bypasses the gate.
	if windowsKeyHookEnabled() || firstEnv("WECHAT_CLI_WX_KEY_DLL") != "" {
		return windowsActiveCapturePreflight("requested-active-route")
	}
	targets, err := selectAccountTargets(scan.ctx, procs, dbs, false, accountFileOwners, pinControlTarget)
	if err != nil {
		return err
	}
	defer releaseAccountTargets(targets)
	procs = nil
	for _, target := range targets {
		procs = append(procs, windowsProcess{pid: target.identity.pid, exe: target.exe})
	}
	marks := map[uint32]map[uintptr]regionFingerprint{}
	// Retain the process handles until the route sequence ends, so a PID
	// cannot be recycled while its job-local baseline is still referenced.
	var observedHandles []uintptr
	defer func() {
		for _, h := range observedHandles {
			procCloseHandle.Call(h)
		}
	}()
	routes := []setupRoute{
		{"primary-literal", func(pid uint32, dl time.Time) error { return windowsScanProcess(pid, salts, scan, dl) }},
		{"signatures", func(pid uint32, dl time.Time) error {
			_, _, _, err := windowsScanSignatures(pid, dbs, salts, scan, dl)
			return err
		}},
	}
	// Third-party active helpers are never activated solely by being installed.
	if firstEnv("WECHAT_CLI_WX_KEY_DLL") != "" {
		routes = append(routes, setupRoute{"wxkey-dll", func(pid uint32, dl time.Time) error {
			_, err := windowsDLLScan([]windowsProcess{{pid: pid}}, dbs, salts, scan, dl)
			return err
		}})
	}
	if windowsKeyHookEnabled() {
		routes = append(routes, setupRoute{"hook", func(pid uint32, dl time.Time) error { return windowsHookScan(pid, dbs, salts, scan, dl) }})
	}
	routes = append(routes,
		setupRoute{"d0-object", func(pid uint32, dl time.Time) error { return windowsKeyObjectScan(pid, dbs, salts, scan, dl) }},
		setupRoute{"d1-salt", func(pid uint32, dl time.Time) error { return windowsSaltNeighborhoodScan(pid, dbs, scan, dl) }},
		setupRoute{"d2-heap", func(pid uint32, dl time.Time) error {
			return windowsBruteForceScan(pid, dbs, salts, scan, dl, marks[pid])
		}},
		setupRoute{"a-poll", func(pid uint32, dl time.Time) error { return windowsPollScan(pid, dbs, salts, scan, dl) }},
	)
	if strings.EqualFold(firstEnv("WECHAT_CLI_BRUTE_PASSPHRASE"), "1") {
		routes = append(routes, setupRoute{"brute-passphrase", func(pid uint32, dl time.Time) error {
			_, _, err := windowsBruteForcePassphrase(pid, dbs, salts, scan, dl)
			return err
		}})
	}
	var totalDeadline time.Time
	if timeout := windowsKeyScanTimeout(); timeout > 0 {
		totalDeadline = time.Now().Add(timeout)
	}
	var lastErr error
	for i, route := range routes {
		if scan.lenFound() == len(salts) {
			break
		}
		if scan.stopped(totalDeadline) {
			return errors.Join(scan.ctx.Err(), errWindowsKeyScanDeadline)
		}
		share := windowsRouteShare(route.name)
		if !totalDeadline.IsZero() {
			share = min(share, time.Until(totalDeadline)/time.Duration(len(routes)-i))
		}
		deadline := windowsRouteDeadline(totalDeadline, share)
		start, before := time.Now(), scan.lenFound()
		progress("采集策略：" + route.name)
		var routeErr error
		for j, p := range procs {
			if scan.lenFound() == len(salts) {
				break
			}
			if scan.stopped(deadline) {
				routeErr = errors.Join(scan.ctx.Err(), errWindowsKeyScanDeadline)
				break
			}
			if err := targets[j].alive(); err != nil {
				return err
			}
			pidDL := windowsRouteDeadline(deadline, time.Until(deadline)/time.Duration(len(procs)-j))
			if i == 0 && j < 32 && windowsDiffScanEnabled() {
				h, _, openErr := procOpenProcess.Call(processVMRead|processQueryInformation, 0, uintptr(p.pid))
				if h == 0 {
					return processAccessError(p.pid, processVMRead|processQueryInformation, openErr)
				}
				observedHandles = append(observedHandles, h)
				markDL := windowsRouteDeadline(pidDL, min(250*time.Millisecond, time.Until(pidDL)/10))
				regions := windowsCollectWritableHeapRegions(h, func() bool { return scan.stopped(markDL) })
				read := windowsProcessRegionReader(h)
				marks[p.pid] = snapshotRegions(func(base uintptr, n int) ([]byte, error) {
					if scan.stopped(markDL) {
						return nil, errWindowsKeyScanDeadline
					}
					return read(base, n)
				}, regions)
				scan.addDiag("change-priority baseline pid=%d regions=%d (bounded sampling; no speedup claim)", p.pid, len(marks[p.pid]))
			}
			beforePID := scan.lenFound()
			err := route.run(p.pid, pidDL)
			stats.ScannedProcesses++
			stats.ScannedPIDs = append(stats.ScannedPIDs, p.pid)
			if *firstPID == 0 && scan.lenFound() > beforePID {
				*firstPID = p.pid
			}
			if err != nil {
				routeErr = err
				scan.addDiag("%s pid=%d: %v", route.name, p.pid, err)
				// An access denial is not a reason to try a more invasive route.
				if isProcessAccessDenied(err) {
					return err
				}
			}
		}
		exit := "completed"
		if routeErr != nil {
			exit = "error"
			if errors.Is(routeErr, errWindowsKeyScanDeadline) {
				exit = "budget-timeout"
			}
			lastErr = routeErr
		}
		stats.Routes = append(stats.Routes, windowsRouteStat{Name: route.name,
			DurationMs: time.Since(start).Milliseconds(), NewCoverage: scan.lenFound() - before, Exit: exit})
	}
	return lastErr
}
