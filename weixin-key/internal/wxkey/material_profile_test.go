package wxkey

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/pe"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// A minimal, synthetic AMD64 PE. No installed module or captured bytes.
func syntheticMaterialPE(count int) []byte {
	b := make([]byte, 0x800)
	copy(b, "MZ")
	binary.LittleEndian.PutUint32(b[0x3c:], 0x80)
	copy(b[0x80:], "PE\x00\x00")
	write := func(off int, v any) {
		var buf bytes.Buffer
		if err := binary.Write(&buf, binary.LittleEndian, v); err != nil {
			panic(err)
		}
		copy(b[off:], buf.Bytes())
	}
	write(0x84, pe.FileHeader{Machine: pe.IMAGE_FILE_MACHINE_AMD64, NumberOfSections: 2, SizeOfOptionalHeader: 240})
	write(0x98, pe.OptionalHeader64{Magic: 0x20b, SizeOfImage: 0x4000, SizeOfHeaders: 0x200,
		SectionAlignment: 0x1000, FileAlignment: 0x200, NumberOfRvaAndSizes: 16})
	write(0x188, pe.SectionHeader32{Name: [8]byte{'.', 't', 'e', 'x', 't'}, VirtualSize: 0x400,
		VirtualAddress: 0x1000, SizeOfRawData: 0x400, PointerToRawData: 0x200, Characteristics: pe.IMAGE_SCN_MEM_EXECUTE})
	write(0x1b0, pe.SectionHeader32{Name: [8]byte{'.', 'p', 'd', 'a', 't', 'a'}, VirtualSize: 0x200,
		VirtualAddress: 0x2000, SizeOfRawData: 0x200, PointerToRawData: 0x600})
	for i := range count {
		copy(b[0x200+i*64:], syntheticMaskCode(0))
		binary.LittleEndian.PutUint32(b[0x600+i*12:], uint32(0x1000+i*64))
		binary.LittleEndian.PutUint32(b[0x604+i*12:], uint32(0x1000+i*64+59))
	}
	return b
}

func TestMaterialProfilePinnedSyntheticPE(t *testing.T) {
	b := syntheticMaterialPE(2)
	path := filepath.Join(t.TempDir(), "synthetic.dll")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(b)
	p, err := loadProbeMaterialProfile(context.Background(), path, hex.EncodeToString(hash[:]))
	if err != nil {
		t.Fatal(err)
	}
	if p.report.CandidateCount != 2 || p.report.SequenceRVAs[1] != 0x1040 || p.report.Status != "static-hypothesis-only" {
		t.Fatal("incorrect static evidence")
	}
	out, err := json.Marshal(p.report)
	if err != nil || bytes.Contains(out, []byte("value")) || bytes.Contains(out, []byte("code")) {
		t.Fatal("static report contains material")
	}
	p.clear()
	for _, m := range p.masks {
		if m.value != [32]byte{} || m.code != [59]byte{} {
			t.Fatal("profile cleanup left material")
		}
	}
}

func TestMaterialProfileRejectsBadPEAndHash(t *testing.T) {
	for _, kind := range []string{"hash", "truncated", "not-pe", "machine", "section-file-range",
		"section-image-range", "no-function", "short-function", "no-shape", "too-many"} {
		t.Run(kind, func(t *testing.T) {
			b := syntheticMaterialPE(1)
			switch kind {
			case "truncated":
				b = b[:600]
			case "not-pe":
				clear(b[:256])
			case "machine":
				binary.LittleEndian.PutUint16(b[0x84:], pe.IMAGE_FILE_MACHINE_I386)
			case "section-file-range":
				binary.LittleEndian.PutUint32(b[0x188+20:], 0xfffffff0)
			case "section-image-range":
				binary.LittleEndian.PutUint32(b[0x188+12:], 0xfffffff0)
			case "no-function":
				clear(b[0x600:])
			case "short-function":
				binary.LittleEndian.PutUint32(b[0x604:], 0x103a)
			case "no-shape":
				b[0x201] ^= 1
			case "too-many":
				b = syntheticMaterialPE(5)
			}
			hash := sha256.Sum256(b)
			if kind == "hash" {
				hash[0] ^= 1
			}
			path := filepath.Join(t.TempDir(), "synthetic.dll")
			if err := os.WriteFile(path, b, 0o600); err != nil {
				t.Fatal(err)
			}
			if p, err := loadProbeMaterialProfile(context.Background(), path, hex.EncodeToString(hash[:])); err == nil || p != nil {
				t.Fatal("invalid profile accepted")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := InspectPassiveMaterialProfile(ctx, "must-not-read", ""); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled static inspection performed IO")
	}
}
