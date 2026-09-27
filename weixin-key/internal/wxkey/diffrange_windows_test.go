//go:build windows

package wxkey

import (
	"errors"
	"testing"
)

// fakeMemory is a synthetic region store for the diff scanner.
type fakeMemory struct {
	pages map[uintptr][]byte
}

func (m *fakeMemory) reader() regionReader {
	return func(base uintptr, n int) ([]byte, error) {
		b, ok := m.pages[base]
		if !ok {
			return nil, errors.New("unmapped")
		}
		if n > len(b) {
			n = len(b)
		}
		return b[:n], nil
	}
}

func TestPrioritizeChangedRegions(t *testing.T) {
	// Regions spaced so head/tail samples never overlap a neighbor region.
	regions := []windowsMemRegion{{base: 0x10000, size: 8192}, {base: 0x20000, size: 8192}, {base: 0x30000, size: 8192}}
	mem := &fakeMemory{pages: map[uintptr][]byte{
		0x10000: []byte("AAAAAAAA"),
		0x20000: []byte("BBBBBBBB"),
		0x30000: []byte("CCCCCCCC"),
	}}

	// First snapshot: no change priority.
	ordered, changed, marks := prioritizeChangedRegions(nil, mem.reader(), regions)
	if changed != 0 || len(ordered) != 3 {
		t.Fatalf("first pass: changed=%d ordered=%d", changed, len(ordered))
	}

	// Mutate region 0x20000 only.
	mem.pages[0x20000] = []byte("XXXXYYYY")
	ordered, changed, _ = prioritizeChangedRegions(marks, mem.reader(), regions)
	if changed != 1 {
		t.Fatalf("changed = %d, want 1", changed)
	}
	if ordered[0].base != 0x20000 {
		t.Fatalf("changed region not prioritized: first=%x", ordered[0].base)
	}
	// Unchanged regions are still present (priority, not exclusion).
	if len(ordered) != 3 {
		t.Fatalf("ordered = %d, want 3 (unchanged kept)", len(ordered))
	}

	// Size change counts as changed even with same content prefix.
	regions[1].size = 16384
	mem.pages[0x20000] = append(mem.pages[0x20000], []byte("ZZZZZZZZ")...)
	_, changed, _ = prioritizeChangedRegions(marks, mem.reader(), regions)
	if changed != 1 {
		t.Fatalf("size change: changed = %d, want 1", changed)
	}
}

func TestDiffScanToggle(t *testing.T) {
	if !windowsDiffScanEnabled() {
		t.Fatal("default should be on")
	}
	t.Setenv("WECHAT_CLI_DIFF_SCAN", "0")
	if windowsDiffScanEnabled() {
		t.Fatal("WECHAT_CLI_DIFF_SCAN=0 should disable")
	}
}

func TestRegionFingerprintBudgetAndSmallRange(t *testing.T) {
	regions := make([]windowsMemRegion, maxRegionFingerprints+12)
	for i := range regions {
		regions[i] = windowsMemRegion{base: uintptr(i + 1), size: 7}
	}
	calls := 0
	read := func(base uintptr, n int) ([]byte, error) {
		calls++
		if n > 7 {
			t.Fatal("sample crossed the small region boundary")
		}
		return make([]byte, n), nil
	}
	marks := snapshotRegions(read, regions)
	if calls != maxRegionFingerprints || len(marks) != maxRegionFingerprints {
		t.Fatal("fingerprint budget not enforced")
	}
	ordered, _, _ := prioritizeChangedRegions(marks, read, regions)
	if len(ordered) != len(regions) {
		t.Fatal("unobserved regions excluded")
	}
}
