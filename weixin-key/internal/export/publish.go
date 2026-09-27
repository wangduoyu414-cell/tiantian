package export

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"weixin-key/internal/safefile"
)

// Publisher publishes generated files into the export directory with
// crash-safe, non-destructive semantics:
//
//   - every file lands via a temp sibling + fsync + atomic rename;
//   - files we generated before (recorded in state) are replaced in place;
//   - files the user modified since (hash differs from state) are preserved
//     and our new content lands in a "<name>.conflict-<ts>" sibling;
//   - pre-existing files we never generated are never overwritten;
//   - the publish state is written LAST, after all content files succeeded;
//     a crash mid-publish leaves the previous state untouched, so the next
//     run reconciles from evidence, not from a half-advanced cursor.
type Publisher struct {
	Root       string // export root directory
	AccountID  string
	Report     *Report
	Cursors    map[string]SnapshotInfo
	Cancelled  func() bool
	afterWrite func(int) error // deterministic failure/crash injection, tests only
}

// plannedFile is one file to publish: content staged at srcPath, destination
// rel inside Root. Staging on disk keeps memory bounded for large JSONL.
type plannedFile struct {
	rel     string // slash-separated path relative to Root
	srcPath string // staged content on disk
}

// publishState is .export-state/state.json: files we published + hashes.
type publishState struct {
	RunID       string                  `json:"run_id"`
	Updated     string                  `json:"updated"`
	Files       map[string]string       `json:"files"` // rel path -> sha256 hex
	AccountID   string                  `json:"account_id,omitempty"`
	HistoryPath string                  `json:"history_path,omitempty"`
	Cursors     map[string]SnapshotInfo `json:"cursors,omitempty"`
}

// PublishReport describes what Publish did.
type PublishReport struct {
	Written   []string `json:"written"`
	Conflicts []string `json:"conflicts,omitempty"`
	Skipped   []string `json:"skipped,omitempty"`
}

// Publish writes all planned files and updates the publish state last.
func (p *Publisher) Publish(runID string, files []plannedFile) (*PublishReport, error) {
	return p.publishTransaction(runID, files)
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return hashReader(f)
}

func hashReader(f io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// writeTempFile writes data to a synced temp file in dir and returns its path.
func writeTempFile(dir string, data []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, ".weixin-export-*")
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(name)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

// atomicCopy replaces dst with the content of src atomically (streamed
// temp+rename in the destination directory, so multi-GB files fit memory).
func atomicCopy(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".weixin-export-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		tmp.Close()
		if !ok {
			os.Remove(tmpName)
		}
	}()
	if _, err := io.Copy(tmp, in); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := safefile.Replace(tmpName, dst); err != nil {
		return err
	}
	ok = true
	return nil
}
