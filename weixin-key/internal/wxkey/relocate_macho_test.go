package wxkey

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// encodeADRP builds the A64 ADRP instruction for (pc -> targetPage).
func encodeADRP(pc, targetPage uint64, rd uint32) uint32 {
	imm := int64(targetPage) - int64(pc&^0xFFF)
	imm >>= 12
	imm21 := uint32(imm) & 0x1FFFFF
	return 0x80000000 | ((imm21 & 0x3) << 29) | (0x10 << 24) | ((imm21 >> 2) << 5) | rd
}

// encodeADDImm builds ADD (immediate, 64-bit, shift=0).
func encodeADDImm(rd, rn, imm12 uint32) uint32 {
	return 0x80000000 | (0x11 << 24) | (imm12 << 10) | (rn << 5) | rd
}

func TestADRPAddRoundTrip(t *testing.T) {
	pc := uint64(0x100000400)
	target := uint64(0x100002008)
	w0 := encodeADRP(pc, target&^0xFFF, 8)
	if !isADRP(w0) {
		t.Fatal("encoded ADRP not recognized")
	}
	if got := adrpPage(pc, w0); got != target&^0xFFF {
		t.Fatalf("adrpPage = 0x%x, want 0x%x", got, target&^0xFFF)
	}
	w1 := encodeADDImm(8, 8, uint32(target&0xFFF))
	rd, rn, imm, ok := isADDImm64(w1)
	if !ok || rd != 8 || rn != 8 || imm != uint32(target&0xFFF) {
		t.Fatalf("add decode: rd=%d rn=%d imm=%d ok=%v", rd, rn, imm, ok)
	}
	// Negative page offset (target page BELOW pc).
	pc2 := uint64(0x100008000)
	tp2 := uint64(0x100000000)
	w0n := encodeADRP(pc2, tp2, 3)
	if got := adrpPage(pc2, w0n); got != tp2 {
		t.Fatalf("negative adrpPage = 0x%x, want 0x%x", got, tp2)
	}
}

// buildSyntheticMachO assembles a minimal arm64 Mach-O: one __cstring section
// carrying the anchors, one __text section with an adrp+add pair per anchor,
// and a symbol table marking the function start.
func buildSyntheticMachO(t *testing.T, anchors []string, withSymbols bool) string {
	t.Helper()

	const vmaddr = 0x100000000
	// File layout.
	const textFileOff = 0x400
	const cstrFileOff = 0x2000
	const symFileOff = 0x3000

	textAddr := uint64(vmaddr + textFileOff)
	cstrAddr := uint64(vmaddr + cstrFileOff)

	// Build __cstring content: anchors NUL-terminated back to back.
	var cstr bytes.Buffer
	anchorOffsets := map[string]uint64{}
	for _, a := range anchors {
		anchorOffsets[a] = cstrAddr + uint64(cstr.Len())
		cstr.WriteString(a)
		cstr.WriteByte(0)
	}

	// Build __text: one adrp+add pair per anchor + a ret.
	var text bytes.Buffer
	funcStart := textAddr
	for _, a := range anchors {
		pc := textAddr + uint64(text.Len())
		target := anchorOffsets[a]
		binary.Write(&text, binary.LittleEndian, encodeADRP(pc, target&^0xFFF, 8))
		binary.Write(&text, binary.LittleEndian, encodeADDImm(8, 8, uint32(target&0xFFF)))
		binary.Write(&text, binary.LittleEndian, uint32(0xD65F03C0)) // ret
	}

	ncmds := uint32(2) // LC_SEGMENT_64 + LC_SYMTAB
	segCmdSize := uint32(72 + 2*80)
	symCmdSize := uint32(24)
	sizeofcmds := segCmdSize + symCmdSize

	var buf bytes.Buffer
	// mach_header_64
	binary.Write(&buf, binary.LittleEndian, uint32(0xFEEDFACF))
	binary.Write(&buf, binary.LittleEndian, uint32(0x0100000C)) // CPU_TYPE_ARM64
	binary.Write(&buf, binary.LittleEndian, uint32(0))
	binary.Write(&buf, binary.LittleEndian, uint32(2)) // MH_EXECUTE
	binary.Write(&buf, binary.LittleEndian, ncmds)
	binary.Write(&buf, binary.LittleEndian, sizeofcmds)
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // flags
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // reserved

	// LC_SEGMENT_64
	binary.Write(&buf, binary.LittleEndian, uint32(0x19)) // LC_SEGMENT_64
	binary.Write(&buf, binary.LittleEndian, segCmdSize)
	segname := make([]byte, 16)
	copy(segname, "__TEXT")
	buf.Write(segname)
	binary.Write(&buf, binary.LittleEndian, uint64(vmaddr))
	binary.Write(&buf, binary.LittleEndian, uint64(0x4000)) // vmsize
	binary.Write(&buf, binary.LittleEndian, uint64(0))      // fileoff
	binary.Write(&buf, binary.LittleEndian, uint64(0x4000)) // filesize
	binary.Write(&buf, binary.LittleEndian, uint32(5))      // maxprot r-x
	binary.Write(&buf, binary.LittleEndian, uint32(5))
	binary.Write(&buf, binary.LittleEndian, uint32(2)) // nsects
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // flags

	writeSection := func(sect, seg string, addr, size, offset uint64, flags uint32) {
		sn := make([]byte, 16)
		copy(sn, sect)
		sg := make([]byte, 16)
		copy(sg, seg)
		buf.Write(sn)
		buf.Write(sg)
		binary.Write(&buf, binary.LittleEndian, addr)
		binary.Write(&buf, binary.LittleEndian, size)
		binary.Write(&buf, binary.LittleEndian, uint32(offset))
		binary.Write(&buf, binary.LittleEndian, uint32(2)) // align
		binary.Write(&buf, binary.LittleEndian, uint32(0)) // reloff
		binary.Write(&buf, binary.LittleEndian, uint32(0)) // nreloc
		binary.Write(&buf, binary.LittleEndian, flags)
		binary.Write(&buf, binary.LittleEndian, uint32(0))
		binary.Write(&buf, binary.LittleEndian, uint32(0))
		binary.Write(&buf, binary.LittleEndian, uint32(0))
	}
	writeSection("__text", "__TEXT", textAddr, uint64(text.Len()), textFileOff, 0x80000400)
	writeSection("__cstring", "__TEXT", cstrAddr, uint64(cstr.Len()), cstrFileOff, 0x2)

	// LC_SYMTAB
	symCount := uint32(1)
	strtab := []byte{0}
	strOff := uint32(1)
	name := "_codec_config"
	strtab = append(strtab, []byte(name)...)
	strtab = append(strtab, 0)
	binary.Write(&buf, binary.LittleEndian, uint32(0x2)) // LC_SYMTAB
	binary.Write(&buf, binary.LittleEndian, symCmdSize)
	if withSymbols {
		binary.Write(&buf, binary.LittleEndian, uint32(symFileOff))
		binary.Write(&buf, binary.LittleEndian, symCount)
		binary.Write(&buf, binary.LittleEndian, uint32(symFileOff+16))
		binary.Write(&buf, binary.LittleEndian, uint32(len(strtab)))
	} else {
		binary.Write(&buf, binary.LittleEndian, uint32(0))
		binary.Write(&buf, binary.LittleEndian, uint32(0))
		binary.Write(&buf, binary.LittleEndian, uint32(0))
		binary.Write(&buf, binary.LittleEndian, uint32(0))
	}

	// Pad and write section contents.
	padTo := func(off int) {
		for buf.Len() < off {
			buf.WriteByte(0)
		}
	}
	padTo(textFileOff)
	buf.Write(text.Bytes())
	padTo(cstrFileOff)
	buf.Write(cstr.Bytes())
	if withSymbols {
		padTo(symFileOff)
		// nlist_64: n_strx u32, n_type u8, n_sect u8, n_desc u16, n_value u64
		binary.Write(&buf, binary.LittleEndian, strOff)
		binary.Write(&buf, binary.LittleEndian, uint8(0xF)) // N_SECT|N_EXT|N_TYPE
		binary.Write(&buf, binary.LittleEndian, uint8(1))
		binary.Write(&buf, binary.LittleEndian, uint16(0))
		binary.Write(&buf, binary.LittleEndian, funcStart)
		buf.Write(strtab)
	}

	path := filepath.Join(t.TempDir(), "WeChat-test")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRelocateHookPointsMachO(t *testing.T) {
	path := buildSyntheticMachO(t, []string{"MMV1", "x'%s'"}, true)
	rep, err := RelocateHookPointsMachO(path, nil)
	if err != nil {
		t.Fatalf("RelocateHookPointsMachO: %v", err)
	}
	if rep.AnchorHits["MMV1"] != 1 || rep.AnchorHits["x'%s'"] != 1 {
		t.Fatalf("anchor hits = %v", rep.AnchorHits)
	}
	if len(rep.Candidates) != 2 {
		t.Fatalf("candidates = %d, want 2", len(rep.Candidates))
	}
	for _, c := range rep.Candidates {
		if c.RefAddr == 0 {
			t.Fatalf("candidate without ref: %+v", c)
		}
		if !rep.HasSymbols {
			t.Fatal("symbols expected")
		}
		if c.Symbol != "_codec_config" || c.FuncStart == 0 {
			t.Fatalf("function attribution wrong: %+v", c)
		}
	}
}

func TestRelocateHookPointsMachONoSymbols(t *testing.T) {
	path := buildSyntheticMachO(t, []string{"MMV1"}, false)
	rep, err := RelocateHookPointsMachO(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.HasSymbols {
		t.Fatal("unexpected symbols")
	}
	if len(rep.Candidates) != 1 || rep.Candidates[0].FuncStart != 0 || rep.Candidates[0].Symbol != "" {
		t.Fatalf("no-symbol candidates must carry ref addr only: %+v", rep.Candidates)
	}
}

func TestRelocateHookPointsMachOEmptyAnchorRejected(t *testing.T) {
	path := buildSyntheticMachO(t, []string{"MMV1"}, true)
	rep, err := RelocateHookPointsMachO(path, []RelocateAnchor{{Name: "empty"}})
	if err != nil {
		t.Fatal(err)
	}
	if rep.AnchorHits["empty"] != 0 || len(rep.Candidates) != 0 {
		t.Fatal("empty anchor must produce zero hits")
	}
}
