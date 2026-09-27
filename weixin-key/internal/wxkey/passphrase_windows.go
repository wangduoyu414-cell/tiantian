//go:build windows

package wxkey

import (
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"weixin-key/internal/wcdb"
)

// windowsEnvPassphrase applies the WECHAT_CLI_PASSPHRASE_HEX override when
// set: it derives enc_keys for every unresolved salt and verifies them.
// It returns nil when the variable is unset. This check is free and
// deterministic, so runSetup invokes it before any memory scanning.
func windowsEnvPassphrase(dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) error {
	raw := strings.TrimSpace(os.Getenv("WECHAT_CLI_PASSPHRASE_HEX"))
	if raw == "" {
		return nil
	}
	pp, err := normalizePassphraseHex(raw)
	if err != nil {
		return err
	}
	scan.addDiag("RouteE: using passphrase from WECHAT_CLI_PASSPHRASE_HEX")
	if matched := windowsApplyPassphrase(pp, "env:WECHAT_CLI_PASSPHRASE_HEX", dbs, salts, scan, deadline); matched == 0 {
		return errors.New("WECHAT_CLI_PASSPHRASE_HEX did not verify against any DB")
	}
	return nil
}

// windowsApplyPassphrase derives enc_keys from passphraseHex for every
// unresolved salt and verifies each against its DB. The DERIVED enc_key (not
// the passphrase) is recorded per salt; the passphrase itself is noted once
// so it can be cached for future/new DBs. Returns newly resolved salt count.
func windowsApplyPassphrase(passphraseHex, source string, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) int {
	matched := 0
	for _, db := range dbs {
		if scan.stopped(deadline) {
			break
		}
		if scan.hasSalt(db.salt) {
			continue
		}
		if !salts[db.salt] {
			continue
		}
		encKeyHex, err := wcdb.DeriveEncKey(passphraseHex, db.salt)
		if err != nil {
			scan.addDiag("RouteE: derive failed for salt %s: %v", db.salt, err)
			continue
		}
		if verifyEncKeyStrict(db.path, encKeyHex) {
			scan.noteVerified(KeyCandidateVerification{
				Match:         true,
				Kind:          MaterialPassphrase,
				SaltHex:       db.salt,
				EncKeyHex:     encKeyHex,
				PassphraseHex: passphraseHex,
				Source:        source,
			})
			matched++
			scan.addDiag("RouteE: verified salt %s via %s", db.salt, source)
		}
	}
	return matched
}

// windowsBruteForcePassphrase scans readable memory of pid for 32-byte
// high-entropy candidates and treats each as a SQLCipher passphrase. It stops
// at the first candidate that derives a key opening any unresolved DB.
func windowsBruteForcePassphrase(pid uint32, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) (string, string, error) {
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
		return "", "", errors.New("no unresolved DBs to verify against")
	}

	h, _, openErr := procOpenProcess.Call(processVMRead|processQueryInformation, 0, uintptr(pid))
	if h == 0 {
		return "", "", processAccessError(pid, processVMRead|processQueryInformation, openErr)
	}
	defer procCloseHandle.Call(h)

	const chunkSize = 4 << 20
	const maxUserAddr = uintptr(0x00007fffffffffff)
	var (
		bytesRead  uint64
		candidates uint64
		regions    int
	)

	scan.addDiag("RouteE: starting passphrase brute-force scan over pid %d", pid)

	for addr := uintptr(0); addr < maxUserAddr; {
		if scan.stopped(deadline) {
			scan.addDiag("RouteE: DEADLINE hit; bytes=%dMB candidates=%d", bytesRead>>20, candidates)
			return "", "", errWindowsKeyScanDeadline
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
		if windowsReadableRegion(m) {
			regions++
			for off := uintptr(0); off < m.RegionSize; {
				if scan.stopped(deadline) {
					scan.addDiag("RouteE: DEADLINE hit mid-region; bytes=%dMB candidates=%d", bytesRead>>20, candidates)
					return "", "", errWindowsKeyScanDeadline
				}
				n := uintptr(chunkSize)
				if remain := m.RegionSize - off; remain < n {
					n = remain
				}
				buf := make([]byte, n)
				var got uintptr
				r, _, _ := procReadProcessMemory.Call(h, m.BaseAddress+off, uintptr(unsafe.Pointer(&buf[0])), n, uintptr(unsafe.Pointer(&got)))
				if r == 0 || got < wcdb.PassphraseLength {
					off += n
					continue
				}
				bytesRead += uint64(got)
				data := buf[:got]
				limit := len(data) - wcdb.PassphraseLength
				for i := 0; i <= limit; i++ {
					cand := data[i : i+wcdb.PassphraseLength]
					if !plausiblePassphrase(cand) {
						continue
					}
					candidates++
					src := "bruteforce@0x" + strconvHex(m.BaseAddress+off+uintptr(i))
					ppHex := hex.EncodeToString(cand)
					if matched := windowsApplyPassphrase(ppHex, src, dbs, salts, scan, deadline); matched > 0 {
						scan.addDiag("RouteE: MATCH after %d candidates, address=%s", candidates, src)
						return ppHex, src, nil
					}
				}
				off += uintptr(got)
			}
		}
		addr = next
	}

	scan.addDiag("RouteE: brute-force COMPLETE, no match; regions=%d bytes=%dMB candidates=%d", regions, bytesRead>>20, candidates)
	return "", "", errors.New("no passphrase candidate verified")
}

// plausiblePassphrase filters 32-byte memory windows. We require at least 20
// distinct byte values and reject all-zero / all-same blocks. Real 256-bit
// passphrase material is high-entropy.
func plausiblePassphrase(b []byte) bool {
	var seen [256]bool
	distinct := 0
	for _, c := range b {
		if !seen[c] {
			seen[c] = true
			distinct++
		}
	}
	return distinct >= 20
}

func strconvHex(u uintptr) string {
	return strings.ToLower(strings.TrimPrefix(strconv.FormatUint(uint64(u), 16), "0x"))
}
