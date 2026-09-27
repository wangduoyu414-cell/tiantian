package wxkey

import (
	"debug/macho"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
)

// Mach-O relocation for the macOS WeChat build (arm64), mirroring the PE
// strategy: anchors are string constants ("MMV1", "x'%s'", "com.Tencent.WCDB…")
// in __cstring/__const; code references them via ADRP+ADD pairs in __text;
// the enclosing function start comes from the symbol table when present.
//
// This file is pure parsing (debug/macho + two instruction decoders), so it
// runs and tests anywhere — no Mac required for the static layer. Real-device
// capture remains a separate, explicitly pending capability.

// MachOCandidate is one auto-derived reference site for an anchor.
type MachOCandidate struct {
	Anchor        string `json:"anchor"`
	StringAddr    uint64 `json:"string_addr"`
	StringSection string `json:"string_section"`
	RefAddr       uint64 `json:"ref_addr"` // address of the ADRP instruction
	FuncStart     uint64 `json:"func_start,omitempty"`
	Symbol        string `json:"symbol,omitempty"`
}

// MachOReport is the relocation result for one Mach-O image.
type MachOReport struct {
	Path       string           `json:"path"`
	Arch       string           `json:"arch"`
	Candidates []MachOCandidate `json:"candidates"`
	AnchorHits map[string]int   `json:"anchor_hits"`
	// HasSymbols reports whether a symbol table was available for function
	// attribution. Without one, candidates carry RefAddr only.
	HasSymbols bool `json:"has_symbols"`
}

// RelocateHookPointsMachO statically analyzes the Mach-O at path.
func RelocateHookPointsMachO(path string, anchors []RelocateAnchor) (*MachOReport, error) {
	if len(anchors) == 0 {
		anchors = DefaultRelocateAnchors()
	}
	f, err := macho.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open Mach-O: %w", err)
	}
	defer f.Close()

	rep := &MachOReport{
		Path:       path,
		Arch:       fmt.Sprintf("%v", f.Cpu),
		AnchorHits: map[string]int{},
		HasSymbols: f.Symtab != nil && len(f.Symtab.Syms) > 0,
	}

	// Anchor sections: string literals live in __cstring (or __TEXT.__const).
	type secData struct {
		name string
		addr uint64
		data []byte
	}
	var anchorSecs []secData
	for _, name := range []string{"__cstring", "__const"} {
		if s := f.Section(name); s != nil {
			if b, err := s.Data(); err == nil {
				anchorSecs = append(anchorSecs, secData{name: name, addr: s.Addr, data: b})
			}
		}
	}

	textSec := f.Section("__text")
	if textSec == nil {
		return nil, fmt.Errorf("no __text section in %s", path)
	}
	text, err := textSec.Data()
	if err != nil {
		return nil, fmt.Errorf("read __text: %w", err)
	}

	// Pre-index symbols (sorted by value) for nearest-lower lookup.
	var syms []macho.Symbol
	if f.Symtab != nil {
		syms = append(syms, f.Symtab.Syms...)
		sort.Slice(syms, func(i, j int) bool { return syms[i].Value < syms[j].Value })
	}

	for _, a := range anchors {
		if len(a.Pattern) == 0 {
			continue // empty patterns are rejected, as on the PE side
		}
		hits := 0
		for _, sec := range anchorSecs {
			for off := 0; off+len(a.Pattern) <= len(sec.data); {
				idx := indexOfBytes(sec.data[off:], a.Pattern)
				if idx < 0 {
					break
				}
				hits++
				strAddr := sec.addr + uint64(off) + uint64(idx)
				for _, ref := range findADRPADDRefs(text, textSec.Addr, strAddr) {
					cand := MachOCandidate{
						Anchor:        a.Name,
						StringAddr:    strAddr,
						StringSection: sec.name,
						RefAddr:       ref,
					}
					if s := nearestSymbolAtOrBelow(syms, ref); s != nil {
						cand.FuncStart = s.Value
						cand.Symbol = s.Name
					}
					rep.Candidates = append(rep.Candidates, cand)
				}
				off += idx + 1
			}
		}
		rep.AnchorHits[a.Name] = hits
	}

	sort.Slice(rep.Candidates, func(i, j int) bool {
		if rep.Candidates[i].Anchor != rep.Candidates[j].Anchor {
			return rep.Candidates[i].Anchor < rep.Candidates[j].Anchor
		}
		return rep.Candidates[i].RefAddr < rep.Candidates[j].RefAddr
	})
	return rep, nil
}

// ---------------------------------------------------------------------------
// A64 ADRP / ADD decoders
// ---------------------------------------------------------------------------

// isADRP reports whether w is an ADRP instruction (op[31]=1, bits[28:24]=10000).
func isADRP(w uint32) bool {
	return w&0x80000000 != 0 && (w>>24)&0x1F == 0x10
}

// adrpPage returns the 4KB-aligned page address the ADRP at pc targets.
func adrpPage(pc uint64, w uint32) uint64 {
	immlo := int64((w >> 29) & 0x3)
	immhi := int64((w >> 5) & 0x7FFFF)
	imm := (immhi << 2) | immlo
	// sign-extend 21 bits
	if imm&(1<<20) != 0 {
		imm -= 1 << 21
	}
	return (pc &^ 0xFFF) + uint64(imm<<12)
}

// isADDImm64 reports whether w is an ADD (immediate, 64-bit, shift=0) and
// returns Rd, Rn, imm12.
func isADDImm64(w uint32) (rd, rn, imm uint32, ok bool) {
	if w>>31 != 1 || (w>>24)&0x3F != 0x11 || (w>>22)&0x3 != 0 {
		return 0, 0, 0, false
	}
	return w & 0x1F, (w >> 5) & 0x1F, (w >> 10) & 0xFFF, true
}

// findADRPADDRefs scans __text for ADRP Xn, page + ADD Xn, Xn, imm12 pairs
// whose final address equals target. Returns the ADRP instruction addresses.
func findADRPADDRefs(text []byte, textAddr, target uint64) []uint64 {
	var out []uint64
	for i := 0; i+8 <= len(text); i += 4 {
		w0 := binary.LittleEndian.Uint32(text[i:])
		if !isADRP(w0) {
			continue
		}
		rd0 := w0 & 0x1F
		pc := textAddr + uint64(i)
		w1 := binary.LittleEndian.Uint32(text[i+4:])
		rd1, rn1, imm12, ok := isADDImm64(w1)
		if !ok || rd1 != rd0 || rn1 != rd0 {
			continue
		}
		if adrpPage(pc, w0)+uint64(imm12) == target {
			out = append(out, pc)
		}
	}
	return out
}

// nearestSymbolAtOrBelow returns the nearest symbol at or below addr.
func nearestSymbolAtOrBelow(syms []macho.Symbol, addr uint64) *macho.Symbol {
	i := sort.Search(len(syms), func(i int) bool { return syms[i].Value > addr })
	if i == 0 {
		return nil
	}
	s := syms[i-1]
	// Skip non-function-ish symbols (section/absolute markers).
	if strings.HasPrefix(s.Name, "ltmp") || s.Name == "" {
		return nil
	}
	return &s
}

// indexOfBytes is bytes.Index with a local alias to keep this file small.
func indexOfBytes(haystack, needle []byte) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
