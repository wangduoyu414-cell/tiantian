//go:build windows

package wxkey

import (
	"bytes"
	"context"
	"crypto/sha512"
	"errors"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/sys/windows"
	"weixin-key/internal/wcdb"
)

type MaterialProbeOptions struct {
	PassiveProbeOptions
	ModulePath   string
	ModuleSHA256 string
	// PrimaryDB is relative to db_storage, explicitly selected by the caller.
	// A miss tests only this database; it says nothing about other key families.
	PrimaryDB string
}

type materialProbeSettings struct {
	profile   *probeMaterialProfile
	primaryDB string
	retained  *VerifiedPassiveKeys
}

// ProbePassiveMaterialObjects is an opt-in diagnostic, never a setup fallback.
// It tries one strict string layout with direct and module-bound XOR32
// passphrase interpretations. All candidate bytes are discarded, not returned
// or persisted. Like the raw probe, it never writes or controls WeChat.
func ProbePassiveMaterialObjects(ctx context.Context, opts MaterialProbeOptions) (PassiveProbeReport, error) {
	return probePassiveMaterialObjects(ctx, opts, nil)
}

func probePassiveMaterialObjects(ctx context.Context, opts MaterialProbeOptions, retained *VerifiedPassiveKeys) (PassiveProbeReport, error) {
	if !filepath.IsLocal(opts.PrimaryDB) || opts.PrimaryDB == "." {
		return PassiveProbeReport{Status: "failed"}, errors.New("material probe requires an explicit relative database")
	}
	p, err := loadProbeMaterialProfile(ctx, opts.ModulePath, opts.ModuleSHA256)
	if err != nil {
		return PassiveProbeReport{Status: "failed"}, err
	}
	defer p.clear()
	return probePassiveObjects(ctx, opts.PassiveProbeOptions, &materialProbeSettings{
		profile: p, primaryDB: opts.PrimaryDB, retained: retained,
	})
}

var errProbeModuleChanged = errors.New("material probe module identity or code changed")

// This observer only reads at most four short code sequences on the known module; it
// does not scan executable memory for account material or install a hook.
func checkProbeMaterialModule(ctx context.Context, h windows.Handle, pid uint32, p *probeMaterialProfile, report *PassiveProbeReport) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	snap, _, _ := procCreateToolhelp32.Call(th32csSnapModule, uintptr(pid))
	if snap == uintptr(syscall.InvalidHandle) || snap == 0 {
		return errors.New("cannot enumerate material probe target modules")
	}
	defer procCloseHandle.Call(snap)
	var entry windowsModuleEntry32W
	entry.Size = uint32(unsafe.Sizeof(entry))
	ok, _, enumErr := procModule32FirstW.Call(snap, uintptr(unsafe.Pointer(&entry)))
	found := 0
	mem := nativeObjectProbeMemory{h}
	for count := 0; ok != 0; count++ {
		if count >= 1024 {
			return ErrPassiveProbeBudget
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if strings.EqualFold(syscall.UTF16ToString(entry.ModuleName[:]), "Weixin.dll") {
			found++
			if found != 1 || !strings.EqualFold(filepath.Clean(syscall.UTF16ToString(entry.ExePath[:])), p.path) ||
				entry.ModBaseSize != p.imageSize {
				return errProbeModuleChanged
			}
			if err := checkProbeMaterialCode(ctx, mem, entry.ModBaseAddr, p, report); err != nil {
				return err
			}
		}
		entry.Size = uint32(unsafe.Sizeof(entry))
		ok, _, enumErr = procModule32NextW.Call(snap, uintptr(unsafe.Pointer(&entry)))
	}
	if !errors.Is(enumErr, windows.ERROR_NO_MORE_FILES) {
		return errors.New("material probe module enumeration did not complete")
	}
	if found != 1 {
		return errProbeModuleChanged
	}
	return ctx.Err()
}

func checkProbeMaterialCode(ctx context.Context, mem objectProbeMemory, base uintptr,
	p *probeMaterialProfile, report *PassiveProbeReport) error {
	for _, mask := range p.masks {
		if err := ctx.Err(); err != nil {
			return err
		}
		addr := base + uintptr(mask.rva)
		if addr < base || addr+probeMaskCodeSize < addr ||
			uint64(mask.rva)+probeMaskCodeSize > uint64(p.imageSize) {
			return errProbeModuleChanged
		}
		region, err := mem.query(addr)
		if err != nil {
			return errors.New("cannot query material profile code mapping")
		}
		if region.Type != 0x1000000 || region.State != windows.MEM_COMMIT ||
			region.AllocationBase != base || region.BaseAddress > addr ||
			region.BaseAddress+region.RegionSize <= region.BaseAddress ||
			addr+probeMaskCodeSize > region.BaseAddress+region.RegionSize ||
			(region.Protect != windows.PAGE_EXECUTE_READ && region.Protect != windows.PAGE_EXECUTE_READWRITE) {
			return errProbeModuleChanged
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		var code [probeMaskCodeSize]byte
		report.ModuleCodeBytes += len(code)
		err = mem.read(addr, code[:])
		same := bytes.Equal(code[:], mask.code[:])
		clear(code[:])
		if err != nil {
			return errors.New("cannot read material profile code")
		}
		if !same {
			return errProbeModuleChanged
		}
	}
	return ctx.Err()
}

func newMaterialProbeVerifier(ctx context.Context, verifiers []*sqlcipherVerifier, primary int, masks []probeMaterialMask,
	report *PassiveProbeReport, maxKDF int) func([]byte) (bool, error) {
	return newMaterialProbeVerifierRetaining(ctx, verifiers, primary, masks, report, maxKDF, nil)
}

func newMaterialProbeVerifierRetaining(ctx context.Context, verifiers []*sqlcipherVerifier, primary int, masks []probeMaterialMask,
	report *PassiveProbeReport, maxKDF int, retain func(int, []byte)) func([]byte) (bool, error) {
	matched := make([]bool, len(verifiers))
	check := func(key []byte, i int, kind int) bool {
		report.HMACChecks++
		if !verifiers[i].verify(key) {
			return false
		}
		if !matched[i] {
			matched[i] = true
			if retain != nil {
				retain(i, key)
			}
			report.VerifiedDBs++
			switch kind {
			case 0:
				report.RawDBs++
			case 1:
				report.DirectPassphraseDBs++
			case 2:
				report.NormalizedDBs++
			}
		}
		return true
	}
	derive := func(pass []byte, i, kind int) (bool, error) {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if report.KDFChecks >= maxKDF {
			return false, ErrPassiveProbeBudget
		}
		report.KDFChecks++
		key := pbkdf2.Key(pass, verifiers[i].dbSalt, wcdb.DefaultKDFIters, 32, sha512.New)
		defer clear(key)
		if err := ctx.Err(); err != nil {
			return false, err
		}
		return check(key, i, kind), nil
	}
	return func(candidate []byte) (bool, error) {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if len(candidate) != 32 || primary < 0 || primary >= len(verifiers) || len(masks) > 4 {
			return false, errors.New("invalid material verifier input")
		}
		if check(candidate, primary, 0) {
			for i := range verifiers {
				if err := ctx.Err(); err != nil {
					return false, err
				}
				if !matched[i] {
					check(candidate, i, 0)
				}
			}
		}
		if report.VerifiedDBs == len(verifiers) {
			return true, nil
		}
		for mode := 0; mode <= len(masks); mode++ {
			var mask []byte
			kind := 1
			if mode > 0 {
				mask = masks[mode-1].value[:]
				kind = 2
			}
			pass, err := NormalizeObservedPassphrase(candidate, mask)
			if err != nil {
				return false, err
			}
			ok, err := derive(pass[:], primary, kind)
			if err == nil && ok {
				for i := range verifiers {
					if matched[i] {
						continue
					}
					if _, err = derive(pass[:], i, kind); err != nil {
						break
					}
				}
			}
			clear(pass[:])
			if err != nil {
				return false, err
			}
			if report.VerifiedDBs == len(verifiers) {
				return true, nil
			}
		}
		return false, nil
	}
}
