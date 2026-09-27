// Package guicore holds the GUI's display-independent logic: the one-click
// export state machine, persisted non-secret settings (directory memory) and
// the offline material preflight that decides whether WeChat must be
// restarted at all. The window layer (Gio) only renders Job state and calls
// Start/Cancel.
package guicore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"weixin-key/internal/safefile"

	"weixin-key/internal/config"
	"weixin-key/internal/export"
	"weixin-key/internal/pathguard"
	"weixin-key/internal/wxkey"
)

// ---------------------------------------------------------------------------
// Settings (non-secret; secrets never live here)
// ---------------------------------------------------------------------------

// Settings are the GUI's persisted non-secret preferences.
type Settings struct {
	LastExportDir string `json:"last_export_dir,omitempty"`
	DBRoot        string `json:"db_root,omitempty"`
}

// SettingsPath lives next to the key config but contains no secrets.
func SettingsPath() (string, error) {
	cfgPath, err := config.Path()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(cfgPath), "gui-settings.json"), nil
}

// LoadSettings reads the settings file; a missing file yields empty settings.
func LoadSettings() (*Settings, error) {
	p, err := SettingsPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return &Settings{}, nil
	}
	if err != nil {
		return nil, err
	}
	var s Settings
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// SaveSettings writes settings atomically (temp + rename), 0600.
func SaveSettings(s *Settings) error {
	p, err := SettingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	if err := safefile.Replace(tmp, p); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// Export directory validation + default suggestion
// ---------------------------------------------------------------------------

// SuggestExportDir returns a default under the user's Documents folder, never
// inside the WeChat source tree.
func SuggestExportDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Documents", "微信导出"), nil
}

// ValidateExportDir checks a user-chosen directory. It must not be empty, a
// file, the source directory or inside the source tree, and it must be
// creatable/writable. The old value is preserved by the caller on failure.
func ValidateExportDir(dir, sourceRoot string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return errors.New("目录为空")
	}
	abs, err := pathguard.ResolveOutput(dir, sourceRoot)
	if err != nil {
		return fmt.Errorf("路径无效: %w", err)
	}
	// Writability probe: create+remove a temp entry in the (created) dir.
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return fmt.Errorf("目录不可创建: %w", err)
	}
	probe, err := os.CreateTemp(abs, ".weixin-key-probe-*")
	if err != nil {
		return fmt.Errorf("目录不可写: %w", err)
	}
	probe.Close()
	os.Remove(probe.Name())
	return nil
}

func sameOrChild(out, src string) bool {
	out = filepath.Clean(out)
	src = filepath.Clean(src)
	if out == src {
		return true
	}
	rel, err := filepath.Rel(src, out)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// ---------------------------------------------------------------------------
// Account discovery (multi-account safe: never guesses "the first one")
// ---------------------------------------------------------------------------

// AccountInfo is one account directory under a WeChat data root.
type AccountInfo struct {
	Dir  string `json:"dir"`  // full path (contains db_storage)
	WxID string `json:"wxid"` // parsed from the dir name
}

// ListAccounts returns every account dir with a db_storage under root.
// The caller must let the user choose when more than one exists; with exactly
// one, it may be auto-selected; with zero, the root is wrong.
func ListAccounts(root string) ([]AccountInfo, error) {
	if root == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("读取数据根目录失败: %w", err)
	}
	var out []AccountInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		switch e.Name() {
		case "all_users", "applet", "backup", "wmpf":
			continue
		}
		full := filepath.Join(root, e.Name())
		if _, err := os.Stat(filepath.Join(full, "db_storage")); err == nil {
			out = append(out, AccountInfo{Dir: full, WxID: accountIDFromDir(full)})
		}
	}
	return out, nil
}

// DetectDataRoot locates the WeChat data root for the GUI: remembered/config
// account dir first, then strict auto-detect (single account), then the first
// base that contains ANY accounts (multi-account roots are fine here - the
// account selector resolves the ambiguity, not a guess).
func DetectDataRoot(cfgDBRoot, settingsDBRoot string) (dataRoot, accountDir string) {
	if settingsDBRoot != "" && hasDBStorageDir(settingsDBRoot) {
		return filepath.Dir(settingsDBRoot), settingsDBRoot
	}
	if cfgDBRoot != "" && hasDBStorageDir(cfgDBRoot) {
		return filepath.Dir(cfgDBRoot), cfgDBRoot
	}
	if root, _, err := config.AutoDetectDBRoot(); err == nil {
		return filepath.Dir(root), root
	}
	// Multi-account or custom root: use the first base that has accounts.
	if bases, err := config.DefaultWeChatBases(); err == nil {
		for _, base := range bases {
			accts, err := ListAccounts(base)
			if err == nil && len(accts) > 0 {
				if len(accts) == 1 {
					return base, accts[0].Dir
				}
				return base, "" // root known, account must be chosen explicitly
			}
		}
	}
	return "", ""
}

func hasDBStorageDir(dir string) bool {
	if dir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(dir, "db_storage"))
	return err == nil
}

// ---------------------------------------------------------------------------
// Offline preflight: is capture needed at all?
// ---------------------------------------------------------------------------

// DBPreflight is one DB's offline-resolvability verdict.
type DBPreflight struct {
	Rel        string `json:"rel"`
	Encrypted  bool   `json:"encrypted"`
	Resolvable bool   `json:"resolvable"`
	KeySource  string `json:"key_source,omitempty"`
	Error      string `json:"error,omitempty"`
}

// Preflight is the aggregate verdict.
type Preflight struct {
	Root             string        `json:"root"`
	AccountID        string        `json:"account_id"`
	DBs              []DBPreflight `json:"dbs"`
	OfflineReady     bool          `json:"offline_ready"`     // everything resolvable without touching WeChat
	NeedsCapture     []string      `json:"needs_capture"`     // DBs with missing/invalid material
	PlaintextSkipped int           `json:"plaintext_skipped"` // plaintext DBs need no material
}

// RunPreflight classifies every source DB and verifies whether existing
// material (cache/passphrase/flags/env) can open it, offline, without
// touching any WeChat process.
func RunPreflight(dbRoot string, ropts wxkey.ResolveOptions) (*Preflight, error) {
	return RunPreflightContext(context.Background(), dbRoot, ropts)
}

func RunPreflightContext(ctx context.Context, dbRoot string, ropts wxkey.ResolveOptions) (*Preflight, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	storage := filepath.Join(dbRoot, "db_storage")
	var rels []string
	err := filepath.WalkDir(storage, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".db") {
			rel, _ := filepath.Rel(storage, p)
			rels = append(rels, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	pf := &Preflight{Root: dbRoot, AccountID: accountIDFromDir(dbRoot), OfflineReady: true}
	if len(rels) == 0 {
		pf.OfflineReady = false
		return pf, fmt.Errorf("no .db files under %s", storage)
	}
	resolver := wxkey.NewKeyResolver()
	for _, rel := range rels {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		full := filepath.Join(storage, filepath.FromSlash(rel))
		d := DBPreflight{Rel: rel, Encrypted: !isPlaintextSQLite(full)}
		if !d.Encrypted {
			d.Resolvable = true
			d.KeySource = "plaintext"
			pf.PlaintextSkipped++
		} else {
			_, err := resolver.ResolveForDB(full, ropts)
			switch {
			case err == nil:
				d.Resolvable = true
				d.KeySource = "resolved-offline"
			case errors.Is(err, wxkey.ErrNoKeyMaterial):
				d.Error = "no-material"
			case errors.Is(err, wxkey.ErrKeyMaterialRejected):
				d.Error = "stale-material"
			default:
				d.Error = err.Error()
			}
			if !d.Resolvable {
				pf.NeedsCapture = append(pf.NeedsCapture, rel)
				pf.OfflineReady = false
			}
		}
		pf.DBs = append(pf.DBs, d)
	}
	return pf, nil
}

func isPlaintextSQLite(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	hdr := make([]byte, 16)
	if _, err := f.Read(hdr); err != nil {
		return false
	}
	return string(hdr) == "SQLite format 3\x00"
}

func accountIDFromDir(root string) string {
	name := filepath.Base(filepath.Clean(root))
	if idx := strings.LastIndex(name, "_"); idx > 0 {
		return name[:idx]
	}
	return name
}

// ---------------------------------------------------------------------------
// Job: the one-click export state machine
// ---------------------------------------------------------------------------

// Stage is the job's coarse state.
type Stage string

const (
	StageIdle      Stage = "idle"
	StageValidate  Stage = "validate"
	StagePreflight Stage = "preflight"
	// StageNeedsCapture waits for the user to approve a WeChat restart.
	StageNeedsCapture Stage = "needs_capture"
	StageCapturing    Stage = "capturing"
	StageRecovering   Stage = "recovering" // still owns capture resources, NOT done
	StageExporting    Stage = "exporting"
	StageDone         Stage = "done" // complete or partial (see Report.Status)
	StageCancelled    Stage = "cancelled"
	StageFailed       Stage = "failed"
)

// CaptureFunc performs material capture (e.g. close WeChat, relaunch, scan).
// It must honor ctx cancellation. The GUI injects the platform adapter.
type CaptureFunc func(context.Context, wxkey.SetupOptions) error

// JobOptions parameterizes one job run.
type JobOptions struct {
	DBRoot  string
	OutDir  string
	Resolve wxkey.ResolveOptions
	// Capture is invoked only when preflight finds unresolvable material.
	// Nil means "no capture available": the job reports needs-capture as failed.
	Capture CaptureFunc
}

// JobEvent is one progress line (stage transition or detail).
type JobEvent struct {
	Time    time.Time `json:"time"`
	Stage   Stage     `json:"stage"`
	Message string    `json:"message"`
}

// Job is a single one-click export run. One Job instance serializes runs;
// Start while running is an error (button re-entry protection).
type Job struct {
	mu         sync.Mutex
	stage      Stage
	events     []JobEvent
	report     *export.Report
	preflight  *Preflight
	err        error
	running    bool
	cancel     <-chan struct{}
	ctx        context.Context
	cancelFunc context.CancelFunc
	approve    chan struct{} // closed by ApproveCapture when parked at needs_capture
	done       chan struct{} // closed only after capture cleanup and worker join
	recovery   *wxkey.CaptureRecovery
	closing    bool // closing is irreversible; no new run may race shutdown
}

func NewJob() *Job {
	return &Job{stage: StageIdle}
}

// State snapshots the current job state.
func (j *Job) State() (stage Stage, events []JobEvent, rep *export.Report, pf *Preflight, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	stage, events = j.stage, append([]JobEvent(nil), j.events...)
	// State is authoritative even if a late Progress callback raced the
	// notification. Display text alone must never hide a retained owner.
	if waiting, message := j.recovery.State(); waiting {
		stage = StageRecovering
		if len(events) == 0 || events[len(events)-1].Stage != StageRecovering {
			events = append(events, JobEvent{Time: time.Now(), Stage: StageRecovering, Message: message})
		}
	}
	return stage, events, j.report, j.preflight, j.err
}

// Running reports whether a run is active.
func (j *Job) Running() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.running
}

// Cancel requests cancellation; the job stops between pipeline steps.
func (j *Job) Cancel() {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.cancelFunc != nil {
		j.cancelFunc()
	}
}

// RequestClose prevents new work, requests cancellation and returns a completion
// signal. Closing a window must not terminate a still-owned capture session.
// Callers keep a visible recovery UI until this signal closes.
func (j *Job) RequestClose() <-chan struct{} {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.closing = true
	if j.cancelFunc != nil {
		j.cancelFunc()
	}
	if j.done == nil {
		j.done = make(chan struct{})
		close(j.done)
	}
	return j.done
}

// RetryRecovery requests another reconciled cleanup attempt, not another run.
func (j *Job) RetryRecovery() bool {
	j.mu.Lock()
	r := j.recovery
	j.mu.Unlock()
	return r.Retry()
}

func (j *Job) recoveryChanged(waiting bool, message string) {
	if waiting {
		j.setStage(StageRecovering, message)
	} else {
		j.setStage(StageCapturing, "调试恢复已完成，等待采集工作线程退出")
	}
}

// ApproveCapture releases a job parked at StageNeedsCapture into the capture
// phase. It is a no-op in any other stage.
func (j *Job) ApproveCapture() {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.stage != StageNeedsCapture || j.approve == nil {
		return
	}
	select {
	case <-j.approve:
	default:
		close(j.approve)
	}
}

func (j *Job) setStage(s Stage, msg string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.stage = s
	if msg != "" {
		j.events = append(j.events, JobEvent{Time: time.Now(), Stage: s, Message: msg})
	}
}

func (j *Job) cancelled() bool {
	j.mu.Lock()
	ch := j.cancel
	j.mu.Unlock()
	if ch == nil {
		return false
	}
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// Start launches the job in the background. It validates inputs first and
// returns synchronously on usage errors; pipeline failures land in State().
func (j *Job) Start(opts JobOptions) error {
	j.mu.Lock()
	if j.closing {
		j.mu.Unlock()
		return errors.New("正在安全退出，不能启动新任务")
	}
	if j.running {
		j.mu.Unlock()
		return errors.New("已有导出任务在运行")
	}
	j.stage = StageValidate
	j.events = nil
	j.report = nil
	j.preflight = nil
	j.err = nil
	j.ctx, j.cancelFunc = context.WithCancel(context.Background())
	j.cancel = j.ctx.Done()
	j.approve = make(chan struct{})
	j.running = true
	j.done = make(chan struct{})
	j.recovery = wxkey.NewCaptureRecovery(j.recoveryChanged)
	j.mu.Unlock()

	j.setStage(StageValidate, "校验目录与账号")
	if err := ValidateExportDir(opts.OutDir, opts.DBRoot); err != nil {
		j.mu.Lock()
		j.err = err
		j.stage = StageFailed
		j.running = false
		j.cancelFunc()
		close(j.done)
		j.mu.Unlock()
		return err
	}

	go j.run(opts)
	return nil
}

// ApproveCapture continues a job parked at needs_capture. M2's GUI calls this
// after the user confirms a WeChat restart. For now a parked job without an
// injected Capture func fails with a clear message.
func (j *Job) run(opts JobOptions) {
	defer j.cancelFunc()
	defer func() {
		j.mu.Lock()
		j.running = false
		close(j.done)
		j.mu.Unlock()
	}()

	j.setStage(StagePreflight, "识别明文/加密与材料可用性")
	pf, err := RunPreflightContext(j.ctx, opts.DBRoot, opts.Resolve)
	if err != nil {
		j.fail(err)
		return
	}
	j.mu.Lock()
	j.preflight = pf
	j.mu.Unlock()
	for _, d := range pf.DBs {
		state := "离线可用"
		if !d.Resolvable {
			state = "需要获取材料(" + d.Error + ")"
		}
		j.setStage(StagePreflight, fmt.Sprintf("%s：%s", d.Rel, state))
	}

	if !pf.OfflineReady {
		if j.cancelled() {
			j.setStage(StageCancelled, "已取消")
			return
		}
		j.setStage(StageNeedsCapture, fmt.Sprintf("%d 个库缺少可用材料，等待用户确认重启微信", len(pf.NeedsCapture)))
		if opts.Capture == nil {
			j.fail(fmt.Errorf("需要重新获取材料（%d 个库），但当前无可用采集通道；请先在目标微信登录后运行 setup 或提供材料", len(pf.NeedsCapture)))
			return
		}
		// Park until the user explicitly approves a WeChat restart or cancels.
		j.mu.Lock()
		approve, cancel := j.approve, j.cancel
		j.mu.Unlock()
		select {
		case <-approve:
		case <-cancel:
			j.setStage(StageCancelled, "用户在获取材料前取消")
			return
		}
		if j.cancelled() {
			j.setStage(StageCancelled, "已取消")
			return
		}
		j.setStage(StageCapturing, "开始按需获取材料（可能提示重启微信）")
		if err := opts.Capture(j.ctx, wxkey.SetupOptions{DBRoot: opts.DBRoot, Resolve: opts.Resolve, Restart: true,
			Recovery: j.recovery, Progress: func(msg string) {
				// A late diagnostic must not overwrite the recovery banner.
				if waiting, _ := j.recovery.State(); !waiting {
					j.setStage(StageCapturing, msg)
				}
			}}); err != nil {
			if j.cancelled() && !errors.Is(err, wxkey.ErrCaptureCleanup) {
				j.setStage(StageCancelled, "采集已取消")
				return
			}
			j.fail(fmt.Errorf("材料获取失败: %w", err))
			return
		}
		// Re-verify material after capture before exporting.
		pf2, err := RunPreflightContext(j.ctx, opts.DBRoot, opts.Resolve)
		if err != nil {
			j.fail(err)
			return
		}
		if !pf2.OfflineReady {
			j.fail(fmt.Errorf("采集后仍有 %d 个库不可读", len(pf2.NeedsCapture)))
			return
		}
		j.mu.Lock()
		j.preflight = pf2
		j.mu.Unlock()
	}

	j.setStage(StageExporting, "开始导出")
	rep, err := export.Run(export.Options{
		DBRoot:  opts.DBRoot,
		OutDir:  opts.OutDir,
		Resolve: opts.Resolve,
		Cancel:  j.cancel,
		OnProgress: func(p export.Progress) {
			j.setStage(StageExporting, fmt.Sprintf("[%s] %s (%d/%d)", p.Stage, p.Detail, p.Current, p.Total))
		},
	})
	j.mu.Lock()
	j.report = rep
	j.mu.Unlock()
	if err != nil {
		if rep != nil && rep.Status == "cancelled" {
			j.setStage(StageCancelled, "导出已取消")
			return
		}
		j.fail(err)
		return
	}
	switch rep.Status {
	case "complete":
		j.setStage(StageDone, fmt.Sprintf("导出完成：%d 条消息", rep.TotalMessages))
	case "partial":
		j.setStage(StageDone, fmt.Sprintf("部分完成：%d 条消息；详见 manifest", rep.TotalMessages))
	default:
		j.setStage(StageFailed, "导出失败")
	}
}

func (j *Job) fail(err error) {
	if errors.Is(err, context.Canceled) && !errors.Is(err, wxkey.ErrCaptureCleanup) {
		j.setStage(StageCancelled, "已取消")
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	j.err = err
	j.stage = StageFailed
	j.events = append(j.events, JobEvent{Time: time.Now(), Stage: StageFailed, Message: err.Error()})
}
