package export

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// Copy previous canonical JSONL into this job's private staging directory,
// validating the hash while copying. Never reread mutable output history
// between the indexing and carry-forward passes.
func stagePreviousJSONL(ctx context.Context, out, staging string) (string, error) {
	root, err := os.OpenRoot(out)
	if err != nil {
		return "", err
	}
	defer root.Close()
	st, err := readState(root)
	if err != nil {
		return "", err
	}
	path := filepath.Join(staging, "previous-messages.jsonl")
	rel := st.HistoryPath
	if rel == "" {
		rel = "data/messages.jsonl"
	}
	expected, owned := st.Files[rel]
	if !owned {
		if st.HistoryPath != "" {
			return "", errors.New("message history is not bound to publish state")
		}
		// A pre-existing, unowned file is not history. The publisher preserves
		// it and records our generated conflict sibling as HistoryPath.
		return path, nil
	}
	if err := safeRootPath(root, rel); err != nil {
		return "", err
	}
	in, err := root.Open(rel)
	if err != nil {
		return "", err
	}
	defer in.Close()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(io.MultiWriter(f, h), contextReader{ctx, in}); err != nil {
		return "", err
	}
	if hex.EncodeToString(h.Sum(nil)) != expected {
		return "", errors.New("published message history was modified; preserved without merging")
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return path, nil
}
