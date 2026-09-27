//go:build windows

package wxkey

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"weixin-key/internal/wcdb"
)

// windowsKeySignature describes a byte pattern inside WeChatWin.dll/Weixin.dll
// that helps locate key/passphrase material. Each signature is version-agnostic
// in shape but must match the byte layout for a specific WeChat build.
type windowsKeySignature struct {
	Name       string `json:"name"`        // human readable, e.g. "wechatwin_4_1_11_passphrase"
	PatternHex string `json:"pattern_hex"` // hex bytes to match, may contain ?? wildcards
	Offset     int    `json:"offset"`      // bytes from pattern start to key/passphrase
	Length     int    `json:"length"`      // bytes to read (32 for passphrase, 32 for raw key)
	KeyType    string `json:"key_type"`    // "passphrase" or "rawkey"
	ModuleHint string `json:"module_hint"` // "wechatwin.dll", "weixin.dll", or empty for any
}

// windowsScanSignatures is the active-capture path for Route E. It looks for
// user-supplied signatures in WeChat's core module, reads candidate key/passphrase
// material at the indicated offset, verifies each against the DBs, and returns
// the first verified material WITH ITS ACTUAL TYPE: a rawkey signature hit is
// an enc_key (already recorded per salt) and must never be persisted as a
// passphrase; a passphrase hit returns the passphrase for the config.
func windowsScanSignatures(pid uint32, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) (kind MaterialKind, materialHex, source string, err error) {
	sigs := windowsLoadSignatures(scan)
	if len(sigs) == 0 {
		return "", "", "", fmt.Errorf("no signatures configured; set WECHAT_CLI_KEY_SIGNATURE or add ~/.config/wxcli/signatures.json")
	}

	h, _, openErr := procOpenProcess.Call(processVMRead|processQueryInformation, 0, uintptr(pid))
	if h == 0 {
		return "", "", "", processAccessError(pid, processVMRead|processQueryInformation, openErr)
	}
	defer procCloseHandle.Call(h)

	for _, sig := range sigs {
		if scan.stopped(deadline) {
			return "", "", "", errWindowsKeyScanDeadline
		}

		modStart, modEnd := windowsFindModuleRangeByHint(pid, sig.ModuleHint)
		if modStart == 0 || modEnd == 0 {
			scan.addDiag("RouteE: module not found for signature %s", sig.Name)
			continue
		}

		pat, err := parseHexPattern(sig.PatternHex)
		if err != nil {
			scan.addDiag("RouteE: invalid pattern %s: %v", sig.Name, err)
			continue
		}

		addrs := windowsFindPatternMatches(h, modStart, modEnd, pat, deadline, scan)
		scan.addDiag("RouteE: signature %s matched %d time(s)", sig.Name, len(addrs))

		for _, addr := range addrs {
			if scan.stopped(deadline) {
				return "", "", "", errWindowsKeyScanDeadline
			}

			keyAddr := addr + uintptr(sig.Offset)
			cand := windowsReadProcessBytes(h, keyAddr, sig.Length)
			if len(cand) != sig.Length {
				continue
			}

			candHex := hex.EncodeToString(cand)
			src := fmt.Sprintf("sig:%s@0x%x", sig.Name, keyAddr)
			if matched := windowsApplyPassphraseOrKey(candHex, sig.KeyType, src, dbs, salts, scan, deadline); matched > 0 {
				if sig.KeyType == "rawkey" {
					return MaterialEncKey, candHex, src, nil
				}
				return MaterialPassphrase, candHex, src, nil
			}
		}
	}

	return "", "", "", fmt.Errorf("no configured signature verified against DBs")
}

// windowsApplyPassphraseOrKey verifies a candidate key/passphrase against
// unresolved DBs. keyType is "passphrase" (derive PBKDF2) or "rawkey" (use as
// enc_key directly).
func windowsApplyPassphraseOrKey(candidateHex, keyType, source string, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) int {
	if keyType == "rawkey" {
		return windowsApplyRawKey(candidateHex, source, dbs, salts, scan, deadline)
	}
	return windowsApplyPassphrase(candidateHex, source, dbs, salts, scan, deadline)
}

// windowsApplyRawKey verifies a 32-byte raw enc_key against unresolved DBs and
// records it unchanged (it IS the enc_key, no derivation needed).
func windowsApplyRawKey(encKeyHex, source string, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) int {
	if len(encKeyHex) != 64 {
		return 0
	}
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
		if verifyEncKeyStrict(db.path, encKeyHex) {
			scan.noteVerified(KeyCandidateVerification{
				Match:     true,
				Kind:      MaterialEncKey,
				SaltHex:   db.salt,
				EncKeyHex: encKeyHex,
				Source:    source,
			})
			matched++
			scan.addDiag("RouteE: verified salt %s via rawkey %s", db.salt, source)
		}
	}
	return matched
}

// windowsFindModuleRangeByHint returns the module range for WeChatWin.dll or
// Weixin.dll based on the hint, falling back to either if the hint is empty.
func windowsFindModuleRangeByHint(pid uint32, hint string) (start, end uintptr) {
	hint = strings.ToLower(strings.TrimSpace(hint))
	names := []string{"wechatwin.dll", "weixin.dll"}
	if hint != "" {
		names = []string{hint}
	}
	for _, name := range names {
		if s, e := windowsFindModuleRange(pid, name); s != 0 && e != 0 {
			return s, e
		}
	}
	return 0, 0
}

// hexPattern is a byte pattern with optional wildcard bytes.
type hexPattern struct {
	bytes  []byte
	mask   []bool // true = must match bytes[i]
	length int
}

func parseHexPattern(s string) (*hexPattern, error) {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ToLower(s)
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("pattern length must be even")
	}

	p := &hexPattern{}
	for i := 0; i < len(s); i += 2 {
		tok := s[i : i+2]
		if tok == "??" {
			p.bytes = append(p.bytes, 0)
			p.mask = append(p.mask, false)
		} else {
			b, err := strconv.ParseUint(tok, 16, 8)
			if err != nil {
				return nil, fmt.Errorf("invalid hex byte %q", tok)
			}
			p.bytes = append(p.bytes, byte(b))
			p.mask = append(p.mask, true)
		}
	}
	p.length = len(p.bytes)
	if p.length == 0 {
		return nil, fmt.Errorf("empty pattern")
	}
	return p, nil
}

// windowsFindPatternMatches scans [start,end) for all occurrences of pat and
// returns their absolute addresses.
func windowsFindPatternMatches(h, start, end uintptr, pat *hexPattern, deadline time.Time, scan *setupScan) []uintptr {
	const chunk = 4 << 20
	var out []uintptr
	overlap := pat.length - 1
	if overlap < 0 {
		overlap = 0
	}

	for addr := start; addr < end; {
		if scan.stopped(deadline) {
			return out
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
			regionStart := m.BaseAddress
			regionEnd := next
			if regionStart < start {
				regionStart = start
			}
			if regionEnd > end {
				regionEnd = end
			}
			var prev []byte
			var prevAddr uintptr
			for off := regionStart; off < regionEnd; {
				if scan.stopped(deadline) {
					return out
				}
				n := uintptr(chunk)
				if remain := regionEnd - off; remain < n {
					n = remain
				}
				buf := make([]byte, n)
				var got uintptr
				r, _, _ := procReadProcessMemory.Call(h, off, uintptr(unsafe.Pointer(&buf[0])), n, uintptr(unsafe.Pointer(&got)))
				if r == 0 || got == 0 {
					off += n
					continue
				}
				data := append(append([]byte{}, prev...), buf[:got]...)
				for i := 0; i+pat.length <= len(data); i++ {
					if matchPattern(data[i:i+pat.length], pat) {
						matchAddr := off + uintptr(i) - uintptr(len(prev))
						if matchAddr >= start {
							out = append(out, matchAddr)
						}
					}
				}
				if len(buf[:got]) >= overlap {
					prev = append(prev[:0], buf[got-uintptr(overlap):got]...)
					_ = prevAddr
					prevAddr = off + got - uintptr(overlap)
				} else {
					prev = append(prev[:0], buf[:got]...)
					prevAddr = off
				}
				_ = prevAddr
				off += uintptr(got)
			}
		}
		addr = next
	}
	return out
}

func matchPattern(data []byte, pat *hexPattern) bool {
	for i := 0; i < pat.length; i++ {
		if pat.mask[i] && data[i] != pat.bytes[i] {
			return false
		}
	}
	return true
}

// windowsReadProcessBytes reads exactly n bytes from addr in the target process.
// Returns a shorter slice on failure.
func windowsReadProcessBytes(h uintptr, addr uintptr, n int) []byte {
	buf := make([]byte, n)
	var got uintptr
	r, _, _ := procReadProcessMemory.Call(h, addr, uintptr(unsafe.Pointer(&buf[0])), uintptr(n), uintptr(unsafe.Pointer(&got)))
	if r == 0 {
		return nil
	}
	return buf[:got]
}

// windowsLoadSignatures loads key signatures from env var or a JSON file. The
// env var takes precedence and is meant for quick iteration; the file is the
// persistent per-user signature store.
func windowsLoadSignatures(scan *setupScan) []windowsKeySignature {
	if raw := os.Getenv("WECHAT_CLI_KEY_SIGNATURE"); raw != "" {
		sig, err := parseSignatureEnv(raw)
		if err == nil {
			return []windowsKeySignature{sig}
		}
		scan.addDiag("RouteE: invalid WECHAT_CLI_KEY_SIGNATURE: %v", err)
	}
	return nil
}

// parseSignatureEnv parses a compact signature spec:
//
//	name:pattern_hex:offset:length:key_type[:module_hint]
//
// Example:
//
//	wechatwin_4_1_11:48 8B ?? ?? ?? ?? ?? 48 8D:24:32:passphrase:wechatwin.dll
func parseSignatureEnv(raw string) (windowsKeySignature, error) {
	parts := strings.Split(raw, ":")
	if len(parts) < 5 {
		return windowsKeySignature{}, fmt.Errorf("need name:pattern_hex:offset:length:key_type")
	}
	offset, err := strconv.Atoi(parts[2])
	if err != nil {
		return windowsKeySignature{}, fmt.Errorf("offset: %w", err)
	}
	length, err := strconv.Atoi(parts[3])
	if err != nil {
		return windowsKeySignature{}, fmt.Errorf("length: %w", err)
	}
	sig := windowsKeySignature{
		Name:       parts[0],
		PatternHex: parts[1],
		Offset:     offset,
		Length:     length,
		KeyType:    parts[4],
	}
	if len(parts) > 5 {
		sig.ModuleHint = parts[5]
	}
	if sig.KeyType != "passphrase" && sig.KeyType != "rawkey" {
		return windowsKeySignature{}, fmt.Errorf("key_type must be passphrase or rawkey")
	}
	return sig, nil
}

// windowsSignatureFromBytes is a helper for callers who already have raw
// candidate bytes (e.g. from a debugger or external tool) and want to verify
// them through the same pipeline.
func windowsVerifyCandidateBytes(candidate []byte, keyType string, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) bool {
	if len(candidate) != wcdb.PassphraseLength {
		return false
	}
	return windowsApplyPassphraseOrKey(hex.EncodeToString(candidate), keyType, "external", dbs, salts, scan, deadline) > 0
}
