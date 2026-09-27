package wxkey

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const probeMaskCodeSize = 59

// This is an exploratory static shape, not a trusted capture ABI. Only
// database authentication can establish that its constants transform material.
type probeMaterialMask struct {
	rva   uint32
	code  [probeMaskCodeSize]byte
	value [32]byte
}

type probeMaterialProfile struct {
	path      string
	imageSize uint32
	report    MaterialProfileReport
	masks     []probeMaterialMask
}

// MaterialProfileReport exposes module evidence, never transformation bytes.
type MaterialProfileReport struct {
	ModuleSHA256   string   `json:"module_sha256"`
	SequenceRVAs   []uint32 `json:"sequence_rvas"`
	CandidateCount int      `json:"candidate_count"`
	Status         string   `json:"status"`
}

// decodeProbeMaskCode accepts four exact MOV imm64/store pairs with consecutive
// stack slots, followed by TEST RAX,RAX. It does not guess instruction lengths
// or walk backwards to a function entry. The containing PE function and the
// full module hash are checked separately.
func decodeProbeMaskCode(b []byte) ([32]byte, bool) {
	var out [32]byte
	if len(b) != probeMaskCodeSize || !bytes.Equal(b[56:], []byte{0x48, 0x85, 0xc0}) {
		return out, false
	}
	start := int(int8(b[13]))
	if start > 103 { // no signed disp8 wrap between adjacent stores
		return out, false
	}
	for i := range 4 {
		at := i * 14
		if b[at] != 0x48 || b[at+1] != 0xba ||
			!bytes.Equal(b[at+10:at+13], []byte{0x48, 0x89, 0x55}) ||
			int(int8(b[at+13])) != start+i*8 {
			return [32]byte{}, false
		}
		copy(out[i*8:i*8+8], b[at+2:at+10])
	}
	return out, true
}

// InspectPassiveMaterialProfile reads a pinned disk image only. A successful
// return means the static hypothesis has candidates, NOT that keys were found.
func InspectPassiveMaterialProfile(ctx context.Context, path, expectedSHA256 string) (MaterialProfileReport, error) {
	p, err := loadProbeMaterialProfile(ctx, path, expectedSHA256)
	if err != nil {
		return MaterialProfileReport{}, err
	}
	defer p.clear()
	return p.report, nil
}

func (p *probeMaterialProfile) clear() {
	for i := range p.masks {
		clear(p.masks[i].value[:])
		clear(p.masks[i].code[:])
	}
}

func loadProbeMaterialProfile(ctx context.Context, path, expected string) (*probeMaterialProfile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	want, err := hex.DecodeString(expected)
	if err != nil || len(want) != sha256.Size || !filepath.IsAbs(path) {
		return nil, errors.New("material profile requires an absolute module path and explicit SHA256")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("cannot open material profile module")
	}
	defer f.Close()
	info, err := f.Stat()
	const maxModuleSize = 256 << 20
	if err != nil || !info.Mode().IsRegular() || info.Size() < 64 || info.Size() > maxModuleSize {
		return nil, errors.New("invalid or oversized material profile module")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxModuleSize+1))
	if err != nil || int64(len(data)) != info.Size() {
		return nil, errors.New("incomplete material profile module")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	hash := sha256.Sum256(data)
	if !bytes.Equal(hash[:], want) {
		return nil, errors.New("material profile module SHA256 mismatch")
	}
	image, err := pe.NewFile(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("invalid material profile PE")
	}
	defer image.Close()
	oh, ok := image.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || image.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
		return nil, errors.New("material hypothesis requires an AMD64 PE")
	}
	var sections []peSection
	for _, s := range image.Sections {
		if uint64(s.Offset)+uint64(s.Size) > uint64(len(data)) ||
			uint64(s.VirtualAddress)+uint64(s.Size) > uint64(oh.SizeOfImage) {
			return nil, errors.New("material profile section outside image")
		}
		sections = append(sections, peSection{name: s.Name, va: s.VirtualAddress, vsize: s.VirtualSize, rawOff: s.Offset, rawSize: s.Size})
	}
	functions := parsePdata(data, sections)
	p := &probeMaterialProfile{
		path: filepath.Clean(path), imageSize: oh.SizeOfImage,
		report: MaterialProfileReport{ModuleSHA256: strings.ToLower(expected), Status: "static-hypothesis-only"},
	}
	fail := func(err error) (*probeMaterialProfile, error) { p.clear(); return nil, err }
	for _, s := range image.Sections {
		if s.Characteristics&pe.IMAGE_SCN_MEM_EXECUTE == 0 {
			continue
		}
		body := data[int(s.Offset) : int(s.Offset)+int(s.Size)]
		for off := 0; off+probeMaskCodeSize <= len(body); {
			if err := ctx.Err(); err != nil {
				return fail(err)
			}
			at := bytes.Index(body[off:], []byte{0x48, 0xba})
			if at < 0 {
				break
			}
			off += at
			if off+probeMaskCodeSize > len(body) {
				break
			}
			code := body[off : off+probeMaskCodeSize]
			value, ok := decodeProbeMaskCode(code)
			rva := s.VirtualAddress + uint32(off)
			off++
			if !ok {
				continue
			}
			entry := findFuncEntryPdata(functions, rva)
			enclosed := false
			for _, fn := range functions {
				if fn.BeginRVA == entry && entry != 0 && uint64(rva)+probeMaskCodeSize <= uint64(fn.EndRVA) {
					enclosed = true
					break
				}
			}
			if !enclosed {
				clear(value[:])
				continue
			}
			if len(p.masks) >= 4 {
				clear(value[:])
				return fail(errors.New("too many static material hypotheses"))
			}
			m := probeMaterialMask{rva: rva, value: value}
			copy(m.code[:], code)
			p.masks = append(p.masks, m)
			p.report.SequenceRVAs = append(p.report.SequenceRVAs, rva)
			clear(value[:])
		}
	}
	p.report.CandidateCount = len(p.masks)
	if len(p.masks) == 0 {
		return fail(errors.New("material hypothesis is not applicable to this image"))
	}
	return p, nil
}
