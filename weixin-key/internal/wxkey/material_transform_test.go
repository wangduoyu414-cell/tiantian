package wxkey

import (
	"bytes"
	"testing"
)

func TestNormalizeObservedPassphrase(t *testing.T) {
	observed, mask := make([]byte, 32), make([]byte, 32)
	for i := range observed {
		observed[i], mask[i] = byte(i), byte(255-i)
	}
	before := append([]byte(nil), observed...)
	got, err := NormalizeObservedPassphrase(observed, mask)
	if err != nil || !bytes.Equal(got[:], bytes.Repeat([]byte{255}, 32)) {
		t.Fatal("XOR normalization failed")
	}
	if !bytes.Equal(observed, before) {
		t.Fatal("normalizer changed borrowed candidate")
	}
	direct, err := NormalizeObservedPassphrase(observed, nil)
	if err != nil || !bytes.Equal(direct[:], observed) {
		t.Fatal("identity normalization failed")
	}
	for _, sizes := range [][2]int{{31, 32}, {33, 32}, {32, 31}, {32, 0}} {
		// An empty non-nil mask is invalid; only nil selects identity.
		out, err := NormalizeObservedPassphrase(make([]byte, sizes[0]), make([]byte, sizes[1]))
		if err == nil || out != [32]byte{} {
			t.Fatal("invalid material shape accepted")
		}
	}
}

func syntheticMaskCode(start byte) []byte {
	var b []byte
	for i := range 4 {
		b = append(b, 0x48, 0xba)
		b = append(b, bytes.Repeat([]byte{byte(i + 1)}, 8)...)
		b = append(b, 0x48, 0x89, 0x55, start+byte(i*8))
	}
	return append(b, 0x48, 0x85, 0xc0)
}

func TestMaterialMaskInstructionShape(t *testing.T) {
	for _, displacement := range []byte{0, 0xb0} {
		code := syntheticMaskCode(displacement)
		value, ok := decodeProbeMaskCode(code)
		if !ok || value[0] != 1 || value[8] != 2 || value[31] != 4 {
			t.Fatal("exact contiguous-store shape rejected")
		}
		for _, at := range []int{0, 1, 10, 11, 12, 13, 27, 41, 55, 56, 57, 58} {
			bad := append([]byte(nil), code...)
			bad[at] ^= 0x10
			if _, ok := decodeProbeMaskCode(bad); ok {
				t.Fatalf("incorrect instruction/store accepted at offset %d", at)
			}
		}
		if _, ok := decodeProbeMaskCode(code[:58]); ok {
			t.Fatal("truncated sequence accepted")
		}
	}
}
