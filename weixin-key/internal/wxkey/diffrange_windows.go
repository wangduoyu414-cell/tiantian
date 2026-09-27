//go:build windows

package wxkey

import (
	"encoding/binary"
	"errors"
	"hash/fnv"
	"unsafe"
)

// Operation-driven change-priority scanning (bounded).
//
// Between login/open operations and the heap scan, the regions that CHANGED
// are statistically the best places to look first. We fingerprint each
// writable region cheaply (head+tail sample, not a full read) at run start,
// then re-fingerprint before the heap brute-force: changed or new regions
// scan first. Unchanged regions are NEVER excluded — change is a priority
// hint only, per the design contract. No full memory is retained; marks are
// released with the run.

const regionSampleBytes = 8192 // head 4KB + tail 4KB per region
const maxRegionFingerprints = 1024

// regionFingerprint is one region's cheap change marker.
type regionFingerprint struct {
	base uintptr
	size uintptr
	fp   uint64
}

// regionReader reads up to n bytes at base. Injectable for tests.
type regionReader func(base uintptr, n int) ([]byte, error)

// fingerprintRegion samples head+tail of a region and hashes them with the
// region size. Unreadable regions get fp=0 (they will sort as "changed",
// which is the safe direction).
func fingerprintRegion(read regionReader, base, size uintptr) uint64 {
	h := fnv.New64a()
	var sz [8]byte
	binary.LittleEndian.PutUint64(sz[:], uint64(size))
	h.Write(sz[:])

	if size == 0 {
		return 0
	}
	head, err := read(base, int(min(size, regionSampleBytes/2)))
	if err != nil || len(head) == 0 {
		return 0
	}
	h.Write(head)
	if size > regionSampleBytes/2 {
		tailOff := size - regionSampleBytes/2
		if tail, err := read(base+tailOff, regionSampleBytes/2); err == nil {
			h.Write(tail)
		}
	}
	return h.Sum64()
}

// snapshotRegions fingerprints every region. Bounded by region count.
func snapshotRegions(read regionReader, regions []windowsMemRegion) map[uintptr]regionFingerprint {
	out := make(map[uintptr]regionFingerprint, min(len(regions), maxRegionFingerprints))
	for _, r := range regions[:min(len(regions), maxRegionFingerprints)] {
		out[r.base] = regionFingerprint{base: r.base, size: r.size, fp: fingerprintRegion(read, r.base, r.size)}
	}
	return out
}

// prioritizeChangedRegions reorders regions: changed-size, changed-fingerprint
// and new regions first (in original order), unchanged after. The returned
// marks snapshot the CURRENT state for the next comparison.
func prioritizeChangedRegions(prev map[uintptr]regionFingerprint, read regionReader, regions []windowsMemRegion) (ordered []windowsMemRegion, changedCount int, current map[uintptr]regionFingerprint) {
	current = snapshotRegions(read, regions)
	if prev == nil {
		return regions, 0, current
	}
	changed := make([]windowsMemRegion, 0, len(regions))
	same := make([]windowsMemRegion, 0, len(regions))
	for _, r := range regions {
		p, ok := prev[r.base]
		if !ok || p.fp == 0 || current[r.base].fp == 0 || p.size != r.size || p.fp != current[r.base].fp {
			changed = append(changed, r)
		} else {
			same = append(same, r)
		}
	}
	return append(changed, same...), len(changed), current
}

// windowsDiffScanEnabled reports whether change-priority ordering is active
// (default on; WECHAT_CLI_DIFF_SCAN=0 disables).
func windowsDiffScanEnabled() bool {
	switch firstEnv("WECHAT_CLI_DIFF_SCAN") {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

// windowsProcessRegionReader adapts ReadProcessMemory to regionReader.
func windowsProcessRegionReader(h uintptr) regionReader {
	return func(base uintptr, n int) ([]byte, error) {
		buf := make([]byte, n)
		var got uintptr
		r, _, _ := procReadProcessMemory.Call(h, base, uintptr(unsafe.Pointer(&buf[0])), uintptr(n), uintptr(unsafe.Pointer(&got)))
		if r == 0 || got == 0 {
			return nil, errRegionUnreadable
		}
		return buf[:got], nil
	}
}

var errRegionUnreadable = errors.New("region unreadable")
