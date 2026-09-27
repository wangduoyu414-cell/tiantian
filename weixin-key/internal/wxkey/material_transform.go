package wxkey

import "errors"

// NormalizeObservedPassphrase only changes a material representation. It does
// NOT authenticate it, associate it with an account, or authorize persistence.
// A nil mask selects direct passphrase bytes; a non-nil mask must be 32 bytes.
// The caller owns the returned buffer and must clear it after verification.
func NormalizeObservedPassphrase(observed, mask []byte) ([32]byte, error) {
	var out [32]byte
	if len(observed) != len(out) || (mask != nil && len(mask) != len(out)) {
		return out, errors.New("invalid observed material or transformation length")
	}
	copy(out[:], observed)
	if mask != nil {
		for i := range out {
			out[i] ^= mask[i]
		}
	}
	return out, nil
}
