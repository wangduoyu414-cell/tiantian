package wxkey

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// This file implements version-adaptive hook-point relocation for Weixin.dll.
//
// WeChat rebuilds Weixin.dll every release, so any hardcoded RVA (e.g. the
// MMV1 codec-config function) goes stale each update. The codec anchors,
// however, survive: WCDB still references the strings "MMV1", "x'%s'", and
// "com.Tencent.WCDB.Config.Cipher", and the KDF iteration constant 256000 is
// compiled into .text. RelocateHookPoints finds those anchor strings in the
// file, scans .text once for all RIP-relative LEA instructions that reference
// them, then resolves the enclosing function entry via the PE .pdata exception
// directory — yielding candidate RVAs that a Frida/native hook can target,
// without any manual reverse work.
//
// The technique is pure static PE parsing (no process access, no debug/pe
// disassembly beyond the tiny LEA pattern), so it is testable against golden
// RVAs recovered from a live capture.

// RelocateAnchor describes one byte-string anchor to locate in the image.
type RelocateAnchor struct {
	Name    string `json:"name"`
	Pattern []byte `json:"-"`
	// PatternHex is the hex form of Pattern, for JSON output.
	PatternHex string `json:"pattern_hex"`
	// IsConstDword marks Pattern as a 4-byte little-endian constant (e.g. the
	// KDF iteration count) rather than a NUL-terminated string.
	IsConstDword bool `json:"is_const_dword,omitempty"`
}

// RelocateCandidate is one auto-derived function entry for an anchor.
type RelocateCandidate struct {
	Anchor        string `json:"anchor"`
	StringRVA     uint32 `json:"string_rva"`
	StringSection string `json:"string_section"`
	LeaRVA        uint32 `json:"lea_rva"`
	FuncEntryRVA  uint32 `json:"func_entry_rva"`
}

// MarshalJSON renders RVAs as 0x-prefixed hex for readability.
func (c RelocateCandidate) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"anchor":         c.Anchor,
		"string_rva":     hexRVA(c.StringRVA),
		"string_section": c.StringSection,
		"lea_rva":        hexRVA(c.LeaRVA),
		"func_entry_rva": hexRVA(c.FuncEntryRVA),
	})
}

func hexRVA(v uint32) string { return fmt.Sprintf("0x%x", v) }

// RelocateReport is the full result of relocating every anchor in one image.
type RelocateReport struct {
	Path        string              `json:"path"`
	Size        int64               `json:"size"`
	ImageBase   uint64              `json:"image_base"`
	SizeOfImage uint32              `json:"size_of_image"`
	Candidates  []RelocateCandidate `json:"candidates"`
	// AnchorHits records how many times each anchor string appears, useful for
	// diagnosing a build where Tencent renames/encrypts the anchors.
	AnchorHits map[string]int `json:"anchor_hits"`
}

// MarshalJSON renders the base/offset fields as 0x-prefixed hex.
func (r *RelocateReport) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"path":          r.Path,
		"size":          r.Size,
		"image_base":    fmt.Sprintf("0x%x", r.ImageBase),
		"size_of_image": fmt.Sprintf("0x%x", r.SizeOfImage),
		"candidates":    r.Candidates,
		"anchor_hits":   r.AnchorHits,
	})
}

// DefaultRelocateAnchors returns the anchors used for WeChat 4.1.x hook
// relocation. String anchors are the primary signal; the KDF constant is a
// fallback when Tencent strips/encrypts the strings.
func DefaultRelocateAnchors() []RelocateAnchor {
	str := func(name, s string) RelocateAnchor {
		return RelocateAnchor{Name: name, Pattern: []byte(s), PatternHex: hexOf([]byte(s))}
	}
	kdf := make([]byte, 4)
	binary.LittleEndian.PutUint32(kdf, 256000)
	return []RelocateAnchor{
		str("MMV1", "MMV1"),
		str("x'%s'", "x'%s'"),
		str("Config.Cipher", "com.Tencent.WCDB.Config.Cipher"),
		{Name: "kdf-256000", Pattern: kdf, PatternHex: hexOf(kdf), IsConstDword: true},
	}
}

func hexOf(b []byte) string {
	const hexdig = "0123456789abcdef"
	var sb strings.Builder
	for _, c := range b {
		sb.WriteByte(hexdig[c>>4])
		sb.WriteByte(hexdig[c&0xf])
	}
	return sb.String()
}

// peSection mirrors the parts of a PE section we need.
type peSection struct {
	name    string
	va      uint32 // virtual address (RVA of section start)
	vsize   uint32
	rawOff  uint32
	rawSize uint32
}

// ---------------------------------------------------------------------------
// .pdata exception directory — exact function boundaries
// ---------------------------------------------------------------------------

// runtimeFunction mirrors one RUNTIME_FUNCTION entry in the PE .pdata section.
// Every non-leaf x64 function has an entry, giving exact [BeginRVA, EndRVA)
// bounds without heuristic backward scanning.
type runtimeFunction struct {
	BeginRVA uint32
	EndRVA   uint32
}

// parsePdata reads the .pdata section and returns the RUNTIME_FUNCTION table
// sorted by BeginRVA. This gives exact function boundaries for every non-leaf
// function, replacing the heuristic backward scan.
func parsePdata(data []byte, secs []peSection) []runtimeFunction {
	pdata, ok := sectionByName(secs, ".pdata")
	if !ok {
		return nil
	}
	body := sectionBytes(data, pdata)
	var funcs []runtimeFunction
	for i := 0; i+12 <= len(body); i += 12 {
		begin := binary.LittleEndian.Uint32(body[i:])
		end := binary.LittleEndian.Uint32(body[i+4:])
		if begin == 0 || end <= begin {
			// Padding or a corrupt entry; real entries span [begin, end).
			// Skip it and keep scanning the rest of the table.
			continue
		}
		funcs = append(funcs, runtimeFunction{BeginRVA: begin, EndRVA: end})
	}
	sort.Slice(funcs, func(i, j int) bool { return funcs[i].BeginRVA < funcs[j].BeginRVA })
	return funcs
}

// findFuncEntryPdata returns the BeginRVA of the function containing rva,
// or 0 if not found. Binary search over the sorted .pdata table.
func findFuncEntryPdata(funcs []runtimeFunction, rva uint32) uint32 {
	i := sort.Search(len(funcs), func(i int) bool { return funcs[i].BeginRVA > rva })
	if i > 0 {
		f := funcs[i-1]
		if rva >= f.BeginRVA && rva < f.EndRVA {
			return f.BeginRVA
		}
	}
	return 0
}

// ---------------------------------------------------------------------------
// Single-pass LEA cross-reference map
// ---------------------------------------------------------------------------

// buildLEAMap scans .text once and collects every RIP-relative LEA into a
// map from target RVA to the LEA instruction RVAs that reference it. This
// replaces the old per-anchor full-.text rescan. Targets that overflow 32
// bits or do not land inside any known section are discarded: they are
// artefacts of matching the LEA pattern at a non-instruction boundary.
func buildLEAMap(data []byte, text peSection, secs []peSection) map[uint32][]uint32 {
	body := sectionBytes(data, text)
	m := make(map[uint32][]uint32)
	for i := 0; i+7 <= len(body); i++ {
		b0 := body[i]
		if b0 != 0x48 && b0 != 0x4C {
			continue
		}
		if body[i+1] != 0x8D {
			continue
		}
		modrm := body[i+2]
		if modrm&0xC7 != 0x05 {
			continue
		}
		disp := int32(binary.LittleEndian.Uint32(body[i+3 : i+7]))
		insnRVA := text.va + uint32(i)
		t := int64(insnRVA) + 7 + int64(disp)
		if t < 0 || t > 0xFFFFFFFF {
			continue
		}
		target := uint32(t)
		if !rvaInAnySection(secs, target) {
			continue
		}
		m[target] = append(m[target], insnRVA)
	}
	return m
}

// rvaInAnySection reports whether rva falls inside any section's virtual
// range (virtual size when present, raw size otherwise). Overflow-safe.
func rvaInAnySection(secs []peSection, rva uint32) bool {
	for _, s := range secs {
		sz := s.vsize
		if sz == 0 {
			sz = s.rawSize
		}
		if sz == 0 {
			continue
		}
		lo := uint64(s.va)
		hi := lo + uint64(sz)
		if uint64(rva) >= lo && uint64(rva) < hi {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Main entry point
// ---------------------------------------------------------------------------

// RelocateHookPoints statically analyzes the PE at path and returns candidate
// function-entry RVAs for each anchor. It reads the whole image into memory
// (Weixin.dll is ~190MB); callers should pass a local file path.
func RelocateHookPoints(path string, anchors []RelocateAnchor) (*RelocateReport, error) {
	if len(anchors) == 0 {
		anchors = DefaultRelocateAnchors()
	}
	f, err := pe.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open PE: %w", err)
	}
	defer f.Close()

	oh, ok := f.OptionalHeader.(*pe.OptionalHeader64)
	if !ok {
		return nil, fmt.Errorf("not a 64-bit PE (Weixin.dll is x64)")
	}

	fileBytes, err := readFileBytes(path)
	if err != nil {
		return nil, err
	}

	secs := make([]peSection, 0, len(f.Sections))
	for _, s := range f.Sections {
		secs = append(secs, peSection{
			name:    strings.TrimRight(s.Name, "\x00"),
			va:      s.VirtualAddress,
			vsize:   s.VirtualSize,
			rawOff:  s.Offset,
			rawSize: s.Size,
		})
	}

	rep := &RelocateReport{
		Path:        path,
		Size:        int64(len(fileBytes)),
		ImageBase:   uint64(oh.ImageBase),
		SizeOfImage: oh.SizeOfImage,
		AnchorHits:  map[string]int{},
	}

	// Single pass over .text: build target→[]leaRVA map shared by all anchors.
	var leaMap map[uint32][]uint32
	if text, ok := sectionByName(secs, ".text"); ok {
		leaMap = buildLEAMap(fileBytes, text, secs)
	}

	// Parse .pdata exception directory for exact function entries.
	pdataFuncs := parsePdata(fileBytes, secs)

	for _, a := range anchors {
		rvas := findAnchorRVAs(fileBytes, secs, a)
		rep.AnchorHits[a.Name] = len(rvas)
		limit := len(rvas)
		if a.IsConstDword && limit > 8 {
			limit = 8
		}
		for _, sv := range rvas[:limit] {
			for _, leaRVA := range leaMap[sv.rva] {
				entry := findFuncEntryPdata(pdataFuncs, leaRVA)
				if entry == 0 {
					// Fallback to heuristic when .pdata lacks this function
					// (leaf functions may be absent from the table).
					entry = findFuncEntry(fileBytes, secs, leaRVA)
				}
				rep.Candidates = append(rep.Candidates, RelocateCandidate{
					Anchor:        a.Name,
					StringRVA:     sv.rva,
					StringSection: sv.section,
					LeaRVA:        leaRVA,
					FuncEntryRVA:  entry,
				})
			}
		}
	}

	sort.Slice(rep.Candidates, func(i, j int) bool {
		if rep.Candidates[i].Anchor != rep.Candidates[j].Anchor {
			return rep.Candidates[i].Anchor < rep.Candidates[j].Anchor
		}
		return rep.Candidates[i].FuncEntryRVA < rep.Candidates[j].FuncEntryRVA
	})
	return rep, nil
}

// ---------------------------------------------------------------------------
// Anchor search
// ---------------------------------------------------------------------------

type anchorHit struct {
	rva     uint32
	section string
}

// findAnchorRVAs locates every occurrence of the anchor pattern. String
// anchors are searched first in .rdata (where the linker places string
// literals) to avoid false hits from instruction immediates in .text, and
// ConstDword anchors first in .text (where instruction immediates live).
// When the preferred section exists but yields no hit, the remaining
// sections are searched too: a linker may legitimately move literals into
// sections such as .rdata2. Empty patterns are rejected outright.
func findAnchorRVAs(data []byte, secs []peSection, a RelocateAnchor) []anchorHit {
	if len(a.Pattern) == 0 {
		// An empty pattern matches at every offset and previously caused a
		// slice-bounds panic; reject it instead.
		return nil
	}
	preferred := ".rdata"
	if a.IsConstDword {
		preferred = ".text"
	}
	if sec, ok := sectionByName(secs, preferred); ok {
		if hits := findPatternInSection(data, sec, a); len(hits) > 0 {
			return hits
		}
		return findPatternInOtherSections(data, secs, a, preferred)
	}
	return findPatternAllSections(data, secs, a)
}

// findPatternInOtherSections searches every section except skip.
func findPatternInOtherSections(data []byte, secs []peSection, a RelocateAnchor, skip string) []anchorHit {
	var out []anchorHit
	for _, s := range secs {
		if s.name == skip {
			continue
		}
		out = append(out, findPatternInSection(data, s, a)...)
	}
	return out
}

// findPatternInSection searches a single section for the anchor pattern.
func findPatternInSection(data []byte, sec peSection, a RelocateAnchor) []anchorHit {
	body := sectionBytes(data, sec)
	var out []anchorHit
	for offset := 0; offset+len(a.Pattern) <= len(body); {
		idx := bytes.Index(body[offset:], a.Pattern)
		if idx < 0 {
			break
		}
		rva := sec.va + uint32(offset+idx)
		out = append(out, anchorHit{rva: rva, section: sec.name})
		offset += idx + 1
	}
	return out
}

// findPatternAllSections searches the entire file for the anchor pattern.
func findPatternAllSections(data []byte, secs []peSection, a RelocateAnchor) []anchorHit {
	var out []anchorHit
	for offset := 0; offset+len(a.Pattern) <= len(data); {
		idx := bytes.Index(data[offset:], a.Pattern)
		if idx < 0 {
			break
		}
		fileOff := uint32(offset + idx)
		if rva, sec := offToRVA(fileOff, secs); sec != "" {
			out = append(out, anchorHit{rva: rva, section: sec})
		}
		offset += idx + 1
	}
	return out
}

// ---------------------------------------------------------------------------
// Heuristic function-entry fallback (used when .pdata is absent)
// ---------------------------------------------------------------------------

// findFuncEntry walks backwards from insnRVA in the .text file bytes to the
// start of the enclosing function: the byte right after a RET (0xC3) or INT3
// (0xCC) whose own first byte is a plausible prologue byte.
//
// This is a fallback for when .pdata is missing or incomplete. Prefer
// findFuncEntryPdata whenever the exception directory is available.
func findFuncEntry(data []byte, secs []peSection, insnRVA uint32) uint32 {
	text, ok := sectionByName(secs, ".text")
	if !ok {
		return 0
	}
	o := int(text.rawOff + (insnRVA - text.va))
	if o <= 0 || o > len(data) {
		return 0
	}
	const maxBack = 0x2000
	for back := 1; back < maxBack && o-back-1 >= 0; back++ {
		p := o - back
		prev := data[p-1]
		if prev != 0xC3 && prev != 0xCC {
			continue
		}
		if isPrologueByte(data[p]) {
			return text.va + uint32(p) - text.rawOff
		}
	}
	return 0
}

func isPrologueByte(b byte) bool {
	switch b {
	case 0x40, 0x41, 0x48, 0x55, 0x53, 0x56, 0x57, 0x4C:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Section utilities
// ---------------------------------------------------------------------------

func sectionByName(secs []peSection, name string) (peSection, bool) {
	for _, s := range secs {
		if s.name == name {
			return s, true
		}
	}
	return peSection{}, false
}

func sectionBytes(data []byte, s peSection) []byte {
	if s.rawSize == 0 || uint64(s.rawOff) >= uint64(len(data)) {
		return nil
	}
	end := uint64(s.rawOff) + uint64(s.rawSize)
	if end > uint64(len(data)) {
		end = uint64(len(data))
	}
	return data[s.rawOff:end]
}

// offToRVA maps a file offset to (RVA, sectionName). Returns "" if unmapped.
func offToRVA(off uint32, secs []peSection) (uint32, string) {
	for _, s := range secs {
		if off >= s.rawOff && off < s.rawOff+s.rawSize {
			return s.va + (off - s.rawOff), s.name
		}
	}
	return 0, ""
}

// readFileBytes loads an entire file into memory.
func readFileBytes(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return b, nil
}

// DefaultWeixinDLLPath discovers the selected installation. This is an
// analysis candidate, not proof of which module a running process loaded.
func DefaultWeixinDLLPath() (string, error) {
	return defaultWeixinDLLPath()
}

// parseDottedVersion parses "4.1.12.55" into []int{4,1,12,55}.
func parseDottedVersion(s string) ([]int, bool) {
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return nil, false
	}
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			return nil, false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return nil, false
			}
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, false
		}
		out = append(out, n)
	}
	return out, true
}

func compareVersions(a, b []int) int {
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		ai, bi := 0, 0
		if i < len(a) {
			ai = a[i]
		}
		if i < len(b) {
			bi = b[i]
		}
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
	}
	return 0
}
