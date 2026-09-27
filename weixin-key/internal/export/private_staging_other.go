//go:build !windows

package export

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

const exportStagingMargin int64 = 512 << 20

// createPrivateStaging creates a run-owned directory and returns a cleanup
// function. On non-Windows systems chmod is the strongest portable primitive
// available here; callers must not treat it as a cross-platform DACL proof.
func createPrivateStaging(parent, runID string) (string, func() error, error) {
	dir := filepath.Join(parent, "staging-"+sanitizeFileComponent(runID))
	if err := os.Mkdir(dir, 0o700); err != nil {
		return "", nil, err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		_ = os.Remove(dir)
		return "", nil, err
	}
	cleanup := func() error { return os.RemoveAll(dir) }
	return dir, cleanup, nil
}

func availableBytes(path string) (int64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}
