package wxkey

import "testing"

// Regression probes for the hook-point relocation rework. These were first
// demonstrated as failing owner probes against the uncommitted relocate.go
// (evidence: D:\weixinpojie\.owner-supervision\relocate-probes\result.txt);
// they now live in the repo as permanent regression tests.

// An empty anchor pattern must be rejected with zero hits instead of
// panicking with a slice-bounds error inside the scan loop.
func TestFindAnchorRVAsRejectsEmptyPattern(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("empty pattern panicked instead of being rejected: %v", r)
		}
	}()
	hits := findAnchorRVAs([]byte("--------"), []peSection{{name: ".rdata", va: 0x1000, rawOff: 0, rawSize: 8}}, RelocateAnchor{Name: "empty"})
	if len(hits) != 0 {
		t.Fatalf("empty anchor returned %d hits, want 0", len(hits))
	}
}

// When the preferred section (.rdata for string anchors) exists but does not
// contain the anchor, the scan must fall back to the remaining sections so a
// linker-relocated anchor (e.g. in .rdata2) is still found.
func TestFindAnchorRVAsFallsBackBeyondPreferredSection(t *testing.T) {
	data := []byte("--------MMV1----")
	secs := []peSection{
		{name: ".rdata", va: 0x1000, rawOff: 0, rawSize: 8},
		{name: ".rdata2", va: 0x2000, rawOff: 8, rawSize: 8},
	}
	hits := findAnchorRVAs(data, secs, RelocateAnchor{Name: "MMV1", Pattern: []byte("MMV1")})
	if len(hits) != 1 || hits[0].rva != 0x2000 {
		t.Fatalf("preferred-section miss should retain nonpreferred-section fallback; got %#v", hits)
	}
}
