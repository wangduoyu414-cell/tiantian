package wcdb

import (
	"fmt"
	"os"
)

// Profile is one SQLCipher crypto configuration, defined centrally so the
// verifier, decryptor and WAL replay never diverge. Profiles are only added
// with sample evidence; unknown configurations must be reported, not guessed.
type Profile struct {
	Name      string `json:"name"`
	PageSize  int    `json:"page_size"`
	Reserve   int    `json:"reserve"` // IV + HMAC bytes at page tail
	IVSize    int    `json:"iv_size"`
	HMACSize  int    `json:"hmac_size"`
	SaltLen   int    `json:"salt_len"`
	MACKeyLen int    `json:"mac_key_len"`
	KDFIters  int    `json:"kdf_iters"`
	// Evidence records where this profile was actually verified.
	Evidence string `json:"evidence"`
}

// SQLCipher4WeChatProfile is the only profile with real-sample evidence:
// WeChat 4.1.x (verified on 4.1.12.x), SQLCipher 4 defaults.
func SQLCipher4WeChatProfile() Profile {
	return Profile{
		Name:      "sqlcipher4-wechat4.1",
		PageSize:  SQLCipherPageSize,
		Reserve:   SQLCipherReserve,
		IVSize:    SQLCipherIVSize,
		HMACSize:  SQLCipherHMACSize,
		SaltLen:   SaltLength,
		MACKeyLen: SQLCipherMACKeyLen,
		KDFIters:  DefaultKDFIters,
		Evidence:  "wechat 4.1.12.x samples",
	}
}

// ErrProfileUnknown means the file's shape matches no evidenced profile; this
// is distinct from "wrong key material" and "corrupt source".
var ErrProfileUnknown = profileError("unknown crypto profile")

type profileError string

func (e profileError) Error() string { return string(e) }

// DetectProfile sanity-checks a candidate DB file against known profiles
// WITHOUT touching key material. It returns the profile when the file shape
// is plausible, and ErrProfileUnknown otherwise, so callers can distinguish:
//
//	material error  = profile known, HMAC failed
//	profile unknown = this file does not even fit the expected layout
//	source corrupt  = profile known, pages fail HMAC with the right key
func DetectProfile(dbPath string) (Profile, error) {
	info, err := os.Stat(dbPath)
	if err != nil {
		return Profile{}, err
	}
	p := SQLCipher4WeChatProfile()
	if info.Size() == 0 || info.Size()%int64(p.PageSize) != 0 {
		return Profile{}, fmt.Errorf("%w: %s size %d is not a multiple of %d", ErrProfileUnknown, dbPath, info.Size(), p.PageSize)
	}
	f, err := os.Open(dbPath)
	if err != nil {
		return Profile{}, err
	}
	defer f.Close()
	// An encrypted SQLCipher file never starts with the plaintext SQLite magic.
	hdr := make([]byte, 16)
	if _, err := f.Read(hdr); err != nil {
		return Profile{}, err
	}
	if string(hdr) == "SQLite format 3\x00" {
		return Profile{}, fmt.Errorf("%w: %s is plaintext SQLite (no encryption profile applies)", ErrProfileUnknown, dbPath)
	}
	return p, nil
}
