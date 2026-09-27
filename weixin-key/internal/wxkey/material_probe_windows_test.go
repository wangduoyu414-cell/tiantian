//go:build windows

package wxkey

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"testing"

	"golang.org/x/sys/windows"
	"weixin-key/internal/testutil"
)

func setStringProbeObject(m *fakeObjectProbeMemory, offset, keyOffset int, key []byte) {
	binary.LittleEndian.PutUint64(m.data[offset:], uint64(fakeProbeBase)+uint64(keyOffset))
	binary.LittleEndian.PutUint64(m.data[offset+8:], 0)
	binary.LittleEndian.PutUint64(m.data[offset+16:], 32)
	binary.LittleEndian.PutUint64(m.data[offset+24:], 47)
	copy(m.data[keyOffset:], key)
}

func TestBoundedStringObjectLayoutAndOverlap(t *testing.T) {
	m := &fakeObjectProbeMemory{data: make([]byte, probeChunk+256)}
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	setStringProbeObject(m, probeChunk-8, probeChunk+80, key)
	var r PassiveProbeReport
	calls := 0
	visit := func(b []byte) (bool, error) { calls++; return true, nil }
	if err := scanBoundedKeyObjects(context.Background(), m, defaultObjectProbeLimits, &r, visit); err != nil || calls != 0 {
		t.Fatal("legacy layout should not see the different string shape")
	}
	r = PassiveProbeReport{}
	if err := scanBoundedStringObjects(context.Background(), m, defaultObjectProbeLimits, &r, visit); err != nil || calls != 1 {
		t.Fatal("string crossing chunk boundary was missed")
	}
	for _, off := range []int{8, 16, 24} {
		setStringProbeObject(m, probeChunk-8, probeChunk+80, key)
		m.data[probeChunk-8+off] ^= 1
		r, calls = PassiveProbeReport{}, 0
		if err := scanBoundedStringObjects(context.Background(), m, defaultObjectProbeLimits, &r, visit); err != nil || calls != 0 {
			t.Fatal("incorrect string layout accepted")
		}
	}
}

func TestMaterialProbeNormalizedPassphraseCoversDifferentSalts(t *testing.T) {
	first, pass, salt1, _ := testutil.MustNewPassphraseDB(t.TempDir(), "first.db", 1)
	second, salt2, _ := testutil.MustNewPassphraseDBWith(t.TempDir(), "second.db", 1, pass)
	pp, _ := hex.DecodeString(pass)
	var mask probeMaterialMask
	for i := range mask.value {
		mask.value[i] = byte(i + 11)
	}
	observed, _ := NormalizeObservedPassphrase(pp, mask.value[:])
	vs := []*sqlcipherVerifier{newSQLCipherVerifier(first, salt1), newSQLCipherVerifier(second, salt2)}
	var r PassiveProbeReport
	verify := newMaterialProbeVerifier(context.Background(), vs, 0, []probeMaterialMask{mask}, &r, 16)
	done, err := verify(observed[:])
	if err != nil || !done || r.VerifiedDBs != 2 || r.NormalizedDBs != 2 || r.DirectPassphraseDBs != 0 {
		t.Fatalf("normalized verification failed: counts=%d/%d err=%v", r.VerifiedDBs, r.NormalizedDBs, err)
	}
	if r.KDFChecks != 3 { // direct miss; transformed primary and second salt
		t.Fatalf("unexpected derivation count: %d", r.KDFChecks)
	}
}

func TestMaterialProbeKDFBudgetAndCancellation(t *testing.T) {
	path, pass, salt, _ := testutil.MustNewPassphraseDB(t.TempDir(), "target.db", 1)
	pp, _ := hex.DecodeString(pass)
	vs := []*sqlcipherVerifier{newSQLCipherVerifier(path, salt)}
	var r PassiveProbeReport
	verify := newMaterialProbeVerifier(context.Background(), vs, 0, nil, &r, 0)
	if _, err := verify(pp); !errors.Is(err, ErrPassiveProbeBudget) || r.KDFChecks != 0 {
		t.Fatal("KDF budget was not checked before deriving")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r = PassiveProbeReport{}
	verify = newMaterialProbeVerifier(ctx, vs, 0, nil, &r, 10)
	if _, err := verify(pp); !errors.Is(err, context.Canceled) || r.HMACChecks != 0 || r.KDFChecks != 0 {
		t.Fatal("cancelled verification performed work")
	}
}

func TestMaterialProbeMissStaysOnSelectedDatabase(t *testing.T) {
	first, _, salt1, _ := testutil.MustNewPassphraseDB(t.TempDir(), "first.db", 1)
	second, pass2, salt2, _ := testutil.MustNewPassphraseDB(t.TempDir(), "second.db", 1)
	candidate, _ := hex.DecodeString(pass2)
	var r PassiveProbeReport
	verify := newMaterialProbeVerifier(context.Background(),
		[]*sqlcipherVerifier{newSQLCipherVerifier(first, salt1), newSQLCipherVerifier(second, salt2)}, 0, nil, &r, 4)
	if done, err := verify(candidate); done || err != nil || r.VerifiedDBs != 0 || r.KDFChecks != 1 || r.HMACChecks != 2 {
		t.Fatal("primary miss expanded the unselected database")
	}
}

func TestBoundedStringObjectBudgetsAndCancel(t *testing.T) {
	for _, kind := range []string{"read", "query", "object", "candidate", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			m := &fakeObjectProbeMemory{data: make([]byte, 4096)}
			key := make([]byte, 32)
			for i := range key {
				key[i] = byte(i + 1)
			}
			setStringProbeObject(m, 0, 512, key)
			key[0]++
			setStringProbeObject(m, 32, 544, key)
			limits := defaultObjectProbeLimits
			switch kind {
			case "read":
				limits.readBytes = 100
			case "query":
				limits.queries = 1
			case "object":
				limits.objects = 1
			case "candidate":
				limits.candidates = 1
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var r PassiveProbeReport
			calls := 0
			var borrowed []byte
			err := scanBoundedStringObjects(ctx, m, limits, &r, func(b []byte) (bool, error) {
				borrowed = b
				calls++
				if kind == "cancel" {
					cancel()
				}
				return false, nil
			})
			if kind == "cancel" {
				if !errors.Is(err, context.Canceled) || calls != 1 {
					t.Fatal("cancelled string scan continued verification")
				}
			} else if !errors.Is(err, ErrPassiveProbeBudget) {
				t.Fatal("missing explicit budget failure")
			}
			for _, b := range borrowed {
				if b != 0 {
					t.Fatal("borrowed candidate not cleared")
				}
			}
			if r.ReadBytes > limits.readBytes || r.Queries > limits.queries ||
				r.Objects > limits.objects || r.Candidates > limits.candidates {
				t.Fatal("string scan exceeded a hard counter")
			}
		})
	}
}

type fakeMaterialCodeMemory struct {
	region windows.MemoryBasicInformation
	code   []byte
	reads  int
	fail   bool
}

func (m *fakeMaterialCodeMemory) query(uintptr) (windows.MemoryBasicInformation, error) {
	return m.region, nil
}
func (m *fakeMaterialCodeMemory) read(_ uintptr, b []byte) error {
	m.reads++
	if m.fail {
		return windows.ERROR_PARTIAL_COPY
	}
	copy(b, m.code)
	return nil
}
func TestMaterialProbeCodeReadBoundary(t *testing.T) {
	for _, kind := range []string{"valid", "wrong-code", "private", "guard", "different-base", "short",
		"overflow", "outside-image", "read-failed", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			m := &fakeMaterialCodeMemory{code: syntheticMaskCode(0),
				region: windows.MemoryBasicInformation{BaseAddress: 0x11000, AllocationBase: 0x10000,
					RegionSize: 4096, Type: 0x1000000, State: windows.MEM_COMMIT, Protect: windows.PAGE_EXECUTE_READ}}
			mask := probeMaterialMask{rva: 0x1000}
			copy(mask.code[:], m.code)
			p := &probeMaterialProfile{imageSize: 0x3000, masks: []probeMaterialMask{mask}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "wrong-code":
				m.code[4] ^= 1
			case "private":
				m.region.Type = probePrivate
			case "guard":
				m.region.Protect |= windows.PAGE_GUARD
			case "different-base":
				m.region.AllocationBase++
			case "short":
				m.region.RegionSize = 58
			case "overflow":
				m.region.RegionSize = ^uintptr(0)
			case "outside-image":
				p.imageSize = 0x103a
			case "read-failed":
				m.fail = true
			case "cancel":
				cancel()
			}
			var r PassiveProbeReport
			err := checkProbeMaterialCode(ctx, m, 0x10000, p, &r)
			switch kind {
			case "valid":
				if err != nil || m.reads != 1 || r.ModuleCodeBytes != 59 {
					t.Fatal("exact image-bound code rejected")
				}
			case "cancel":
				if !errors.Is(err, context.Canceled) || m.reads != 0 {
					t.Fatal("cancelled code reader performed work")
				}
			case "read-failed":
				if err == nil || errors.Is(err, errProbeModuleChanged) {
					t.Fatal("unavailable code must not claim observed change")
				}
			default:
				if !errors.Is(err, errProbeModuleChanged) {
					t.Fatal("unsafe mapping/code accepted")
				}
				if kind != "wrong-code" && m.reads != 0 {
					t.Fatal("unsafe mapping read")
				}
			}
		})
	}
}

func TestMaterialProbeIncompleteValidationCannotClaimSuccess(t *testing.T) {
	for _, kind := range []string{"cancel", "unavailable", "changed", "stable"} {
		t.Run(kind, func(t *testing.T) {
			r := PassiveProbeReport{IdentityStable: true, AccountStable: true, SourceStable: true, VerifiedDBs: 2, NormalizedDBs: 2}
			var err error
			switch kind {
			case "cancel":
				err = context.DeadlineExceeded
			case "unavailable":
				err = errors.New("cannot enumerate")
			case "changed":
				err = errProbeModuleChanged
			case "stable":
				r.ModuleStable = true
			}
			finalizeProbeStability(&r, true, nil, nil, err, nil)
			if kind == "stable" {
				if r.VerifiedDBs != 2 {
					t.Fatal("stable evidence discarded")
				}
			} else if r.VerifiedDBs != 0 || r.NormalizedDBs != 0 {
				t.Fatal("unvalidated result retained success counts")
			}
			if (r.Status == "identity-or-source-invalidated") != (kind == "changed") {
				t.Fatal("unavailable post-validation falsely reported observed identity change")
			}
		})
	}
}

func TestMaterialProbeAccountOwnerRecheck(t *testing.T) {
	expected := accountProcess{pid: 123, start: windows.Filetime{LowDateTime: 777}}
	dbs := []windowsSourceDB{{rel: `message\message_0.db`}}
	for _, kind := range []string{"stable", "lost", "different-pid", "reused-pid", "query-failed", "cancel-before", "cancel-during"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if kind == "cancel-before" {
				cancel()
			}
			calls := 0
			observe := func(got []windowsSourceDB) ([]accountProcess, error) {
				calls++
				if len(got) != 1 || got[0].rel != dbs[0].rel {
					t.Fatal("rechecked wrong account")
				}
				owner := expected
				switch kind {
				case "lost":
					return nil, nil
				case "different-pid":
					owner.pid++
				case "reused-pid":
					owner.start.LowDateTime++
				case "query-failed":
					return nil, windows.ERROR_ACCESS_DENIED
				case "cancel-during":
					cancel()
				}
				return []accountProcess{owner}, nil
			}
			stable, err := recheckProbeAccount(ctx, dbs, expected, observe)
			r := PassiveProbeReport{IdentityStable: true, SourceStable: true, ModuleStable: true,
				AccountStable: stable, VerifiedDBs: 2, NormalizedDBs: 2}
			finalizeProbeStability(&r, true, nil, nil, nil, err)
			switch kind {
			case "stable":
				if err != nil || !stable || r.VerifiedDBs != 2 {
					t.Fatal("stable account rejected")
				}
			case "lost", "different-pid", "reused-pid":
				if !errors.Is(err, errProbeAccountChanged) || r.Status != "identity-or-source-invalidated" {
					t.Fatal("changed account owner accepted")
				}
			case "query-failed":
				if !errors.Is(err, windows.ERROR_ACCESS_DENIED) || r.Status != "validation-incomplete" {
					t.Fatal("unavailable account incorrectly reported as changed")
				}
			case "cancel-before", "cancel-during":
				if !errors.Is(err, context.Canceled) || r.Status != "validation-incomplete" {
					t.Fatal("cancelled account recheck ignored")
				}
				if kind == "cancel-before" && calls != 0 {
					t.Fatal("cancelled recheck performed IO")
				}
			}
			if kind != "stable" && (stable || r.VerifiedDBs != 0 || r.NormalizedDBs != 0) {
				t.Fatal("lost/unverified account retained success evidence")
			}
		})
	}
}
