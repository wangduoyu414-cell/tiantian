package export

import (
	"bytes"
	"crypto/aes"
	"encoding/binary"
	"fmt"
)

// WeChat v4 .dat container (4.1 verified, format per chatlog/dat2img):
//
//	[0:4]   magic 07 08 56 32 (v2) / 07 08 56 31 (v1, different key)
//	[4:6]   08 07
//	[6:10]  AES-128-ECB region length, LE uint32 (1024 observed)
//	[10:14] XOR region length at file tail, LE uint32
//	[14]    0x01
//	[15:]   payload = AES-ECB(region, rounded up to 16) | middle | XOR tail
//
// The image AES key is account-global; the XOR key varies per file and is
// recovered from the known image tail (FF D9 for JPEG).
var (
	datMagicV2 = []byte{0x07, 0x08, 0x56, 0x32}
	datMagicV1 = []byte{0x07, 0x08, 0x56, 0x31}
	// v1Key is the documented v1 key (chatlog/dat2img); v2 keys are
	// per-installation and must be supplied.
	datV1Key = []byte("cfcd208495d565ef")
)

var datImageSigs = []struct {
	magic []byte
	ext   string
}{
	{[]byte{0xFF, 0xD8, 0xFF}, "jpg"},
	{[]byte{0x89, 0x50, 0x4E, 0x47}, "png"},
	{[]byte{0x47, 0x49, 0x46, 0x38}, "gif"},
	{[]byte{0x49, 0x49, 0x2A, 0x00}, "tiff"},
	{[]byte{0x42, 0x4D}, "bmp"},
	{[]byte("wxgf"), "wxgf"}, // WeChat proprietary; kept raw for later ffmpeg decode
}

// isDatContainer reports whether b looks like a WeChat v4 .dat file.
func isDatContainer(b []byte) bool {
	return len(b) > 15 && (bytes.HasPrefix(b, datMagicV2) || bytes.HasPrefix(b, datMagicV1))
}

// decodeDat decrypts a WeChat v4/v1 .dat image. aesKey is used for v2;
// v1 uses its documented fixed key. fallbackXorKey (when non-nil) is the
// account-global tail key learned from an earlier JPEG decode, used when the
// payload is not a JPEG (wxgf) and per-file derivation has no anchor.
// Returns plaintext image bytes, extension, and the tail XOR key used (valid
// when the file has an XOR region), so callers can cache it as the
// account-global fallback for non-JPEG payloads.
func decodeDat(data, aesKey []byte, fallbackXorKey *byte) ([]byte, string, byte, error) {
	if len(data) < 15 {
		return nil, "", 0, fmt.Errorf("dat too short: %d", len(data))
	}
	key := aesKey
	switch {
	case bytes.HasPrefix(data, datMagicV1):
		key = datV1Key
	case bytes.HasPrefix(data, datMagicV2):
		if len(key) != 16 {
			return nil, "", 0, fmt.Errorf("v2 dat needs a 16-byte image key")
		}
	default:
		return nil, "", 0, fmt.Errorf("not a v4 dat container")
	}

	aesLen := binary.LittleEndian.Uint32(data[6:10])
	xorLen := binary.LittleEndian.Uint32(data[10:14])
	fileData := data[15:]
	if aesLen > uint32(len(fileData)) || xorLen > uint32(len(fileData)) || uint64(aesLen)+uint64(xorLen) > uint64(len(fileData))+16 {
		return nil, "", 0, fmt.Errorf("dat header lengths inconsistent: aes=%d xor=%d data=%d", aesLen, xorLen, len(fileData))
	}

	// AES-ECB region, rounded up to a full block (the extra block exists even
	// when aesLen is already aligned).
	aesLen0 := aesLen/16*16 + 16
	if aesLen0 > uint32(len(fileData)) {
		aesLen0 = uint32(len(fileData))
	}
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, "", 0, err
	}
	result := make([]byte, 0, len(fileData))
	if aesLen0 > 0 {
		dec := make([]byte, aesLen0)
		for i := 0; i+16 <= int(aesLen0); i += 16 {
			c.Decrypt(dec[i:i+16], fileData[i:i+16])
		}
		n := int(aesLen)
		if n > len(dec) {
			n = len(dec)
		}
		result = append(result, dec[:n]...)
	}

	midEnd := uint32(len(fileData)) - xorLen
	if aesLen0 < midEnd {
		result = append(result, fileData[aesLen0:midEnd]...)
	}
	var usedXor byte
	if xorLen > 0 && midEnd < uint32(len(fileData)) {
		tail := fileData[midEnd:]
		xk, ok := datXorKey(result, tail)
		if !ok {
			if fallbackXorKey == nil {
				return nil, "", 0, fmt.Errorf("cannot derive tail xor key")
			}
			xk = *fallbackXorKey
		}
		usedXor = xk
		for _, b := range tail {
			result = append(result, b^xk)
		}
	}

	for _, sig := range datImageSigs {
		if bytes.HasPrefix(result, sig.magic) {
			return result, sig.ext, usedXor, nil
		}
	}
	return nil, "", 0, fmt.Errorf("unknown image type after dat decode: %x", result[:min(4, len(result))])
}

// datXorKey derives the tail XOR key from known image tails. JPEG ends FF D9;
// both tail bytes must agree on the key.
func datXorKey(head []byte, tail []byte) (byte, bool) {
	if len(tail) < 2 {
		return 0, false
	}
	type tailSig struct{ b1, b2 byte }
	var candidates []tailSig
	switch {
	case bytes.HasPrefix(head, []byte{0xFF, 0xD8, 0xFF}):
		candidates = []tailSig{{0xFF, 0xD9}}
	case bytes.HasPrefix(head, []byte{0x89, 0x50}):
		candidates = []tailSig{{0x42, 0x60}, {0x60, 0x82}} // PNG IEND tail variants
	case bytes.HasPrefix(head, []byte{0x47, 0x49}):
		candidates = []tailSig{{0x00, 0x3B}} // GIF trailer
	default:
		candidates = []tailSig{{0xFF, 0xD9}}
	}
	for _, c := range candidates {
		k1 := tail[len(tail)-2] ^ c.b1
		k2 := tail[len(tail)-1] ^ c.b2
		if k1 == k2 {
			return k1, true
		}
	}
	return 0, false
}
