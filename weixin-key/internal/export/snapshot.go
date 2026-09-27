package export

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"modernc.org/sqlite"
	"weixin-key/internal/sqliteengine"
	"weixin-key/internal/wcdb"
	"weixin-key/internal/wxkey"
)

var ErrLiveEncryptedSnapshot = errors.New("consistent encrypted snapshot unavailable while source may be written")

func sqliteURI(path, mode string) string {
	p := filepath.ToSlash(path)
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	u := url.URL{Scheme: "file", Path: p}
	q := url.Values{}
	q.Set("mode", mode)
	q.Add("_pragma", "busy_timeout(100)")
	u.RawQuery = q.Encode()
	return u.String()
}

func backupPlainSQLite(ctx context.Context, src, dst string) (retErr error) {
	db, err := sql.Open("sqlite", sqliteURI(src, "ro"))
	if err != nil {
		return err
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	// Pin one source read transaction for the full backup, including WAL.
	if _, err = conn.ExecContext(ctx, "BEGIN"); err != nil {
		return err
	}
	defer conn.ExecContext(context.Background(), "ROLLBACK")
	var count int64
	if err = conn.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master").Scan(&count); err != nil {
		return err
	}
	return conn.Raw(func(driverConn any) error {
		creator, ok := driverConn.(interface {
			NewBackup(string) (*sqlite.Backup, error)
		})
		if !ok {
			return errors.New("SQLite driver has no online backup support")
		}
		backup, err := creator.NewBackup(sqliteURI(dst, "rwc"))
		if err != nil {
			return err
		}
		finished := false
		defer func() {
			if !finished {
				_ = backup.Finish()
			}
		}()
		deadline := time.Now().Add(2 * time.Minute)
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			if time.Now().After(deadline) {
				return errors.New("SQLite backup timed out")
			}
			more, err := backup.Step(128)
			if err != nil {
				var sqliteErr *sqlite.Error
				if errors.As(err, &sqliteErr) && (sqliteErr.Code()&0xff == 5 || sqliteErr.Code()&0xff == 6) {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(20 * time.Millisecond):
					}
					continue
				}
				return err
			}
			if !more {
				finished = true
				return backup.Finish()
			}
		}
	})
}

// snapshotDB preserves the package's test/helper entry point.
func snapshotDB(db sourceDB, staging string, resolver *wxkey.KeyResolver, ropts wxkey.ResolveOptions) (string, *SnapshotInfo, string, []string, error) {
	return snapshotDBContext(context.Background(), db, staging, resolver, ropts)
}

func snapshotDBContext(ctx context.Context, db sourceDB, staging string, resolver *wxkey.KeyResolver, ropts wxkey.ResolveOptions) (string, *SnapshotInfo, string, []string, error) {
	if err := ctx.Err(); err != nil {
		return "", nil, "", nil, err
	}
	srcInfo, err := os.Stat(db.path)
	if err != nil {
		return "", nil, "", nil, err
	}
	info := &SnapshotInfo{SourceSize: srcInfo.Size(), SourceModTime: srcInfo.ModTime().Format(time.RFC3339Nano), SnapshotTime: time.Now().Format(time.RFC3339Nano)}
	snapDir := filepath.Join(staging, "snap")
	if err := os.MkdirAll(snapDir, 0o700); err != nil {
		return "", info, "", nil, err
	}
	stageDB := filepath.Join(snapDir, sanitizeFileComponent(db.rel)+".db")
	if wi, err := os.Stat(db.path + "-wal"); err == nil {
		info.HadWAL = wi.Size() > 0
		info.WALSize = wi.Size()
	} else if !os.IsNotExist(err) {
		return "", info, "", nil, err
	}
	if isPlaintextSQLite(db.path) {
		info.Method = "sqlite-online-backup-read-transaction"
		if err := backupPlainSQLite(ctx, db.path, stageDB); err != nil {
			return "", info, "", nil, err
		}
		current, err := os.Stat(db.path)
		if err != nil || !os.SameFile(srcInfo, current) {
			return "", info, "", nil, errors.New("source identity changed during SQLite backup")
		}
		return stageDB, info, "plaintext", nil, nil
	}

	// Prefer the validated engine's pinned read transaction, including live
	// WAL. A real engine error must never silently fall back to a raw copy.
	if _, err := wcdb.DetectProfile(db.path); err != nil {
		return "", info, "", nil, err
	}
	res, err := resolver.ResolveForDB(db.path, ropts)
	if err != nil {
		return "", info, "", nil, err
	}
	plain := stageDB + ".plain.db"
	err = sqliteengine.Backup(ctx, db.path, plain, res.EncKeyHex, res.SaltHex)
	if err == nil {
		info.Method = "sqlite3mc-online-backup-read-transaction/" + sqliteengine.Version
		if fi, e := os.Stat(plain); e == nil {
			info.MainPages = int(fi.Size() / wcdb.SQLCipherPageSize)
		}
		return plain, info, res.Source, nil, nil
	}
	if !errors.Is(err, sqliteengine.ErrUnavailable) {
		return "", info, res.Source, nil, err
	}

	// A byte-copy is permitted only while the OS prevents writers and rename.
	// This is NOT an online encrypted-engine snapshot. Unsupported platforms
	// and active sources fail closed rather than pretending to be consistent.
	main, err := openFrozenSource(db.path)
	if err != nil {
		return "", info, "", nil, err
	}
	defer main.Close()
	if journal, err := os.Stat(db.path + "-journal"); err == nil && journal.Size() > 0 {
		return "", info, "", nil, errors.New("source has a rollback journal requiring engine recovery; refusing raw snapshot")
	} else if err != nil && !os.IsNotExist(err) {
		return "", info, "", nil, err
	}
	var wal *os.File
	wal, err = openFrozenSource(db.path + "-wal")
	if err != nil && !os.IsNotExist(err) {
		return "", info, "", nil, err
	}
	if wal != nil {
		defer wal.Close()
	}
	if err := copyOpenSource(ctx, main, stageDB); err != nil {
		return "", info, "", nil, err
	}
	stageWal := ""
	if wal != nil {
		wi, err := wal.Stat()
		if err != nil {
			return "", info, "", nil, err
		}
		info.WALSize = wi.Size()
		info.HadWAL = wi.Size() > 0
		if wi.Size() > 0 {
			stageWal = stageDB + "-wal"
			if err := copyOpenSource(ctx, wal, stageWal); err != nil {
				return "", info, "", nil, err
			}
		}
	}
	info.Method = "windows-deny-write-file-snapshot"
	// Resolve against the frozen COPY, not a separately reopened live DB.
	if _, err := wcdb.DetectProfile(stageDB); err != nil {
		return "", info, "", nil, err
	}
	res, err = resolver.ResolveForDB(stageDB, ropts)
	if err != nil {
		return "", info, "", nil, err
	}
	key, err := hex.DecodeString(res.EncKeyHex)
	if err != nil {
		return "", info, "", nil, err
	}
	defer clear(key)
	report, err := wcdb.DecryptDBWithWALContext(ctx, stageDB, stageWal, plain, key)
	if err != nil {
		return "", info, "", nil, err
	}
	info.MainPages = report.MainPages
	info.WALFramesApplied = report.WALFramesApplied
	info.WALNote = report.Note
	var warnings []string
	if report.Note != "" {
		warnings = append(warnings, report.Note)
	}
	return plain, info, res.Source, warnings, nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(b []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(b)
}

func copyOpenSource(ctx context.Context, src *os.File, dst string) error {
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err = io.Copy(out, contextReader{ctx, src}); err != nil {
		return err
	}
	if err = out.Sync(); err != nil {
		return fmt.Errorf("sync snapshot: %w", err)
	}
	return out.Close()
}
