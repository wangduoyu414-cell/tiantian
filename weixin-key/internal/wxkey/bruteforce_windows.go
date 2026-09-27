//go:build windows

package wxkey

// Route D — heap brute-force scan with in-process SQLCipher verification
// (version-independent; works for WeChat 4.0 / 4.1.x).
//
// The transient `x'%s'` raw-key literal disappears right after a DB is opened,
// which defeats timing-based scans (Route A) and hook-based capture (Route B/C).
// However, the 32-byte *raw encryption key* stays resident in the SQLCipher
// cipher context for as long as WeChat keeps the database open — i.e. the whole
// time WeChat is running. Route D exploits that:
//
//  1. Read page 1 (first 4096 bytes) of each target .db file. The first 16 bytes
//     are the SQLCipher KDF salt.
//  2. Walk WeChat's committed writable heap, treating every 8-byte-aligned
//     32-byte window as a candidate key.
//  3. Verify each candidate with SQLCipher 4's page-1 HMAC entirely in Go
//     (PBKDF2-HMAC-SHA512 with fast_kdf_iter=2 over the candidate, then one
//     HMAC-SHA512 over the page). This is microseconds per candidate, so a
//     multi-million-candidate scan finishes in seconds.
//  4. A candidate whose computed page-1 MAC matches the stored MAC is the real
//     raw key. WeChat 4.x shares one key across its DBs, so the winner is then
//     applied to every target salt.
//
// This does not hook, patch, or write to WeChat, and needs no timing window.

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// windowsKeyObjectScan looks for the {data pointer, length=32} byte-container
// passed to Weixin's x'%s' key-formatting path. Unlike a byte-by-byte heap
// brute force, this only verifies candidates referenced by a live 32-byte
// object, which keeps the scan bounded enough to use before active capture.
func windowsKeyObjectScan(pid uint32, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) error {
	var verifiers []*sqlcipherVerifier
	for _, db := range dbs {
		if scan.hasSalt(db.salt) {
			continue
		}
		if v := newSQLCipherVerifier(db.path, db.salt); v != nil {
			verifiers = append(verifiers, v)
		}
	}
	if len(verifiers) == 0 {
		return nil
	}
	ppBudget := windowsD0PassphraseBudget()

	h, _, openErr := procOpenProcess.Call(processVMRead|processQueryInformation, 0, uintptr(pid))
	if h == 0 {
		return processAccessError(pid, processVMRead|processQueryInformation, openErr)
	}
	defer procCloseHandle.Call(h)

	const (
		chunkSize      = 4 << 20
		maxUserAddress = uintptr(0x00007fffffffffff)
		objectSize     = 16
	)
	seenPointers := make(map[uintptr]struct{})
	var regions, objects, candidates int

	for addr := uintptr(0); addr < maxUserAddress; {
		if scan.stopped(deadline) {
			scan.addDiag("RouteD-object: deadline; regions=%d objects=%d candidates=%d", regions, objects, candidates)
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
			break
		}
		if windowsWritableHeapRegion(m) {
			regions++
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
				if rr != 0 && got >= objectSize {
					data := buf[:got]
					for i := 0; i+objectSize <= len(data); i += 8 {
						if binary.LittleEndian.Uint64(data[i+8:i+16]) != 32 {
							continue
						}
						objects++
						ptr64 := binary.LittleEndian.Uint64(data[i : i+8])
						if ptr64 == 0 || ptr64 > uint64(maxUserAddress) {
							continue
						}
						ptr := uintptr(ptr64)
						if _, ok := seenPointers[ptr]; ok {
							continue
						}
						seenPointers[ptr] = struct{}{}
						key := make([]byte, 32)
						if !windowsReadExact(h, ptr, key) || !plausibleKeyBytes(key) {
							continue
						}
						candidates++
						for _, v := range verifiers {
							if v.verify(key) {
								keyHex := hex.EncodeToString(key)
								windowsApplyBruteForceKey(keyHex, key, dbs, scan, "routeD0:key-object")
								scan.addDiag("RouteD-object: MATCH after %d candidate(s)", candidates)
								return nil
							}
						}
						// Dual mode, budgeted: a structure-backed candidate that
						// is not a raw enc_key may be the account passphrase.
						// PBKDF2 is ~89ms per derivation, capped by
						// WECHAT_CLI_D0_PASSPHRASE_BUDGET (default 8).
						if ppBudget > 0 && windowsD0DualModeCandidate(hex.EncodeToString(key), dbs, salts, scan, deadline, &ppBudget) {
							return nil
						}
					}
				}
				off += n
			}
		}
		addr = next
	}

	scan.addDiag("RouteD-object: complete; regions=%d objects=%d candidates=%d", regions, objects, candidates)
	return nil
}

// SQLCipher 4 defaults used by WeChat 4.x databases.
const (
	sqlcipherPageSize    = 4096
	sqlcipherHMACSize    = 64 // HMAC-SHA512 output
	sqlcipherMACKeySize  = 32 // derived MAC key length (SQLCipher 4 fast KDF output)
	sqlcipherReserve     = 80 // IV(16) + HMAC(64), aligned to AES block
	sqlcipherFastKDFIter = 2
	sqlcipherSaltMask    = 0x3a
)

// sqlcipherVerifier holds the precomputed page-1 material for one DB so we can
// verify a candidate 32-byte key with a single PBKDF2(2)+HMAC pass.
type sqlcipherVerifier struct {
	salt      string // hex salt (16 bytes) identifying the DB in the salts map
	dbSalt    []byte // raw 16-byte KDF salt from the database header
	hmacSalt  []byte // salt XOR 0x3a
	hmacInput []byte // page1[16 : pageSize-64] followed by pgno(1) LE32
	storedMAC []byte // page1[pageSize-64 : pageSize]
}

// newSQLCipherVerifier builds a verifier from a DB file's first page. Returns
// nil if the file is too small or unreadable.
func newSQLCipherVerifier(path, saltHex string) *sqlcipherVerifier {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	page := make([]byte, sqlcipherPageSize)
	if _, err := readFull(f, page); err != nil {
		return nil
	}
	salt := make([]byte, 16)
	copy(salt, page[:16])
	hmacSalt := make([]byte, 16)
	for i := range salt {
		hmacSalt[i] = salt[i] ^ sqlcipherSaltMask
	}
	// HMAC input = ciphertext+IV region, then the page number (1) as LE32.
	region := page[16 : sqlcipherPageSize-sqlcipherHMACSize]
	hmacInput := make([]byte, 0, len(region)+4)
	hmacInput = append(hmacInput, region...)
	hmacInput = append(hmacInput, 1, 0, 0, 0) // pgno 1, little-endian
	storedMAC := make([]byte, sqlcipherHMACSize)
	copy(storedMAC, page[sqlcipherPageSize-sqlcipherHMACSize:])
	return &sqlcipherVerifier{
		salt:      saltHex,
		dbSalt:    salt,
		hmacSalt:  hmacSalt,
		hmacInput: hmacInput,
		storedMAC: storedMAC,
	}
}

// windowsSaltNeighborhoodScan searches for known database salts in Weixin's
// writable memory and verifies only nearby 32-byte candidates. SQLCipher keeps
// the salt and derived key material in the same cipher-context object, so this
// is substantially cheaper than treating every heap window as a possible key.
func windowsSaltNeighborhoodScan(pid uint32, dbs []windowsSourceDB, scan *setupScan, deadline time.Time) error {
	verifiers := make([]*sqlcipherVerifier, 0, len(dbs))
	for _, db := range dbs {
		if scan.hasSalt(db.salt) {
			continue
		}
		if v := newSQLCipherVerifier(db.path, db.salt); v != nil {
			verifiers = append(verifiers, v)
		}
	}
	if len(verifiers) == 0 {
		return nil
	}

	h, _, openErr := procOpenProcess.Call(processVMRead|processQueryInformation, 0, uintptr(pid))
	if h == 0 {
		return processAccessError(pid, processVMRead|processQueryInformation, openErr)
	}
	defer procCloseHandle.Call(h)

	const (
		chunkSize      = 4 << 20
		contextRadius  = 4096
		maxUserAddress = uintptr(0x00007fffffffffff)
		keyLen         = 32
	)
	var regions, saltHits, candidates int
	seen := make(map[uintptr]struct{})

	for addr := uintptr(0); addr < maxUserAddress; {
		if scan.stopped(deadline) {
			scan.addDiag("RouteD-salt: deadline; regions=%d salt_hits=%d candidates=%d", regions, saltHits, candidates)
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
			break
		}
		if windowsWritableHeapRegion(m) {
			regions++
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
				if rr != 0 && got >= keyLen {
					data := buf[:got]
					for _, v := range verifiers {
						if scan.hasSalt(v.salt) {
							continue
						}
						for from := 0; from < len(data); {
							rel := bytes.Index(data[from:], v.dbSalt)
							if rel < 0 {
								break
							}
							idx := from + rel
							absoluteSalt := m.BaseAddress + off + uintptr(idx)
							from = idx + 1
							if _, ok := seen[absoluteSalt]; ok {
								continue
							}
							seen[absoluteSalt] = struct{}{}
							saltHits++
							start := idx - contextRadius
							if start < 0 {
								start = 0
							}
							end := idx + len(v.dbSalt) + contextRadius
							if max := len(data) - keyLen; end > max {
								end = max
							}
							for i := start; i <= end; i++ {
								cand := data[i : i+keyLen]
								if !plausibleKeyBytes(cand) {
									continue
								}
								candidates++
								if v.verify(cand) {
									keyHex := hex.EncodeToString(cand)
									windowsApplyBruteForceKey(keyHex, cand, dbs, scan, "routeD1:salt-neighborhood")
									scan.addDiag("RouteD-salt: verified key material near salt after %d candidates", candidates)
									break
								}
							}
						}
					}
				}
				off += n
			}
		}
		addr = next
	}
	scan.addDiag("RouteD-salt: complete; regions=%d salt_hits=%d candidates=%d matched=%d", regions, saltHits, candidates, scan.lenFound())
	return nil
}

// verify returns true if key32 is the raw encryption key for this DB.
func (v *sqlcipherVerifier) verify(key32 []byte) bool {
	hmacKey := pbkdf2SHA512(key32, v.hmacSalt, sqlcipherFastKDFIter, sqlcipherMACKeySize)
	mac := hmac.New(sha512.New, hmacKey)
	mac.Write(v.hmacInput)
	sum := mac.Sum(nil)
	return hmac.Equal(sum, v.storedMAC)
}

// pbkdf2SHA512 is a minimal PBKDF2-HMAC-SHA512 (enough for keyLen<=64, the only
// case we need). Avoids adding a dependency on golang.org/x/crypto.
func pbkdf2SHA512(password, salt []byte, iter, keyLen int) []byte {
	// Single output block (SHA512 = 64 bytes) covers keyLen<=64.
	prf := hmac.New(sha512.New, password)
	prf.Write(salt)
	prf.Write([]byte{0, 0, 0, 1}) // block index 1, big-endian
	u := prf.Sum(nil)
	out := make([]byte, len(u))
	copy(out, u)
	for i := 1; i < iter; i++ {
		prf.Reset()
		prf.Write(u)
		u = prf.Sum(nil)
		for j := range out {
			out[j] ^= u[j]
		}
	}
	if keyLen > len(out) {
		keyLen = len(out)
	}
	return out[:keyLen]
}

func readFull(f *os.File, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := f.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// windowsBruteForceScan is Route D. It scans the committed writable heap of pid
// for 32-byte raw-key candidates and verifies each in-process against the DB
// page-1 HMAC. On a match it fills found for every target salt (WeChat 4.x
// shares one key across DBs).
//
// The scan is parallelized over memory regions: a bounded worker pool
// (WECHAT_CLI_SCAN_WORKERS, default min(4, NumCPU)) pulls regions from a queue;
// verification is pure CPU (HMAC), and a shared quit flag stops everyone on
// full coverage or deadline. Results converge on the mutex-guarded setupScan.
func windowsBruteForceScan(pid uint32, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time, marks map[uintptr]regionFingerprint) error {
	var verifiers []*sqlcipherVerifier
	for _, db := range dbs {
		if scan.hasSalt(db.salt) {
			continue
		}
		if v := newSQLCipherVerifier(db.path, db.salt); v != nil {
			verifiers = append(verifiers, v)
		}
	}
	if len(verifiers) == 0 {
		scan.addDiag("RouteD: no verifiers built (all salts resolved or page1 unreadable)")
		return nil
	}

	h, _, openErr := procOpenProcess.Call(processVMRead|processQueryInformation, 0, uintptr(pid))
	if h == 0 {
		return processAccessError(pid, processVMRead|processQueryInformation, openErr)
	}
	defer procCloseHandle.Call(h)

	regions := windowsCollectWritableHeapRegions(h, func() bool { return scan.stopped(deadline) })
	if len(regions) == 0 {
		scan.addDiag("RouteD: no writable heap regions")
		return nil
	}
	// Change-priority ordering: regions that changed since the run-start
	// fingerprint (or are new) scan first. Unchanged regions are kept, just
	// later — change is a priority hint, never an exclusion.
	if marks != nil {
		read := windowsProcessRegionReader(h)
		markDL := windowsRouteDeadline(deadline, 250*time.Millisecond)
		ordered, changed, _ := prioritizeChangedRegions(marks, func(base uintptr, n int) ([]byte, error) {
			if scan.stopped(markDL) {
				return nil, errWindowsKeyScanDeadline
			}
			return read(base, n)
		}, regions)
		regions = ordered
		scan.addDiag("RouteD: %d/%d regions changed since run start (prioritized)", changed, len(regions))
	}
	scan.addDiag("RouteD: built %d verifier(s), scanning pid %d heap: %d regions, %d workers",
		len(verifiers), pid, len(regions), windowsScanWorkerCount())

	quit := make(chan struct{})
	var quitOnce sync.Once
	stop := func() { quitOnce.Do(func() { close(quit) }) }
	stopped := func() bool {
		if scan.ctx.Err() != nil {
			stop()
			return true
		}
		select {
		case <-quit:
			return true
		default:
			return false
		}
	}
	if !deadline.IsZero() {
		timer := time.AfterFunc(time.Until(deadline), stop)
		defer timer.Stop()
	}

	jobs := make(chan windowsMemRegion)
	var wg sync.WaitGroup
	var candidates uint64

	for w := 0; w < windowsScanWorkerCount(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for region := range jobs {
				if stopped() {
					return
				}
				if key := windowsScanRegionForKeys(h, region, verifiers, &candidates, stopped); key != nil {
					keyHex := hex.EncodeToString(key)
					windowsApplyBruteForceKey(keyHex, key, dbs, scan, "routeD2:heap-bruteforce")
					scan.addDiag("RouteD: MATCH after ~%d candidates", atomic.LoadUint64(&candidates))
					if scan.lenFound() == len(salts) {
						stop()
						return
					}
				}
			}
		}()
	}

	for _, r := range regions {
		if stopped() {
			break
		}
		select {
		case jobs <- r:
		case <-quit:
			break
		case <-scan.ctx.Done():
			stop()
		}
	}
	close(jobs)
	wg.Wait()

	scan.addDiag("RouteD: scan done; regions=%d candidates=%d matched=%d", len(regions), candidates, scan.lenFound())
	if scan.lenFound() > 0 {
		return nil
	}
	if !deadline.IsZero() && scan.stopped(deadline) {
		return errWindowsKeyScanDeadline
	}
	return nil
}

// windowsApplyBruteForceKey records keyHex for the matched salt and re-verifies
// it against every other unresolved DB, filling found where it also matches.
func windowsApplyBruteForceKey(keyHex string, key32 []byte, dbs []windowsSourceDB, scan *setupScan, source string) {
	for _, db := range dbs {
		if scan.hasSalt(db.salt) {
			continue
		}
		v := newSQLCipherVerifier(db.path, db.salt)
		if v != nil && v.verify(key32) {
			scan.noteRawKey(db.salt, keyHex, source)
		}
	}
}

// plausibleKeyBytes rejects obviously-non-key windows cheaply before the HMAC
// check: all-zero, single repeated byte, or too few distinct byte values.
func plausibleKeyBytes(b []byte) bool {
	var seen [256]bool
	distinct := 0
	for _, c := range b {
		if !seen[c] {
			seen[c] = true
			distinct++
		}
	}
	// A real 256-bit key is high-entropy; require a healthy spread of bytes.
	return distinct >= 16
}

// windowsWritableHeapRegion reports whether m is committed, readable, writable
// memory that is not a guard page — i.e. a heap region worth scanning for keys.
func windowsWritableHeapRegion(m windowsMemoryBasicInformation) bool {
	if m.State != memCommit {
		return false
	}
	if m.Protect&pageGuard != 0 || m.Protect&pageNoAccess != 0 {
		return false
	}
	switch m.Protect & 0xFF {
	case 0x04, 0x40, 0x08, 0x80: // READWRITE, EXECUTE_READWRITE, WRITECOPY, EXECUTE_WRITECOPY
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Route D parallelism helpers
// ---------------------------------------------------------------------------

// windowsMemRegion is one committed writable heap region to scan.
type windowsMemRegion struct {
	base uintptr
	size uintptr
}

// windowsCollectWritableHeapRegions enumerates committed, writable, non-guard
// regions once, so workers share the enumeration instead of racing on
// VirtualQueryEx order.
func windowsCollectWritableHeapRegions(h uintptr, stopFns ...func() bool) []windowsMemRegion {
	const maxUserAddress = uintptr(0x00007fffffffffff)
	const maxRegions = 1 << 20 // sanity bound
	var out []windowsMemRegion
	for addr := uintptr(0); addr < maxUserAddress; {
		if len(stopFns) > 0 && stopFns[0]() {
			break
		}
		var m windowsMemoryBasicInformation
		r, _, _ := procVirtualQueryEx.Call(h, addr, uintptr(unsafe.Pointer(&m)), unsafe.Sizeof(m))
		if r == 0 {
			addr += 0x10000
			continue
		}
		next := m.BaseAddress + m.RegionSize
		if next <= addr {
			break
		}
		if windowsWritableHeapRegion(m) {
			out = append(out, windowsMemRegion{base: m.BaseAddress, size: m.RegionSize})
			if len(out) >= maxRegions {
				break
			}
		}
		addr = next
	}
	return out
}

// windowsScanWorkerCount resolves the Route D worker count:
// WECHAT_CLI_SCAN_WORKERS, else min(4, NumCPU).
func windowsScanWorkerCount() int {
	if raw := firstEnv("WECHAT_CLI_SCAN_WORKERS"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 64 {
			return n
		}
	}
	if n := runtime.NumCPU(); n < 4 {
		return n
	}
	return 4
}

// windowsScanRegionForKeys scans one region in 4MB chunks (with carry-over so
// windows split by chunking are still found) and returns the verified key.
func windowsScanRegionForKeys(h uintptr, region windowsMemRegion, verifiers []*sqlcipherVerifier, candidates *uint64, stopped func() bool) []byte {
	const chunkSize = 4 << 20
	var overlap []byte
	for off := uintptr(0); off < region.size; {
		if stopped() {
			return nil
		}
		n := uintptr(chunkSize)
		if remain := region.size - off; remain < n {
			n = remain
		}
		buf := make([]byte, n)
		var got uintptr
		rr, _, _ := procReadProcessMemory.Call(h, region.base+off, uintptr(unsafe.Pointer(&buf[0])), n, uintptr(unsafe.Pointer(&got)))
		if rr != 0 && got >= 32 {
			data := append(append([]byte{}, overlap...), buf[:got]...)
			if key := scanChunkForKeys(data, verifiers, candidates); key != nil {
				return key
			}
			if len(data) > 64 {
				overlap = append(overlap[:0], data[len(data)-64:]...)
			} else {
				overlap = append(overlap[:0], data...)
			}
		}
		off += n
	}
	return nil
}

// scanChunkForKeys verifies every 8-byte-aligned 32-byte window of data
// against the verifiers. Pure function (no process IO), so it is directly
// testable and its result is worker-count-independent.
func scanChunkForKeys(data []byte, verifiers []*sqlcipherVerifier, candidates *uint64) []byte {
	limit := len(data) - 32
	for i := 0; i <= limit; i += 8 {
		cand := data[i : i+32]
		if !plausibleKeyBytes(cand) {
			continue
		}
		if candidates != nil {
			atomic.AddUint64(candidates, 1)
		}
		for _, v := range verifiers {
			if v.verify(cand) {
				key := make([]byte, 32)
				copy(key, cand)
				return key
			}
		}
	}
	return nil
}

// windowsD0PassphraseBudget caps how many structure-backed candidates get a
// PBKDF2 passphrase check per D0 run (PBKDF2-256000 ≈ 89ms each).
func windowsD0PassphraseBudget() int {
	if raw := firstEnv("WECHAT_CLI_D0_PASSPHRASE_BUDGET"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			return n
		}
	}
	return 8
}

// windowsD0DualModeCandidate verifies one structure-evidenced candidate in
// dual mode (raw enc_key, then passphrase) against the unresolved DBs,
// decrementing *budget per candidate attempted. When the candidate turns out
// to be the passphrase, the derived enc_keys are applied to every unresolved
// salt at once.
func windowsD0DualModeCandidate(candHex string, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time, budget *int) bool {
	if *budget <= 0 {
		return false
	}
	*budget--
	resolved := false
	for _, db := range dbs {
		if scan.hasSalt(db.salt) {
			continue
		}
		if scan.stopped(deadline) {
			return resolved
		}
		v, err := VerifyKeyCandidate(db.path, candHex, db.salt)
		if err != nil {
			continue
		}
		if v.Match {
			v.Source = "routeD0:key-object-dual"
			scan.noteVerified(v)
			resolved = true
			if v.Kind == MaterialPassphrase {
				windowsApplyPassphrase(v.PassphraseHex, "routeD0:key-object-dual", dbs, salts, scan, deadline)
				scan.addDiag("RouteD-object: candidate is the account PASSPHRASE; derived all salts")
				return true
			}
		}
	}
	return resolved
}
