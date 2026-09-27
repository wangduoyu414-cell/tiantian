//go:build windows

package wxkey

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

const maxNativeCaptureFileSize = 1 << 20
const maxNativeCaptureCandidates = 64

// windowsNativeCaptureScan imports raw 32-byte keys collected by the injected
// capture DLL. The file is an intentionally narrow hand-off: one hex key per
// line, no database paths or account metadata. Every candidate is verified
// against the live database headers before it can be persisted.
func windowsNativeCaptureScan(dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) (uint32, error) {
	path := firstEnv("WECHAT_CLI_CAPTURE_KEY_FILE", "WX_MCP_CAPTURE_KEY_FILE")
	if path == "" {
		return 0, nil
	}
	candidates, err := windowsReadNativeCaptureFile(path)
	if err != nil {
		return 0, fmt.Errorf("native capture file: %w", err)
	}
	scan.addDiag("RouteN: loaded %d candidate(s) from injected capture", len(candidates))
	if err := windowsVerifyCandidateKeys(candidates, dbs, salts, scan, deadline); err != nil {
		return 0, err
	}
	if len(scan.found) == 0 {
		return 0, fmt.Errorf("native capture file contained %d candidate(s), but none verified against the selected account", len(candidates))
	}
	pid, err := windowsNativeCapturePID()
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func windowsReadNativeCaptureFile(path string) ([]string, error) {
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 || !pathInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("capture path must be a regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	currentPathInfo, err := os.Lstat(path)
	if err != nil || currentPathInfo.Mode()&os.ModeSymlink != 0 ||
		!info.Mode().IsRegular() || !os.SameFile(pathInfo, info) ||
		!os.SameFile(currentPathInfo, info) {
		return nil, fmt.Errorf("capture path changed during validation")
	}
	if info.Size() <= 0 || info.Size() > maxNativeCaptureFileSize {
		return nil, fmt.Errorf("capture file size %d is outside the allowed range", info.Size())
	}
	b, err := io.ReadAll(io.LimitReader(file, maxNativeCaptureFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxNativeCaptureFileSize {
		return nil, fmt.Errorf("capture file exceeds the allowed size")
	}
	seen := map[string]bool{}
	out := make([]string, 0, 16)
	for _, line := range strings.Split(string(b), "\n") {
		candidate := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(line, "\r")))
		if len(candidate) != 64 || seen[candidate] {
			continue
		}
		decoded, err := hex.DecodeString(candidate)
		if err != nil || len(decoded) != 32 {
			continue
		}
		seen[candidate] = true
		out = append(out, candidate)
		if len(out) > maxNativeCaptureCandidates {
			return nil, fmt.Errorf("capture file contains more than %d unique candidates", maxNativeCaptureCandidates)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("capture file contains no valid 32-byte hex candidates")
	}
	return out, nil
}

func windowsNativeCapturePID() (uint32, error) {
	raw := firstEnv("WECHAT_CLI_CAPTURE_PID", "WX_MCP_CAPTURE_PID")
	if raw == "" {
		return 0, nil
	}
	pid, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || pid == 0 {
		return 0, fmt.Errorf("native capture PID must be a positive 32-bit integer")
	}
	return uint32(pid), nil
}
