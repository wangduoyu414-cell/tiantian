package wxkey

import (
	"encoding/binary"
	"testing"
)

// When the preferred section already yields hits, the fallback must not run:
// the same pattern living in two sections must not produce duplicated hits.
func TestFindAnchorRVAsPreferredHitSkipsFallback(t *testing.T) {
	data := []byte("MMV1----MMV1----")
	secs := []peSection{
		{name: ".rdata", va: 0x1000, rawOff: 0, rawSize: 8},
		{name: ".rdata2", va: 0x2000, rawOff: 8, rawSize: 8},
	}
	hits := findAnchorRVAs(data, secs, RelocateAnchor{Name: "MMV1", Pattern: []byte("MMV1")})
	if len(hits) != 1 || hits[0].rva != 0x1000 || hits[0].section != ".rdata" {
		t.Fatalf("preferred hit should short-circuit fallback; got %#v", hits)
	}
}

// ConstDword anchors prefer .text over .rdata.
func TestFindAnchorRVAsConstDwordPrefersText(t *testing.T) {
	kdf := make([]byte, 4)
	binary.LittleEndian.PutUint32(kdf, 256000)
	data := append(append([]byte{}, kdf...), make([]byte, 4)...)
	data = append(data, kdf...)
	data = append(data, make([]byte, 4)...)
	secs := []peSection{
		{name: ".rdata", va: 0x1000, rawOff: 0, rawSize: 8},
		{name: ".text", va: 0x2000, rawOff: 8, rawSize: 8},
	}
	hits := findAnchorRVAs(data, secs, RelocateAnchor{Name: "kdf", Pattern: kdf, IsConstDword: true})
	if len(hits) != 1 || hits[0].section != ".text" || hits[0].rva != 0x2000 {
		t.Fatalf("const dword should hit in .text; got %#v", hits)
	}
}

// Degenerate .pdata entries (zero padding, end<=begin) are skipped without
// discarding the valid entries that follow them.
func TestParsePdataSkipsDegenerateEntries(t *testing.T) {
	data := make([]byte, 36)
	// entry 0: all zero (padding)
	// entry 1: end < begin (corrupt)
	binary.LittleEndian.PutUint32(data[12:16], 0x2000)
	binary.LittleEndian.PutUint32(data[16:20], 0x1000)
	// entry 2: valid
	binary.LittleEndian.PutUint32(data[24:28], 0x1000)
	binary.LittleEndian.PutUint32(data[28:32], 0x1100)
	secs := []peSection{{name: ".pdata", va: 0x3000, rawOff: 0, rawSize: 36}}

	funcs := parsePdata(data, secs)
	if len(funcs) != 1 || funcs[0].BeginRVA != 0x1000 || funcs[0].EndRVA != 0x1100 {
		t.Fatalf("parsePdata = %#v, want single valid entry", funcs)
	}
	if got := findFuncEntryPdata(funcs, 0x1050); got != 0x1000 {
		t.Fatalf("findFuncEntryPdata = 0x%x, want 0x1000", got)
	}
}

// A section whose declared raw range runs past EOF must clamp, not panic
// (malformed/truncated PE input).
func TestFindPatternInSectionToleratesTruncatedRaw(t *testing.T) {
	data := []byte("MMV1") // 4 bytes on disk
	sec := peSection{name: ".rdata", va: 0x1000, rawOff: 0, rawSize: 4096}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("truncated section panicked: %v", r)
		}
	}()
	hits := findPatternInSection(data, sec, RelocateAnchor{Name: "MMV1", Pattern: []byte("MMV1")})
	if len(hits) != 1 || hits[0].rva != 0x1000 {
		t.Fatalf("hits = %#v, want one hit at 0x1000", hits)
	}
}

// LEA targets that land outside every section (or overflow) are artefacts of
// matching the opcode pattern mid-instruction; they must be dropped.
func TestBuildLEAMapDropsUnmappedTargets(t *testing.T) {
	// .text: 48 8D 05 <disp->0x2000>  | 48 8D 05 <disp->outside> | 48 8D 05 <disp->negative>
	text := peSection{name: ".text", va: 0x1000, rawOff: 0, rawSize: 21}
	rdata := peSection{name: ".rdata", va: 0x2000, rawOff: 32, rawSize: 16}
	secs := []peSection{text, rdata}

	buf := make([]byte, 48)
	// insn 1 at file 0 (RVA 0x1000): target = 0x1000 + 7 + disp = 0x2000
	buf[0], buf[1], buf[2] = 0x48, 0x8D, 0x05
	binary.LittleEndian.PutUint32(buf[3:7], uint32(int32(0x2000-0x1000-7)))
	// insn 2 at file 7 (RVA 0x1007): target far past every section
	buf[7], buf[8], buf[9] = 0x48, 0x8D, 0x05
	binary.LittleEndian.PutUint32(buf[10:14], uint32(int32(0x7FFF0000)))
	// insn 3 at file 14 (RVA 0x100E): negative displacement outside the image
	buf[14], buf[15], buf[16] = 0x48, 0x8D, 0x05
	negDisp := int32(-0x3000)
	binary.LittleEndian.PutUint32(buf[17:21], uint32(negDisp))

	m := buildLEAMap(buf, text, secs)
	if len(m) != 1 {
		t.Fatalf("expected exactly 1 mapped target, got %d keys: %#v", len(m), m)
	}
	leas, ok := m[0x2000]
	if !ok || len(leas) != 1 || leas[0] != 0x1000 {
		t.Fatalf("leaMap[0x2000] = %#v (ok=%v), want [0x1000]", leas, ok)
	}
}
