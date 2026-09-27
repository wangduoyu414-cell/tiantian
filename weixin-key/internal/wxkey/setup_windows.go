//go:build windows

package wxkey

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"weixin-key/internal/wcdb"
)

const (
	processVMRead           = 0x0010
	processQueryInformation = 0x0400

	memCommit    = 0x1000
	pageNoAccess = 0x01
	pageGuard    = 0x100

	th32csSnapProcess = 0x00000002
	th32csSnapModule  = 0x00000008 // enumerate modules for a process
)

var errWindowsKeyScanDeadline = errors.New("windows key scan deadline exceeded")

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procOpenProcess         = kernel32.NewProc("OpenProcess")
	procCloseHandle         = kernel32.NewProc("CloseHandle")
	procVirtualQueryEx      = kernel32.NewProc("VirtualQueryEx")
	procReadProcessMemory   = kernel32.NewProc("ReadProcessMemory")
	procCreateToolhelp32    = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW     = kernel32.NewProc("Process32FirstW")
	procProcess32NextW      = kernel32.NewProc("Process32NextW")
	procGetCurrentProcessID = kernel32.NewProc("GetCurrentProcessId")
	procModule32FirstW      = kernel32.NewProc("Module32FirstW")
	procModule32NextW       = kernel32.NewProc("Module32NextW")
)

type windowsMemoryBasicInformation struct {
	BaseAddress       uintptr
	AllocationBase    uintptr
	AllocationProtect uint32
	_                 uint32
	RegionSize        uintptr
	State             uint32
	Protect           uint32
	Type              uint32
	_                 uint32
}

type windowsProcessEntry32 struct {
	Size            uint32
	CntUsage        uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	CntThreads      uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

// windowsModuleEntry32W mirrors MODULEENTRY32W (64-bit layout).
// Field layout on 64-bit Windows (with 4-byte padding before each pointer):
//
//	offset 0:  Size         (4)
//	offset 4:  ModuleID     (4)
//	offset 8:  ProcessID    (4)
//	offset 12: GlblCntUsage (4)
//	offset 16: ProcCntUsage (4)
//	offset 20: _pad0        (4) — align ModBaseAddr on 8-byte boundary
//	offset 24: ModBaseAddr  (8)
//	offset 32: ModBaseSize  (4)
//	offset 36: _pad1        (4) — align Module on 8-byte boundary
//	offset 40: Module       (8)
//	offset 48: ModuleName   [256]uint16 (512)
//	offset 560: ExePath     [260]uint16 (520)
//	total: 1080 bytes
type windowsModuleEntry32W struct {
	Size         uint32
	ModuleID     uint32
	ProcessID    uint32
	GlblCntUsage uint32
	ProcCntUsage uint32
	_            uint32  // padding
	ModBaseAddr  uintptr // base address of module in the owner process
	ModBaseSize  uint32  // size of the module in bytes
	_            uint32  // padding
	Module       uintptr // handle to the module
	ModuleName   [256]uint16
	ExePath      [260]uint16
}

type windowsSourceDB struct {
	rel  string
	path string
	salt string
}

type windowsProcess struct {
	pid uint32
	exe string
}

type windowsSetupStats struct {
	SourceDBs        int                `json:"source_dbs"`
	TargetSalts      int                `json:"target_salts"`
	ScannedProcesses int                `json:"scanned_processes"`
	ScannedPIDs      []uint32           `json:"scanned_pids"`
	MatchedSalts     int                `json:"matched_salts"`
	VerifiedDBs      int                `json:"verified_dbs"`
	Routes           []windowsRouteStat `json:"routes,omitempty"`
}

func windowsKeyScanTimeout() time.Duration {
	raw := firstEnv("WECHAT_CLI_KEY_SCAN_TIMEOUT", "WECHAT_CLI_WINDOWS_KEY_SCAN_TIMEOUT", "WX_MCP_KEY_SCAN_TIMEOUT")
	if raw == "" {
		return 3 * time.Minute
	}
	if d, err := time.ParseDuration(raw); err == nil {
		if d < 0 {
			return 0
		}
		return d
	}
	if sec, err := strconv.Atoi(raw); err == nil {
		if sec < 0 {
			return 0
		}
		return time.Duration(sec) * time.Second
	}
	return 3 * time.Minute
}

func windowsKeyScanDeadlineExceeded(deadline time.Time) bool {
	return !deadline.IsZero() && time.Now().After(deadline)
}

func windowsFindWCDB() (string, error) {
	names := []string{"libWCDB.dll", "WCDB.dll", "e_sqlcipher.dll"}
	var candidates []string
	for _, env := range []string{"WECHAT_CLI_WCDB_LIB", "WECHAT_CLI_WCDB_DYLIB", "WX_MCP_WCDB_LIB", "WX_MCP_WCDB_DYLIB"} {
		if p := strings.TrimSpace(os.Getenv(env)); p != "" {
			candidates = append(candidates, p)
		}
	}
	if exe, err := os.Executable(); err == nil {
		if exe, err = filepath.EvalSymlinks(exe); err == nil {
			dir := filepath.Dir(exe)
			for _, name := range names {
				candidates = append(candidates, filepath.Join(dir, name), filepath.Join(dir, "lib", name), filepath.Join(dir, "..", "lib", name))
			}
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		for _, name := range names {
			candidates = append(candidates, filepath.Join(cwd, name), filepath.Join(cwd, "lib", name))
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		for _, name := range names {
			candidates = append(candidates, filepath.Join(home, ".config", "wxcli", "lib", name))
		}
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("WCDB/SQLCipher DLL not found for Windows key verification")
}

func windowsListSourceDBs(root string) ([]windowsSourceDB, map[string]bool, error) {
	base := filepath.Join(root, "db_storage")
	var out []windowsSourceDB
	salts := map[string]bool{}
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("discover required databases: %w", err)
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("source database tree contains a link: %s", path)
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.EqualFold(filepath.Ext(name), ".db") {
			return nil
		}
		salt, err := windowsReadSaltHex(path)
		if err != nil {
			return fmt.Errorf("read required database %s: %w", path, err)
		}
		if salt == hex.EncodeToString([]byte("SQLite format 3\x00")) {
			return nil
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		out = append(out, windowsSourceDB{rel: rel, path: path, salt: salt})
		salts[salt] = true
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].rel < out[j].rel })
	return out, salts, err
}

func windowsReadSaltHex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	b := make([]byte, 16)
	if _, err := io.ReadFull(f, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// verifyEncKeyStrict reports whether encKeyHex is the raw enc_key for the DB
// at path. Single-mode on purpose: callers must already know the material
// type (untyped candidates go through VerifyKeyCandidate, which returns the
// derived value to store). The previous dual-mode bool plus a native WCDB
// open() fallback were removed: the bool made callers store passphrases in
// enc_key slots, and the native branch crashed when WCDB was not bootstrapped.
func verifyEncKeyStrict(path, encKeyHex string) bool {
	ok, err := wcdb.VerifyEncKeyPage1(path, encKeyHex)
	return err == nil && ok
}

func windowsTargetProcesses() ([]windowsProcess, error) {
	if raw := firstEnv("WECHAT_CLI_WECHAT_PID", "WX_MCP_WECHAT_PID"); raw != "" {
		var out []windowsProcess
		for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
			if part == "" {
				continue
			}
			pid, err := strconv.ParseUint(part, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("parse WECHAT_CLI_WECHAT_PID=%q: %w", raw, err)
			}
			out = append(out, windowsProcess{pid: uint32(pid), exe: "env"})
		}
		return out, nil
	}
	names := map[string]bool{"weixin.exe": true, "wechat.exe": true}
	if raw := firstEnv("WECHAT_CLI_WECHAT_PROCESS", "WX_MCP_WECHAT_PROCESS"); raw != "" {
		names = map[string]bool{}
		for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
			part = strings.ToLower(strings.TrimSpace(part))
			if part == "" {
				continue
			}
			if !strings.HasSuffix(part, ".exe") {
				part += ".exe"
			}
			names[part] = true
		}
	}
	all, err := windowsEnumerateProcesses()
	if err != nil {
		return nil, err
	}
	var out []windowsProcess
	currentPID := windowsCurrentPID()
	for _, p := range all {
		if p.pid == currentPID {
			continue
		}
		if names[strings.ToLower(p.exe)] {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].exe != out[j].exe {
			return out[i].exe < out[j].exe
		}
		return out[i].pid < out[j].pid
	})
	return out, nil
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	return ""
}

func windowsEnumerateProcesses() ([]windowsProcess, error) {
	snap, _, err := procCreateToolhelp32.Call(th32csSnapProcess, 0)
	if snap == uintptr(syscall.InvalidHandle) || snap == 0 {
		return nil, err
	}
	defer procCloseHandle.Call(snap)
	return readProcessSnapshot(func(first bool, entry *windowsProcessEntry32) (uintptr, error) {
		proc := procProcess32NextW
		if first {
			proc = procProcess32FirstW
		}
		r, _, err := proc.Call(snap, uintptr(unsafe.Pointer(entry)))
		return r, err
	})
}

func readProcessSnapshot(read func(bool, *windowsProcessEntry32) (uintptr, error)) ([]windowsProcess, error) {
	var out []windowsProcess
	for first := true; ; first = false {
		var entry windowsProcessEntry32
		entry.Size = uint32(unsafe.Sizeof(entry))
		r, err := read(first, &entry)
		if r == 0 {
			// Only the documented exhaustion code proves the snapshot is
			// complete. Do not return a usable partial set after API failure.
			if errors.Is(err, syscall.ERROR_NO_MORE_FILES) {
				return out, nil
			}
			if err == nil || errors.Is(err, syscall.Errno(0)) {
				err = errors.New("enumeration failed without a Win32 error code")
			}
			api := "Process32NextW"
			if first {
				api = "Process32FirstW"
			}
			return nil, fmt.Errorf("%s: %w", api, err)
		}
		out = append(out, windowsProcess{
			pid: entry.ProcessID,
			exe: syscall.UTF16ToString(entry.ExeFile[:]),
		})
	}
}

func windowsCurrentPID() uint32 {
	r, _, _ := procGetCurrentProcessID.Call()
	return uint32(r)
}

func windowsScanProcess(pid uint32, targetSalts map[string]bool, scan *setupScan, deadline time.Time) error {
	h, _, err := procOpenProcess.Call(processVMRead|processQueryInformation, 0, uintptr(pid))
	if h == 0 {
		return processAccessError(pid, processVMRead|processQueryInformation, err)
	}
	defer procCloseHandle.Call(h)
	const maxUserAddress = uintptr(0x00007fffffffffff)
	for addr := uintptr(0); addr < maxUserAddress; {
		if scan.lenFound() == len(targetSalts) {
			return nil
		}
		if scan.stopped(deadline) {
			return errWindowsKeyScanDeadline
		}
		var m windowsMemoryBasicInformation
		r, _, _ := procVirtualQueryEx.Call(h, addr, uintptr(unsafe.Pointer(&m)), unsafe.Sizeof(m))
		if r == 0 {
			addr += 0x10000
			continue
		}
		next := m.BaseAddress + m.RegionSize
		if next <= addr {
			return nil
		}
		if windowsReadableRegion(m) {
			if err := windowsScanRegion(h, m.BaseAddress, m.RegionSize, targetSalts, scan, deadline); err != nil {
				return err
			}
		}
		addr = next
	}
	return nil
}

func windowsReadableRegion(m windowsMemoryBasicInformation) bool {
	return m.State == memCommit && m.RegionSize > 0 && m.Protect&pageNoAccess == 0 && m.Protect&pageGuard == 0
}

func windowsScanRegion(process uintptr, base, size uintptr, targetSalts map[string]bool, scan *setupScan, deadline time.Time) error {
	const chunkSize = 4 << 20
	var overlap []byte
	for off := uintptr(0); off < size; {
		if scan.lenFound() == len(targetSalts) {
			return nil
		}
		if scan.stopped(deadline) {
			return errWindowsKeyScanDeadline
		}
		n := chunkSize
		if remain := size - off; remain < uintptr(n) {
			n = int(remain)
		}
		buf := make([]byte, n)
		var got uintptr
		r, _, _ := procReadProcessMemory.Call(process, base+off, uintptr(unsafe.Pointer(&buf[0])), uintptr(n), uintptr(unsafe.Pointer(&got)))
		if r != 0 && got > 0 {
			data := append(append([]byte{}, overlap...), buf[:got]...)
			scanRawKeyLiterals(data, targetSalts, scan)
			if len(data) > 128 {
				overlap = append(overlap[:0], data[len(data)-128:]...)
			} else {
				overlap = append(overlap[:0], data...)
			}
		}
		off += uintptr(n)
	}
	return nil
}

// scanRawKeyLiterals matches transient x"<96 hex>" literals (64-hex enc_key +
// 32-hex salt). The literal shape binds key to salt, so hits are raw enc_keys.
func scanRawKeyLiterals(data []byte, targetSalts map[string]bool, scan *setupScan) int {
	var hits int
	for i := 0; i+99 <= len(data); i++ {
		if data[i] != 'x' || data[i+1] != '\'' || data[i+98] != '\'' {
			continue
		}
		hexBytes := data[i+2 : i+98]
		if !asciiHex(hexBytes) {
			continue
		}
		salt := strings.ToLower(string(hexBytes[64:96]))
		if !targetSalts[salt] {
			continue
		}
		key := strings.ToLower(string(hexBytes[:64]))
		if _, err := hex.DecodeString(key); err != nil {
			continue
		}
		scan.noteRawKey(salt, key, "rawkey-literal")
		hits++
	}
	return hits
}

func asciiHex(b []byte) bool {
	for _, c := range b {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func wxidFromAccountDir(path string) string {
	name := filepath.Base(filepath.Clean(path))
	if idx := strings.LastIndex(name, "_"); idx > 0 {
		return name[:idx]
	}
	return name
}

// windowsFindWeChatWinRange returns the memory range [start, end) occupied by
// WeChatWin.dll in the given process. Returns (0, 0) if not found.
func windowsFindWeChatWinRange(pid uint32) (start, end uintptr) {
	snap, _, _ := procCreateToolhelp32.Call(th32csSnapModule, uintptr(pid))
	if snap == uintptr(syscall.InvalidHandle) || snap == 0 {
		return 0, 0
	}
	defer procCloseHandle.Call(snap)

	var entry windowsModuleEntry32W
	entry.Size = uint32(unsafe.Sizeof(entry))
	r, _, _ := procModule32FirstW.Call(snap, uintptr(unsafe.Pointer(&entry)))
	if r == 0 {
		return 0, 0
	}
	for {
		name := strings.ToLower(syscall.UTF16ToString(entry.ModuleName[:]))
		if strings.Contains(name, "wechatwin.dll") {
			base := entry.ModBaseAddr
			return base, base + uintptr(entry.ModBaseSize)
		}
		entry.Size = uint32(unsafe.Sizeof(entry))
		r, _, _ = procModule32NextW.Call(snap, uintptr(unsafe.Pointer(&entry)))
		if r == 0 {
			break
		}
	}
	return 0, 0
}

// windowsReadKeyAtPointer reads a pointer-sized value at addr, interprets it
// as the address of a 32-byte SQLCipher key, reads those 32 bytes, and returns
// the key as a lowercase hex string. Returns "" on any read failure or if the
// key bytes are all-zero.
func windowsReadKeyAtPointer(process uintptr, addr uintptr) string {
	const ptrLen = 8
	ptrBuf := make([]byte, ptrLen)
	var got uintptr
	r, _, _ := procReadProcessMemory.Call(process, addr, uintptr(unsafe.Pointer(&ptrBuf[0])), ptrLen, uintptr(unsafe.Pointer(&got)))
	if r == 0 || got < ptrLen {
		return ""
	}
	keyAddr := uintptr(ptrBuf[0]) | uintptr(ptrBuf[1])<<8 | uintptr(ptrBuf[2])<<16 |
		uintptr(ptrBuf[3])<<24 | uintptr(ptrBuf[4])<<32 | uintptr(ptrBuf[5])<<40 |
		uintptr(ptrBuf[6])<<48 | uintptr(ptrBuf[7])<<56
	if keyAddr < 0x10000 || keyAddr > 0x00007fffffffffff {
		return ""
	}
	keyBuf := make([]byte, 32)
	r, _, _ = procReadProcessMemory.Call(process, keyAddr, uintptr(unsafe.Pointer(&keyBuf[0])), 32, uintptr(unsafe.Pointer(&got)))
	if r == 0 || got < 32 {
		return ""
	}
	// Reject trivially bad keys (all-zero)
	allZero := true
	for _, b := range keyBuf {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return ""
	}
	return hex.EncodeToString(keyBuf)
}

// ---------------------------------------------------------------------------
// Route A — poll scan (WeChat 4.1.10+)
//
// Codex static analysis of Weixin 4.1.11.54 found that the SQLCipher raw-key
// literal is produced by a single `x'%s'` format call at DB-open time and only
// lives briefly on the stack/heap of the opening thread. By the time
// wechat-cli scans (after login settles), that window is usually gone.
//
// Route A does not change WeChat; it repeatedly re-scans process memory in a
// tight loop until the deadline, so that if the user triggers a DB open (open
// a chat, switch conversations) during the scan window we catch the transient
// literal. It also matches the 64-hex "raw key only" form (salt read from the
// DB file header), not just the 96-hex "key+salt" form, because 4.1.x may emit
// either.
// ---------------------------------------------------------------------------

// scanRawKeyLiterals64 finds transient x'<64 hex>' raw-key literals (32-byte
// key only, no appended salt) and appends each unique candidate key to out.
// Unlike scanRawKeyLiterals it cannot match by salt, so candidates must be
// verified against the DB files by the caller.
func scanRawKeyLiterals64(data []byte, seen map[string]bool, out *[]string) int {
	var hits int
	// x' + 64 hex + ' == 67 bytes
	for i := 0; i+67 <= len(data); i++ {
		if data[i] != 'x' || data[i+1] != '\'' || data[i+66] != '\'' {
			continue
		}
		hexBytes := data[i+2 : i+66]
		if !asciiHex(hexBytes) {
			continue
		}
		key := strings.ToLower(string(hexBytes))
		if seen[key] {
			continue
		}
		seen[key] = true
		*out = append(*out, key)
		hits++
	}
	return hits
}

// windowsSinglePassScan performs one full-memory pass, matching both the
// 96-hex key+salt form (resolved directly against targetSalts) and the 64-hex
// key-only form (collected as candidates for later DB verification).
func windowsSinglePassScan(h uintptr, targetSalts map[string]bool, scan *setupScan, seen64 map[string]bool, candidates64 *[]string, deadline time.Time) error {
	const (
		chunkSize   = 4 << 20
		overlapKeep = 128
	)
	const maxUserAddress = uintptr(0x00007fffffffffff)
	for addr := uintptr(0); addr < maxUserAddress; {
		if scan.stopped(deadline) {
			return errWindowsKeyScanDeadline
		}
		var m windowsMemoryBasicInformation
		r, _, _ := procVirtualQueryEx.Call(h, addr, uintptr(unsafe.Pointer(&m)), unsafe.Sizeof(m))
		if r == 0 {
			addr += 0x10000
			continue
		}
		next := m.BaseAddress + m.RegionSize
		if next <= addr {
			return nil
		}
		if windowsReadableRegion(m) {
			var overlap []byte
			for off := uintptr(0); off < m.RegionSize; {
				if scan.stopped(deadline) {
					return errWindowsKeyScanDeadline
				}
				n := uintptr(chunkSize)
				if remain := m.RegionSize - off; remain < n {
					n = remain
				}
				buf := make([]byte, n)
				var got uintptr
				rr, _, _ := procReadProcessMemory.Call(h, m.BaseAddress+off, uintptr(unsafe.Pointer(&buf[0])), n, uintptr(unsafe.Pointer(&got)))
				if rr != 0 && got > 0 {
					data := append(append([]byte{}, overlap...), buf[:got]...)
					scanRawKeyLiterals(data, targetSalts, scan)
					scanRawKeyLiterals64(data, seen64, candidates64)
					if len(data) > overlapKeep {
						overlap = append(overlap[:0], data[len(data)-overlapKeep:]...)
					} else {
						overlap = append(overlap[:0], data...)
					}
				}
				off += n
			}
		}
		addr = next
	}
	return nil
}

// windowsPollScan repeatedly scans the process until keys are found or the
// deadline expires. Between passes it sleeps briefly so the user has time to
// trigger a DB open in WeChat. It resolves both the 96-hex form (via salt
// match) and the 64-hex form (via DB verification).
func windowsPollScan(pid uint32, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) error {
	h, _, _ := procOpenProcess.Call(processVMRead|processQueryInformation, 0, uintptr(pid))
	if h == 0 {
		return nil
	}
	defer procCloseHandle.Call(h)

	seen64 := map[string]bool{}
	verified64 := map[string]bool{}
	pollInterval := windowsPollInterval()

	for {
		if scan.lenFound() == len(salts) {
			return nil
		}
		if scan.stopped(deadline) {
			return errWindowsKeyScanDeadline
		}

		var candidates64 []string
		if err := windowsSinglePassScan(h, salts, scan, seen64, &candidates64, deadline); err != nil {
			return err
		}

		// Verify any newly-seen 64-hex candidates against unresolved DBs.
		newlySeen := candidates64[:0]
		for _, keyHex := range candidates64 {
			if verified64[keyHex] {
				continue
			}
			verified64[keyHex] = true
			newlySeen = append(newlySeen, keyHex)
		}
		if err := windowsVerifyCandidateKeys(newlySeen, dbs, salts, scan, deadline); err != nil {
			return err
		}

		if scan.lenFound() == len(salts) {
			return nil
		}
		if scan.stopped(deadline) {
			return errWindowsKeyScanDeadline
		}
		if err := scan.wait(pollInterval); err != nil {
			return err
		}
	}
}

func windowsPollInterval() time.Duration {
	raw := firstEnv("WECHAT_CLI_KEY_POLL_INTERVAL", "WX_MCP_KEY_POLL_INTERVAL")
	if raw == "" {
		return 500 * time.Millisecond
	}
	if d, err := time.ParseDuration(raw); err == nil && d > 0 {
		return d
	}
	if ms, err := strconv.Atoi(raw); err == nil && ms > 0 {
		return time.Duration(ms) * time.Millisecond
	}
	return 500 * time.Millisecond
}

func windowsKeyHookEnabled() bool {
	switch strings.ToLower(firstEnv("WECHAT_CLI_KEY_HOOK", "WX_MCP_KEY_HOOK")) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// windowsVerifyCandidateKeys verifies a batch of untyped 64-hex candidates
// against unresolved DBs, recording the DERIVED enc_key (never the raw
// candidate) for each match, plus the passphrase itself when a candidate
// turns out to be one. Shared by Route A, Route B and the capture-file route.
func windowsVerifyCandidateKeys(candidates []string, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) error {
	for _, keyHex := range candidates {
		if scan.lenFound() == len(salts) {
			return nil
		}
		if scan.stopped(deadline) {
			return errWindowsKeyScanDeadline
		}
		for _, db := range dbs {
			if scan.hasSalt(db.salt) {
				continue
			}
			v, err := VerifyKeyCandidate(db.path, keyHex, db.salt)
			if err != nil {
				scan.addDiag("candidate verify against %s: %v", db.rel, err)
				continue
			}
			scan.noteVerified(v)
		}
	}
	return nil
}
