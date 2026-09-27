package export

import (
	"crypto/rand"
	"crypto/sha256"
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
)

const transactionPath = ".export-state/transaction.json"

type publishEntry struct {
	Rel     string `json:"rel"`
	NewHash string `json:"new_hash"`
	OldHash string `json:"old_hash,omitempty"`
	Backup  string `json:"backup,omitempty"`
	source  string
}
type publishTransaction struct {
	RunID     string         `json:"run_id"`
	BackupDir string         `json:"backup_dir"`
	Entries   []publishEntry `json:"entries"`
}

// safeRootPath rejects aliases inside the destination tree. os.Root also pins
// parent handles and prevents directory-swap/symlink escapes during operations.
func safeRootPath(root *os.Root, rel string) error {
	if !filepath.IsLocal(rel) || strings.Contains(rel, ":") {
		return fmt.Errorf("unsafe output path %q", rel)
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for i := range parts {
		p := strings.Join(parts[:i+1], "/")
		info, err := root.Lstat(p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("output path uses a link: %s", p)
		}
	}
	return nil
}

func contentPath(root *os.Root, rel string) error {
	if err := safeRootPath(root, rel); err != nil {
		return err
	}
	clean := filepath.ToSlash(filepath.Clean(rel))
	if rel != clean {
		return errors.New("publish paths must be normalized relative paths")
	}
	low := strings.ToLower(clean)
	if low == ".export-state" || strings.HasPrefix(low, ".export-state/") {
		return errors.New("reserved publish path")
	}
	return nil
}

func rootHash(root *os.Root, path string) (string, error) {
	f, err := root.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return hashReader(f)
}

func rootCopy(root *os.Root, dst string, in io.Reader) error {
	return rootCopyChecked(root, dst, in, "")
}

func rootCopyChecked(root *os.Root, dst string, in io.Reader, expected string) error {
	if err := safeRootPath(root, dst); err != nil {
		return err
	}
	if err := root.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	var nonce [12]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	tmp := filepath.Join(filepath.Dir(dst), ".weixin-export-"+hex.EncodeToString(nonce[:]))
	f, err := root.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer root.Remove(tmp)
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(io.MultiWriter(f, h), in); err != nil {
		return err
	}
	if expected != "" && hex.EncodeToString(h.Sum(nil)) != expected {
		return errors.New("staged content changed after publish preflight")
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	// os.Root.Rename replaces without deleting the destination first, and
	// uses pinned parent handles on Windows (no path-based rename race).
	return root.Rename(tmp, dst)
}
func rootJSON(root *os.Root, path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return rootCopy(root, path, strings.NewReader(string(b)))
}

func readState(root *os.Root) (*publishState, error) {
	if err := safeRootPath(root, ".export-state/state.json"); err != nil {
		return nil, err
	}
	st := &publishState{Files: map[string]string{}}
	b, err := root.ReadFile(".export-state/state.json")
	if os.IsNotExist(err) {
		return st, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, st); err != nil {
		return nil, fmt.Errorf("invalid publish state; refusing to discard history: %w", err)
	}
	if st.Files == nil {
		return nil, errors.New("publish state missing file hashes")
	}
	return st, nil
}

func checkAccount(root *os.Root, st *publishState, account string) error {
	if account == "" {
		return nil
	} // low-level Publisher helpers without export data
	owner := st.AccountID
	if owner == "" && len(st.Files) > 0 {
		// Legacy states may only be adopted from their hash-bound manifest.
		if err := safeRootPath(root, "manifest.json"); err != nil {
			return err
		}
		b, err := root.ReadFile("manifest.json")
		if err != nil {
			return errors.New("unbound output history; use a separate export directory")
		}
		if hash := fmt.Sprintf("%x", sha256.Sum256(b)); st.Files["manifest.json"] != hash {
			return errors.New("legacy manifest is not bound to published state")
		}
		var rep Report
		if err := json.Unmarshal(b, &rep); err != nil {
			return err
		}
		owner = rep.AccountID
		if owner == "" {
			return errors.New("legacy output account is unknown")
		}
	}
	if owner != "" && owner != account {
		return errors.New("output directory belongs to another account; select a separate export directory")
	}
	return nil
}

func validateOutputAccount(path, account string) error {
	root, err := os.OpenRoot(path)
	if err != nil {
		return err
	}
	defer root.Close()
	st, err := readState(root)
	if err != nil {
		return err
	}
	return checkAccount(root, st, account)
}

func (p *Publisher) openLocked() (root *os.Root, release func(), err error) {
	if p.Root == "" {
		return nil, nil, errors.New("empty publish root")
	}
	if err = os.MkdirAll(p.Root, 0o700); err != nil {
		return
	}
	root, err = os.OpenRoot(p.Root)
	if err != nil {
		return
	}
	if err = safeRootPath(root, ".export-state/publish.lock"); err != nil {
		root.Close()
		return nil, nil, err
	}
	unlock, err := joblock.Acquire(filepath.Join(p.Root, ".export-state", "publish.lock"))
	if err != nil {
		root.Close()
		return nil, nil, err
	}
	return root, func() { unlock(); root.Close() }, nil
}

// Recover runs before previous JSONL is consumed. An interrupted publish is
// either already committed (state.RunID) or rolled back from its journal.
func (p *Publisher) Recover() error {
	root, release, err := p.openLocked()
	if err != nil {
		return err
	}
	defer release()
	return recoverPublish(root)
}

func recoverPublish(root *os.Root) error {
	if err := safeRootPath(root, transactionPath); err != nil {
		return err
	}
	b, err := root.ReadFile(transactionPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var tx publishTransaction
	if err = json.Unmarshal(b, &tx); err != nil {
		return fmt.Errorf("invalid recovery journal: %w", err)
	}
	if tx.RunID == "" || sanitizeFileComponent(tx.RunID) != tx.RunID || tx.BackupDir != ".export-state/backup-"+tx.RunID {
		return errors.New("unsafe recovery transaction identity")
	}
	st, err := readState(root)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, e := range tx.Entries {
		if err := contentPath(root, e.Rel); err != nil {
			return err
		}
		key := strings.ToLower(e.Rel)
		if seen[key] {
			return errors.New("duplicate recovery target")
		}
		seen[key] = true
	}
	if st.RunID != tx.RunID {
		for i := len(tx.Entries) - 1; i >= 0; i-- {
			e := tx.Entries[i]
			if err := safeRootPath(root, e.Rel); err != nil {
				return err
			}
			got, err := rootHash(root, e.Rel)
			if os.IsNotExist(err) && e.OldHash == "" {
				continue
			}
			if err != nil {
				return fmt.Errorf("cannot safely recover %s: %w", e.Rel, err)
			}
			if got == e.OldHash {
				continue
			}
			if got != e.NewHash {
				return fmt.Errorf("output modified outside interrupted job: %s; preserved for manual reconciliation", e.Rel)
			}
			if e.OldHash == "" {
				if err := root.Remove(e.Rel); err != nil {
					return err
				}
			} else {
				expected := fmt.Sprintf("%s/%06d", tx.BackupDir, i)
				if e.Backup != expected {
					return errors.New("invalid recovery backup path")
				}
				if err := safeRootPath(root, e.Backup); err != nil {
					return err
				}
				sum, err := rootHash(root, e.Backup)
				if err != nil {
					return err
				}
				if sum != e.OldHash {
					return errors.New("recovery backup checksum mismatch")
				}
				f, err := root.Open(e.Backup)
				if err != nil {
					return err
				}
				err = rootCopy(root, e.Rel, f)
				f.Close()
				if err != nil {
					return err
				}
			}
		}
	}
	if err := safeRootPath(root, tx.BackupDir); err != nil {
		return err
	}
	if err := root.RemoveAll(tx.BackupDir); err != nil {
		return err
	}
	return root.Remove(transactionPath)
}

func (p *Publisher) publishTransaction(runID string, files []plannedFile) (rep *PublishReport, retErr error) {
	rep = &PublishReport{}
	root, release, err := p.openLocked()
	if err != nil {
		return rep, err
	}
	defer release()
	if err := recoverPublish(root); err != nil {
		return rep, err
	}
	st, err := readState(root)
	if err != nil {
		return rep, err
	}
	if err := checkAccount(root, st, p.AccountID); err != nil {
		return rep, err
	}
	if runID == "" || sanitizeFileComponent(runID) != runID || runID == st.RunID {
		return rep, errors.New("publish requires a new safe run ID")
	}
	tx := publishTransaction{RunID: runID, BackupDir: ".export-state/backup-" + runID}
	next := &publishState{RunID: runID, AccountID: st.AccountID, HistoryPath: st.HistoryPath, Updated: time.Now().Format(time.RFC3339Nano), Files: map[string]string{}, Cursors: p.Cursors}
	if p.AccountID != "" {
		next.AccountID = p.AccountID
	}
	for k, v := range st.Files {
		next.Files[k] = v
	}
	files = append([]plannedFile(nil), files...)
	sort.Slice(files, func(i, j int) bool {
		if files[i].rel == "manifest.json" {
			return false
		}
		if files[j].rel == "manifest.json" {
			return true
		}
		return files[i].rel < files[j].rel
	})
	seen := map[string]bool{}
	// Validate the ENTIRE plan before touching previously published files.
	for _, f := range files {
		if err := contentPath(root, f.rel); err != nil {
			return rep, err
		}
		pathKey := strings.ToLower(f.rel)
		if seen[pathKey] {
			return rep, errors.New("duplicate publish path")
		}
		seen[pathKey] = true
		sum, err := sha256File(f.srcPath)
		if err != nil {
			return rep, err
		}
		e := publishEntry{Rel: f.rel, NewHash: sum, source: f.srcPath}
		if info, err := root.Lstat(f.rel); err == nil {
			if !info.Mode().IsRegular() {
				return rep, fmt.Errorf("publish target not regular: %s", f.rel)
			}
			e.OldHash, err = rootHash(root, f.rel)
			if err != nil {
				return rep, err
			}
			if e.OldHash == sum && f.rel != "manifest.json" {
				rep.Skipped = append(rep.Skipped, f.rel)
				next.Files[f.rel] = sum
				if f.rel == "data/messages.jsonl" {
					next.HistoryPath = f.rel
				}
				continue
			}
			old, ours := st.Files[f.rel]
			if !ours || old != e.OldHash {
				e.Rel = f.rel + ".conflict-" + runID
				e.OldHash = ""
				if _, err := root.Lstat(e.Rel); !os.IsNotExist(err) {
					return rep, errors.New("conflict destination already exists or cannot be inspected")
				}
				rep.Conflicts = append(rep.Conflicts, f.rel+" -> "+e.Rel)
			}
		} else if !os.IsNotExist(err) {
			return rep, err
		}
		rep.Written = append(rep.Written, e.Rel)
		if f.rel == "data/messages.jsonl" {
			next.HistoryPath = e.Rel
		}
		tx.Entries = append(tx.Entries, e)
	}
	if p.Report != nil {
		p.Report.FilesWritten = append([]string(nil), rep.Written...)
		p.Report.Conflicts = append([]string(nil), rep.Conflicts...)
		if len(rep.Conflicts) > 0 && p.Report.Status == "complete" {
			p.Report.Status = "partial"
		}
		for i := range tx.Entries {
			e := &tx.Entries[i]
			if e.Rel == "manifest.json" || strings.HasPrefix(e.Rel, "manifest.json.conflict-") {
				b, err := json.MarshalIndent(p.Report, "", "  ")
				if err != nil {
					return rep, err
				}
				if err = os.WriteFile(e.source, b, 0o600); err != nil {
					return rep, err
				}
				e.NewHash, err = sha256File(e.source)
				if err != nil {
					return rep, err
				}
			}
		}
	}
	if err := root.MkdirAll(".export-state", 0o700); err != nil {
		return rep, err
	}
	if err := root.Mkdir(tx.BackupDir, 0o700); err != nil {
		return rep, err
	}
	for i := range tx.Entries {
		e := &tx.Entries[i]
		next.Files[e.Rel] = e.NewHash
		if e.OldHash == "" {
			continue
		}
		e.Backup = fmt.Sprintf("%s/%06d", tx.BackupDir, i)
		f, err := root.Open(e.Rel)
		if err != nil {
			return rep, err
		}
		err = rootCopy(root, e.Backup, f)
		f.Close()
		if err != nil {
			return rep, err
		}
		sum, err := rootHash(root, e.Backup)
		if err != nil || sum != e.OldHash {
			return rep, errors.New("output changed while saving recovery backup")
		}
	}
	if err := rootJSON(root, transactionPath, &tx); err != nil {
		return rep, err
	}
	// On a normal failure restore the old generation. A process crash leaves
	// the journal for Recover; a successfully committed state is never undone.
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, recoverPublish(root))
		}
	}()
	for i, e := range tx.Entries {
		if p.Cancelled != nil && p.Cancelled() {
			return rep, errors.New("publish cancelled")
		}
		current, err := rootHash(root, e.Rel)
		if e.OldHash == "" {
			if !os.IsNotExist(err) {
				return rep, errors.New("new publish target appeared unexpectedly")
			}
		} else if err != nil || current != e.OldHash {
			return rep, errors.New("publish target changed since preflight")
		}
		f, err := os.Open(e.source)
		if err != nil {
			return rep, err
		}
		err = rootCopyChecked(root, e.Rel, f, e.NewHash)
		f.Close()
		if err != nil {
			return rep, err
		}
		if p.afterWrite != nil {
			if err := p.afterWrite(i); err != nil {
				return rep, err
			}
		}
	}
	if p.Cancelled != nil && p.Cancelled() {
		return rep, errors.New("publish cancelled")
	}
	// Content, manifest and cursors have ONE final, atomic commit point.
	if err := rootJSON(root, ".export-state/state.json", next); err != nil {
		return rep, err
	}
	if err := recoverPublish(root); err != nil {
		return rep, fmt.Errorf("committed, but recovery-file cleanup failed: %w", err)
	}
	return rep, nil
}
