package export

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"weixin-key/internal/joblock"

	"weixin-key/internal/config"
	"weixin-key/internal/pathguard"
	"weixin-key/internal/wxkey"
)

// Options controls one export run.
type Options struct {
	// DBRoot is the account directory that contains db_storage. Empty means
	// config db_root, then platform auto-detect.
	DBRoot string
	// OutDir is the export output directory (created when missing).
	OutDir string
	// AccountID overrides the account identity; defaults to the DBRoot dir name.
	AccountID string
	// Resolve carries explicit key material (CLI flags). The config cache is
	// consulted by default; Resolve.NoConfig disables that.
	Resolve wxkey.ResolveOptions
	// OnProgress receives stage transitions for UI progress display. It must
	// be fast and non-blocking; nil disables reporting.
	OnProgress func(Progress)
	// Cancel, when closed or when it returns true via PollCancel, aborts the
	// run between DBs with a "cancelled" report status.
	Cancel <-chan struct{}
	// Full disables incremental merging (previous messages.jsonl is ignored;
	// everything renders fresh). Default is incremental.
	Full bool
	// ImgKeyHex is the optional WeChat v4 image AES-128 key (hex). With it,
	// .dat image attachments are decoded to plain images; without it they are
	// copied as stored-encrypted evidence.
	ImgKeyHex string
}

// Progress is one stage transition or heartbeat.
type Progress struct {
	Stage   string `json:"stage"`   // validate|discover|snapshot|parse|render|publish
	Detail  string `json:"detail"`  // human-readable, no secrets
	Current int    `json:"current"` // DBs processed
	Total   int    `json:"total"`   // DBs discovered (0 = indeterminate)
}

// SnapshotInfo records one DB's snapshot evidence.
type SnapshotInfo struct {
	Method           string `json:"method"`
	SourceSize       int64  `json:"source_size"`
	SourceModTime    string `json:"source_mod_time"`
	HadWAL           bool   `json:"had_wal"`
	WALSize          int64  `json:"wal_size,omitempty"`
	MainPages        int    `json:"main_pages,omitempty"`
	WALFramesApplied int    `json:"wal_frames_applied,omitempty"`
	WALNote          string `json:"wal_note,omitempty"`
	SnapshotTime     string `json:"snapshot_time"`
}

// DBResult is one source DB's outcome.
type DBResult struct {
	Path            string         `json:"path"` // relative to db_storage
	Encrypted       bool           `json:"encrypted"`
	KeySource       string         `json:"key_source,omitempty"`
	Snapshot        *SnapshotInfo  `json:"snapshot,omitempty"`
	Messages        int64          `json:"messages"`
	PartialMessages int64          `json:"partial_messages,omitempty"`
	SkippedTables   []SkippedTable `json:"skipped_tables,omitempty"`
	Warnings        []string       `json:"warnings,omitempty"`
	Error           string         `json:"error,omitempty"`
}

// Report is the machine-readable export manifest (manifest.json).
type Report struct {
	progressFn         func(Progress)  `json:"-"`
	cancelCh           <-chan struct{} `json:"-"`
	RunID              string          `json:"run_id"`
	Schema             string          `json:"schema"`
	Tool               string          `json:"tool"`
	Status             string          `json:"status"` // complete | partial | failed
	Started            string          `json:"started"`
	Finished           string          `json:"finished"`
	AccountID          string          `json:"account_id"`
	DBRoot             string          `json:"db_root"`
	OutDir             string          `json:"out_dir"`
	DBs                []DBResult      `json:"dbs"`
	TotalMessages      int64           `json:"total_messages"`
	TotalConversations int             `json:"total_conversations"`
	FilesWritten       []string        `json:"files_written"`
	Conflicts          []string        `json:"conflicts,omitempty"`
	Merge              *MergeStats     `json:"merge,omitempty"`
}

// progress reports a stage transition when a listener is attached.
func (rep *Report) progress(p Progress) {
	if rep.progressFn != nil {
		rep.progressFn(p)
	}
}

// cancelled reports whether the run's cancel channel fired.
func (rep *Report) cancelled() bool {
	if rep.cancelCh == nil {
		return false
	}
	select {
	case <-rep.cancelCh:
		return true
	default:
		return false
	}
}

// Run executes one export: discover DBs, snapshot+decrypt, parse, render,
// publish. Source files are opened strictly read-only and never modified;
// snapshot/decrypt work happens on staged copies.
func Run(opts Options) (*Report, error) {
	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()
	if opts.Cancel != nil {
		go func() {
			select {
			case <-opts.Cancel:
				cancelRun()
			case <-runCtx.Done():
			}
		}()
	}
	rep := &Report{
		RunID:   newRunID(),
		Schema:  SchemaVersion,
		Tool:    "weixin-key",
		Started: time.Now().Format(time.RFC3339),
	}
	rep.progressFn = opts.OnProgress
	rep.cancelCh = opts.Cancel
	fail := func(err error) (*Report, error) {
		rep.Status = "failed"
		if rep.cancelled() || errors.Is(err, context.Canceled) {
			rep.Status = "cancelled"
		}
		rep.Finished = time.Now().Format(time.RFC3339)
		return rep, err
	}

	dbRoot := opts.DBRoot
	if dbRoot == "" {
		cfg, err := config.Load()
		if err == nil && cfg.DBRoot != "" {
			dbRoot = cfg.DBRoot
		} else {
			root, _, derr := config.AutoDetectDBRoot()
			if derr != nil {
				return fail(fmt.Errorf("locate DB root: %w", derr))
			}
			dbRoot = root
		}
	}
	if opts.OutDir == "" {
		return fail(errors.New("export: --out is required"))
	}
	outAbs, err := pathguard.ResolveOutput(opts.OutDir, dbRoot)
	if err != nil {
		return fail(err)
	}
	rep.DBRoot = dbRoot
	rep.OutDir = outAbs
	if err := os.MkdirAll(outAbs, 0o700); err != nil {
		return fail(err)
	}
	rootGuard, err := os.OpenRoot(outAbs)
	if err != nil {
		return fail(err)
	}
	err = safeRootPath(rootGuard, ".export-state/export.lock")
	rootGuard.Close()
	if err != nil {
		return fail(err)
	}
	unlock, err := joblock.Acquire(filepath.Join(outAbs, ".export-state", "export.lock"))
	if err != nil {
		return fail(err)
	}
	defer unlock()
	if err := (&Publisher{Root: outAbs}).Recover(); err != nil {
		return fail(err)
	}

	account := opts.AccountID
	if account == "" {
		account = accountIDFromDir(dbRoot)
	}
	rep.AccountID = account
	if err := validateOutputAccount(outAbs, account); err != nil {
		return fail(err)
	}

	storage := filepath.Join(dbRoot, "db_storage")
	rep.progress(Progress{Stage: "discover", Detail: storage})
	dbs, err := discoverDBs(storage)
	if err != nil {
		return fail(err)
	}
	if len(dbs) == 0 {
		return fail(fmt.Errorf("no .db files under %s", storage))
	}
	rep.progress(Progress{Stage: "discover", Detail: fmt.Sprintf("%d databases", len(dbs)), Total: len(dbs)})
	if err := checkExportSpace(outAbs, dbs); err != nil {
		return fail(err)
	}

	// Staging lives under .export-state with owner-only permissions; on
	// Windows its DACL is private at CreateDirectory time and verified by
	// reading the resulting security descriptor. Plaintext snapshots exist
	// only for the duration of the run and are wiped on return.
	staging, cleanupStaging, err := createPrivateStaging(filepath.Join(outAbs, ".export-state"), rep.RunID)
	if err != nil {
		return fail(err)
	}
	defer func() { _ = cleanupStaging() }()

	filesDir := filepath.Join(staging, "files")
	if err := os.MkdirAll(filesDir, 0o700); err != nil {
		return fail(err)
	}

	resolver := wxkey.NewKeyResolver()

	sinks, err := newRenderSinks(filesDir, account)
	if err != nil {
		return fail(err)
	}
	defer sinks.Close()
	sinks.cancelled = rep.cancelled

	ctx := &parseContext{
		context:      runCtx,
		accountID:    account,
		snapshotID:   rep.RunID,
		contacts:     map[string]string{},
		convNames:    map[string]string{},
		convKinds:    map[string]string{},
		ownSenderIDs: map[string]bool{},
	}

	// Stage + decrypt every DB first, so contact/session catalogs from one DB
	// enrich the message parsing of the others regardless of file order.
	if rep.cancelled() {
		rep.Status = "cancelled"
		rep.Finished = time.Now().Format(time.RFC3339)
		return rep, errors.New("export cancelled")
	}
	type stagedDB struct {
		src   sourceDB
		plain string
		idx   int
	}
	var staged []stagedDB
	for i, db := range dbs {
		if rep.cancelled() {
			rep.Status = "cancelled"
			rep.Finished = time.Now().Format(time.RFC3339)
			return rep, errors.New("export cancelled")
		}
		rep.progress(Progress{Stage: "snapshot", Detail: db.rel, Current: i, Total: len(dbs)})
		dres := DBResult{Path: db.rel, Encrypted: !isPlaintextSQLite(db.path)}
		plain, sinfo, ksrc, warnings, err := snapshotDBContext(runCtx, db, staging, resolver, opts.Resolve)
		dres.Snapshot = sinfo
		dres.KeySource = ksrc
		dres.Warnings = warnings
		if err != nil {
			dres.Error = err.Error()
		}
		rep.DBs = append(rep.DBs, dres)
		if err == nil {
			staged = append(staged, stagedDB{src: db, plain: plain, idx: len(rep.DBs) - 1})
		}
	}

	// Pass 1: contact + session catalogs only (shared identity context).
	for _, s := range staged {
		if rep.cancelled() {
			return fail(context.Canceled)
		}
		w, err := enrichContext(s.plain, s.src.rel, ctx)
		rep.DBs[s.idx].Warnings = append(rep.DBs[s.idx].Warnings, w...)
		if err != nil {
			rep.DBs[s.idx].Warnings = append(rep.DBs[s.idx].Warnings, "catalog probe: "+err.Error())
		}
	}

	// Attachment extraction: voice payloads live in media DB snapshots, and
	// resource sizes (for linking on-disk attachment files) come from
	// message_resource-style DBs. Both were staged+decrypted above.
	att := &attacher{dbRoot: dbRoot, filesDir: filesDir, convUser: ctx.convUsers,
		voice: map[string][]byte{}, resSize: map[string][]int64{}}
	if opts.ImgKeyHex != "" {
		k, err := hex.DecodeString(opts.ImgKeyHex)
		if err != nil || len(k) != 16 {
			return fail(fmt.Errorf("--img-key must be 16 bytes hex"))
		}
		att.imgKey = k
	}
	for _, s := range staged {
		if rep.cancelled() {
			return fail(context.Canceled)
		}
		db, err := openSnapshot(s.plain)
		if err != nil {
			continue
		}
		if v, w := probeVoice(db); len(v) > 0 {
			for k, data := range v {
				att.voice[k] = data
			}
			rep.DBs[s.idx].Warnings = append(rep.DBs[s.idx].Warnings, w...)
		}
		if r, w := probeResourceSizes(db); len(r) > 0 {
			for k, sizes := range r {
				att.resSize[k] = sizes
			}
			rep.DBs[s.idx].Warnings = append(rep.DBs[s.idx].Warnings, w...)
		}
		db.Close()
	}
	ctx.attach = att

	// Pass 2: stream messages through the incremental merger.
	prevJSONL := ""
	if !opts.Full {
		prevJSONL, err = stagePreviousJSONL(runCtx, outAbs, staging)
		if err != nil {
			return fail(err)
		}
	}
	m, err := newMerger(runCtx, prevJSONL, !opts.Full, account)
	if err != nil {
		return fail(err)
	}
	for i, s := range staged {
		rep.progress(Progress{Stage: "parse", Detail: s.src.rel, Current: i, Total: len(staged)})
		if rep.cancelled() {
			return fail(context.Canceled)
		}
		count, partial, skipped, w2, err := streamDBMessages(s.plain, s.src, ctx, m, sinks)
		rep.DBs[s.idx].Messages = count
		rep.DBs[s.idx].PartialMessages = partial
		rep.DBs[s.idx].SkippedTables = skipped
		rep.DBs[s.idx].Warnings = append(rep.DBs[s.idx].Warnings, w2...)
		if err != nil {
			rep.DBs[s.idx].Error = err.Error()
		}
	}

	if rep.cancelled() {
		return fail(context.Canceled)
	}

	// Carry forward previously exported messages that the current source no
	// longer contains (never auto-delete history).
	if err := m.flushCarried(prevJSONL, sinks.emit); err != nil {
		return fail(err)
	}
	m.stats.Dupes = m.droppedDupes
	rep.Merge = &m.stats

	if err := sinks.Close(); err != nil {
		return fail(err)
	}

	// Markdown volumes render from the merged JSONL (post-merge), so
	// carried-forward history lands in correct time order.
	if err := renderVolumesFromJSONLContext(runCtx, sinks.jsonlPath(), filesDir, account, sinks.conversations); err != nil {
		return fail(fmt.Errorf("render volumes: %w", err))
	}

	if _, err := writeConversationsFile(filesDir, sinks); err != nil {
		return fail(err)
	}
	if _, err := writeContactsFile(filesDir, ctx.contacts); err != nil {
		return fail(err)
	}

	rep.TotalConversations = len(sinks.conversations)
	rep.TotalMessages = sinks.totalMessages

	status := "complete"
	for _, d := range rep.DBs {
		if d.Error != "" || d.PartialMessages > 0 ||
			(d.Snapshot != nil && d.Snapshot.WALNote != "") {
			status = "partial"
			break
		}
	}
	if rep.TotalMessages == 0 {
		allErr := len(rep.DBs) > 0
		for _, d := range rep.DBs {
			if d.Error == "" {
				allErr = false
			}
		}
		if allErr {
			status = "failed"
		}
	}
	rep.Status = status
	rep.Finished = time.Now().Format(time.RFC3339)

	manifestPath := filepath.Join(filesDir, "manifest.json")
	manifestBytes, _ := json.MarshalIndent(rep, "", "  ")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o600); err != nil {
		return fail(err)
	}
	indexPath := filepath.Join(filesDir, "index.md")
	if err := os.WriteFile(indexPath, []byte(renderIndex(rep, sinks)), 0o600); err != nil {
		return fail(err)
	}

	// Publish everything staged under filesDir into OutDir, safely.
	var plan []plannedFile
	err = filepath.WalkDir(filesDir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(filesDir, p)
		if err != nil {
			return err
		}
		plan = append(plan, plannedFile{rel: filepath.ToSlash(rel), srcPath: p})
		return nil
	})
	if err != nil {
		return fail(err)
	}
	rep.progress(Progress{Stage: "publish", Detail: outAbs})
	if rep.cancelled() {
		return fail(context.Canceled)
	}
	cursors := map[string]SnapshotInfo{}
	for _, d := range rep.DBs {
		if d.Snapshot != nil && d.Error == "" && d.Snapshot.WALNote == "" {
			cursors[d.Path] = *d.Snapshot
		}
	}
	pub := &Publisher{Root: outAbs, AccountID: account, Report: rep, Cursors: cursors, Cancelled: rep.cancelled}
	if _, err := pub.Publish(rep.RunID, plan); err != nil {
		return fail(fmt.Errorf("publish: %w", err))
	}
	return rep, nil
}

// ---------------------------------------------------------------------------
// discovery + snapshot
// ---------------------------------------------------------------------------

type sourceDB struct {
	rel  string // relative to db_storage
	path string
}

func discoverDBs(storage string) ([]sourceDB, error) {
	var out []sourceDB
	err := filepath.WalkDir(storage, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("source database tree contains a link: %s", p)
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".db") {
			return nil
		}
		rel, _ := filepath.Rel(storage, p)
		out = append(out, sourceDB{rel: rel, path: p})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].rel < out[j].rel })
	return out, err
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

// copyFileReadOnly streams src to a fresh dst, opening src strictly read-only.
func copyFileReadOnly(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	return out.Close()
}

// sameOrChild reports whether out equals src or lives under src.
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

func accountIDFromDir(root string) string {
	name := filepath.Base(filepath.Clean(root))
	if idx := strings.LastIndex(name, "_"); idx > 0 {
		return name[:idx]
	}
	return name
}

func newRunID() string {
	now := time.Now()
	return fmt.Sprintf("%s-%09d", now.Format("20060102-150405"), now.UnixNano()%1e9)
}
