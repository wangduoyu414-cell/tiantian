//go:build windows

package wxkey

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ErrPassiveProbeBudget means the observation was incomplete, not "no key".
var ErrPassiveProbeBudget = errors.New("passive probe observation budget exhausted")

// PassiveProbeOptions names one already observed process and one explicit
// account. The probe never uses config/env defaults or discovers other targets.
type PassiveProbeOptions struct {
	DBRoot     string
	PID        uint32
	CreatedUTC time.Time
	ExePath    string
}

// PassiveProbeReport deliberately contains no material, salts or addresses.
// VerifiedDBs is page-one HMAC evidence, not a decrypted database or export.
type PassiveProbeReport struct {
	Status              string                 `json:"status"`
	Route               string                 `json:"route"`
	SourceDBs           int                    `json:"source_dbs"`
	VerifiedDBs         int                    `json:"verified_dbs"`
	Regions             int                    `json:"regions"`
	Queries             int                    `json:"queries"`
	ReadBytes           uint64                 `json:"read_bytes"`
	ReadFailures        int                    `json:"read_failures"`
	Objects             int                    `json:"objects"`
	Candidates          int                    `json:"candidates"`
	HMACChecks          int                    `json:"hmac_checks"`
	ElapsedMS           int64                  `json:"elapsed_ms"`
	IdentityStable      bool                   `json:"identity_stable"`
	AccountStable       bool                   `json:"account_owner_stable"`
	SourceStable        bool                   `json:"source_page1_stable"`
	ProbeDBs            int                    `json:"probe_dbs,omitempty"`
	KDFChecks           int                    `json:"kdf_checks,omitempty"`
	RawDBs              int                    `json:"raw_dbs,omitempty"`
	DirectPassphraseDBs int                    `json:"direct_passphrase_dbs,omitempty"`
	NormalizedDBs       int                    `json:"normalized_dbs,omitempty"`
	ModuleStable        bool                   `json:"module_stable,omitempty"`
	ModuleCodeBytes     int                    `json:"module_code_read_bytes,omitempty"`
	MaterialProfile     *MaterialProfileReport `json:"material_profile,omitempty"`
	Scope               string                 `json:"scope,omitempty"`
}

type objectProbeLimits struct {
	readBytes  uint64
	queries    int
	objects    int
	candidates int
}

var defaultObjectProbeLimits = objectProbeLimits{512 << 20, 65536, 65536, 4096}

const (
	probeMaxAddress = uintptr(0x00007fffffffffff)
	probePrivate    = 0x20000 // WinNT.h MEM_PRIVATE
	probeChunk      = 256 << 10
)

// objectProbeMemory is a read-only boundary, also used by synthetic tests.
// Query returns ERROR_INVALID_PARAMETER when enumeration has reached the end.
type objectProbeMemory interface {
	query(uintptr) (windows.MemoryBasicInformation, error)
	read(uintptr, []byte) error
}

type nativeObjectProbeMemory struct{ h windows.Handle }

func (m nativeObjectProbeMemory) query(addr uintptr) (windows.MemoryBasicInformation, error) {
	var region windows.MemoryBasicInformation
	err := windows.VirtualQueryEx(m.h, addr, &region, unsafe.Sizeof(region))
	return region, err
}

func (m nativeObjectProbeMemory) read(addr uintptr, b []byte) error {
	var n uintptr
	err := windows.ReadProcessMemory(m.h, addr, &b[0], uintptr(len(b)), &n)
	if err != nil {
		return err
	}
	if n != uintptr(len(b)) {
		return errors.New("partial passive memory read")
	}
	return nil
}

func probeReadable(m windows.MemoryBasicInformation) bool {
	// Deliberately narrower than legacy D0: never scan code/image mappings.
	return m.State == windows.MEM_COMMIT && m.Type == probePrivate &&
		m.Protect == windows.PAGE_READWRITE && m.RegionSize > 0 &&
		m.BaseAddress+m.RegionSize > m.BaseAddress
}

func probeContains(m windows.MemoryBasicInformation, addr, n uintptr) bool {
	end := addr + n
	return probeReadable(m) && n > 0 && end > addr &&
		addr >= m.BaseAddress && end <= m.BaseAddress+m.RegionSize
}

func scanBoundedKeyObjects(ctx context.Context, mem objectProbeMemory, limits objectProbeLimits,
	report *PassiveProbeReport, verify func([]byte) (bool, error)) error {
	return scanBoundedObjects(ctx, mem, limits, report, false, verify)
}

func scanBoundedStringObjects(ctx context.Context, mem objectProbeMemory, limits objectProbeLimits,
	report *PassiveProbeReport, verify func([]byte) (bool, error)) error {
	return scanBoundedObjects(ctx, mem, limits, report, true, verify)
}

func scanBoundedObjects(ctx context.Context, mem objectProbeMemory, limits objectProbeLimits,
	report *PassiveProbeReport, stringLayout bool, verify func([]byte) (bool, error)) error {
	objectSize := 16
	if stringLayout {
		objectSize = 32
	}
	seen := map[[32]byte]bool{}
	defer clear(seen)
	query := func(addr uintptr) (windows.MemoryBasicInformation, error) {
		if err := ctx.Err(); err != nil {
			return windows.MemoryBasicInformation{}, err
		}
		if report.Queries >= limits.queries {
			return windows.MemoryBasicInformation{}, ErrPassiveProbeBudget
		}
		report.Queries++
		return mem.query(addr)
	}
	read := func(addr uintptr, b []byte) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if report.ReadBytes > limits.readBytes || uint64(len(b)) > limits.readBytes-report.ReadBytes {
			return ErrPassiveProbeBudget
		}
		// Charge requested bytes, including failed/partial reads.
		report.ReadBytes += uint64(len(b))
		if err := mem.read(addr, b); err != nil {
			report.ReadFailures++
			clear(b)
			return err
		}
		return nil
	}
	buf := make([]byte, probeChunk+objectSize-8)
	defer clear(buf)
	for addr := uintptr(0); addr < probeMaxAddress; {
		m, err := query(addr)
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			return nil // OS-reported end of the address space
		}
		if err != nil {
			return err
		}
		end := m.BaseAddress + m.RegionSize
		if end <= addr || m.BaseAddress > addr || end <= m.BaseAddress || end > probeMaxAddress {
			return errors.New("invalid passive memory region")
		}
		if !probeReadable(m) {
			addr = end
			continue
		}
		report.Regions++
		for off := m.BaseAddress; off < end; {
			n := min(uintptr(probeChunk), end-off)
			// Enough overlap for the selected aligned object layout, without
			// stitching across an unreadable gap.
			readN := n
			if off+n < end {
				readN = min(n+uintptr(objectSize-8), end-off)
			}
			b := buf[:readN]
			if err := read(off, b); err != nil {
				if errors.Is(err, ErrPassiveProbeBudget) || ctx.Err() != nil {
					return err
				}
				off += n
				continue
			}
			for i := int((8 - off%8) % 8); i+objectSize <= len(b) && uintptr(i) < n; i += 8 {
				if (i/8)%512 == 0 {
					if err := ctx.Err(); err != nil {
						return err
					}
				}
				if stringLayout {
					// Distinct hypothesis: pointer, unused 8 bytes, size, capacity.
					if binary.LittleEndian.Uint64(b[i+8:i+16]) != 0 ||
						binary.LittleEndian.Uint64(b[i+16:i+24]) != 32 ||
						binary.LittleEndian.Uint64(b[i+24:i+32]) != 47 {
						continue
					}
				} else {
					if binary.LittleEndian.Uint64(b[i+8:i+16]) != 32 {
						continue
					}
				}
				if report.Objects >= limits.objects {
					return ErrPassiveProbeBudget
				}
				report.Objects++
				ptr := uintptr(binary.LittleEndian.Uint64(b[i : i+8]))
				if ptr < 0x10000 || ptr > probeMaxAddress-32 {
					continue
				}
				target, err := query(ptr)
				if errors.Is(err, ErrPassiveProbeBudget) || ctx.Err() != nil {
					return errors.Join(err, ctx.Err())
				}
				if err != nil || !probeContains(target, ptr, 32) {
					continue
				}
				var key [32]byte
				if err := read(ptr, key[:]); err != nil {
					if errors.Is(err, ErrPassiveProbeBudget) || ctx.Err() != nil {
						return err
					}
					continue
				}
				hash := sha256.Sum256(key[:])
				if seen[hash] || !plausibleKeyBytes(key[:]) {
					clear(key[:])
					continue
				}
				if report.Candidates >= limits.candidates {
					clear(key[:])
					return ErrPassiveProbeBudget
				}
				report.Candidates++
				seen[hash] = true
				// No verification goroutine or queue survives cancellation.
				if err := ctx.Err(); err != nil {
					clear(key[:])
					return err
				}
				done, err := verify(key[:])
				clear(key[:])
				if err != nil {
					return err
				}
				if done {
					return nil
				}
			}
			clear(b)
			off += n
		}
		addr = end
	}
	return nil
}

type probeDirectory interface {
	ReadDir(int) ([]fs.DirEntry, error)
	Close() error
}

// Unlike filepath.WalkDir, directory reads never allocate the whole directory
// before the entry/cancellation limit can run. At most 9 batches of 128 entries
// are retained on the depth-limited traversal stack.
func walkProbeDirectories(ctx context.Context, base string, open func(string) (probeDirectory, error),
	visit func(string, fs.DirEntry) error) error {
	entries := 0
	var walk func(string, int) error
	walk = func(dir string, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if depth > 8 {
			return ErrPassiveProbeBudget
		}
		f, err := open(dir)
		if err != nil {
			return err
		}
		defer f.Close()
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			batch, readErr := f.ReadDir(128)
			if len(batch) > 128 || (len(batch) == 0 && readErr == nil) {
				return errors.New("invalid bounded directory response")
			}
			for _, d := range batch {
				if err := ctx.Err(); err != nil {
					return err
				}
				if entries >= 65536 {
					return ErrPassiveProbeBudget
				}
				entries++
				path := filepath.Join(dir, d.Name())
				if err := visit(path, d); err != nil {
					return err
				}
				if d.IsDir() {
					if err := walk(path, depth+1); err != nil {
						return err
					}
				}
			}
			if errors.Is(readErr, io.EOF) {
				return ctx.Err()
			}
			if readErr != nil {
				return readErr
			}
		}
	}
	return walk(base, 0)
}

func probeSourceDBs(ctx context.Context, root string) ([]windowsSourceDB, error) {
	base := filepath.Join(root, "db_storage")
	guard, err := openProbeRoot(base, os.OpenRoot)
	if err != nil {
		return nil, err
	}
	defer guard.Close()
	return probeSourceDBsInRoot(ctx, guard, base)
}

type probeRoot struct {
	*os.Root
	pin windows.Handle
}

func (r *probeRoot) Close() error {
	return errors.Join(r.Root.Close(), windows.CloseHandle(r.pin))
}

// Lstat's Windows file ID can be lazy/path-based. Instead capture an actual ID
// through a no-follow handle and retain that handle throughout the observation.
func openProbeDirectoryIdentity(path string) (windows.Handle, windows.ByHandleFileInformation, error) {
	var info windows.ByHandleFileInformation
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, info, err
	}
	h, err := windows.CreateFile(p, windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return 0, info, err
	}
	err = windows.GetFileInformationByHandle(h, &info)
	if err == nil && (info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 ||
		info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0) {
		err = errors.New("probe source root must be a real directory, not a link")
	}
	if err != nil {
		windows.CloseHandle(h)
		return 0, info, err
	}
	return h, info, nil
}

func sameProbeDirectory(a, b windows.ByHandleFileInformation) bool {
	return a.VolumeSerialNumber == b.VolumeSerialNumber &&
		a.FileIndexHigh == b.FileIndexHigh && a.FileIndexLow == b.FileIndexLow
}

func openProbeRoot(path string, open func(string) (*os.Root, error)) (*probeRoot, error) {
	pin, before, err := openProbeDirectoryIdentity(path)
	if err != nil {
		return nil, err
	}
	r, err := open(path)
	if err != nil {
		windows.CloseHandle(pin)
		return nil, err
	}
	fail := func() (*probeRoot, error) {
		r.Close()
		windows.CloseHandle(pin)
		return nil, errors.New("probe source root identity changed while opening")
	}
	d, err := r.Open(".")
	if err != nil {
		return fail()
	}
	var opened windows.ByHandleFileInformation
	statErr := windows.GetFileInformationByHandle(windows.Handle(d.Fd()), &opened)
	d.Close()
	afterPin, after, pathErr := openProbeDirectoryIdentity(path)
	if afterPin != 0 {
		defer windows.CloseHandle(afterPin)
	}
	if statErr != nil || pathErr != nil || !sameProbeDirectory(before, opened) || !sameProbeDirectory(after, opened) {
		return fail()
	}
	return &probeRoot{Root: r, pin: pin}, nil
}

func probeReparse(info fs.FileInfo) bool {
	// Junctions are ModeIrregular on Go >=1.23, not necessarily ModeSymlink.
	attrs, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return !ok || attrs.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0
}

func probeSourceDBsInRoot(ctx context.Context, guard *probeRoot, base string) ([]windowsSourceDB, error) {
	var out []windowsSourceDB
	files := 0
	err := walkProbeDirectories(ctx, ".", func(path string) (probeDirectory, error) {
		// Root.Open is handle-relative; it cannot follow an outside link even
		// if the directory entry changed after the previous batch was read.
		return guard.Open(path)
	}, func(path string, d fs.DirEntry) error {
		// Windows directory-enumeration Type can describe a junction as an
		// irregular entry rather than ModeSymlink. Check the actual entry
		// relative to the pinned root, not only cached enumeration flags.
		info, err := guard.Lstat(path)
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 || probeReparse(info) {
			return errors.New("probe source tree contains a link")
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(d.Name()), ".db") {
			return nil
		}
		if !d.Type().IsRegular() {
			return errors.New("probe database is not a regular file")
		}
		files++
		if files > 64 {
			return ErrPassiveProbeBudget
		}
		f, err := guard.Open(path)
		if err != nil {
			return err
		}
		var header [16]byte
		_, readErr := io.ReadFull(f, header[:])
		closeErr := f.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		salt := hex.EncodeToString(header[:])
		if salt == hex.EncodeToString([]byte("SQLite format 3\x00")) {
			return nil
		}
		out = append(out, windowsSourceDB{rel: path, path: filepath.Join(base, path), salt: salt})
		return nil
	})
	return out, err
}

func readProbeVerifier(root *probeRoot, db windowsSourceDB) *sqlcipherVerifier {
	f, err := root.Open(db.rel)
	if err != nil {
		return nil
	}
	defer f.Close()
	page := make([]byte, sqlcipherPageSize)
	if _, err := io.ReadFull(f, page); err != nil || hex.EncodeToString(page[:16]) != db.salt {
		return nil
	}
	hmacSalt := make([]byte, 16)
	for i, b := range page[:16] {
		hmacSalt[i] = b ^ sqlcipherSaltMask
	}
	input := append([]byte(nil), page[16:sqlcipherPageSize-sqlcipherHMACSize]...)
	input = append(input, 1, 0, 0, 0)
	return &sqlcipherVerifier{
		salt: db.salt, dbSalt: page[:16], hmacSalt: hmacSalt,
		hmacInput: input, storedMAC: page[sqlcipherPageSize-sqlcipherHMACSize:],
	}
}

var errProbeSourceChanged = errors.New("source verification page changed during observation")
var errProbeAccountChanged = errors.New("selected account owner association changed during observation")

// Check the selected account again, not merely whether the same process still
// exists. A process can stay alive after closing the selected account's files.
// These are before/after observations, not proof of uninterrupted ownership.
func recheckProbeAccount(ctx context.Context, dbs []windowsSourceDB, expected accountProcess,
	observe func([]windowsSourceDB) ([]accountProcess, error)) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	owners, err := observe(dbs)
	if cancelErr := ctx.Err(); cancelErr != nil {
		return false, errors.Join(err, cancelErr)
	}
	if err != nil {
		return false, err
	}
	for _, owner := range owners {
		if owner.pid == expected.pid && owner.start == expected.start {
			return true, nil
		}
	}
	return false, errProbeAccountChanged
}

func recheckProbeSources(ctx context.Context, dbs []windowsSourceDB, before []*sqlcipherVerifier,
	read func(windowsSourceDB) *sqlcipherVerifier) (bool, error) {
	for i, db := range dbs {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		now := read(db)
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if now == nil || !bytes.Equal(now.dbSalt, before[i].dbSalt) ||
			!bytes.Equal(now.hmacInput, before[i].hmacInput) || !bytes.Equal(now.storedMAC, before[i].storedMAC) {
			return false, errProbeSourceChanged
		}
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	return true, nil
}

// ProbePassiveKeyObjects makes ONE bounded, read-only D0 raw-enc-key attempt.
// It does not invoke setup, KDF guessing, other routes, debugger, launch/close,
// config persistence or plaintext output. Candidate material is discarded.
// Its 90s deadline is cooperative; OS calls are not a hard real-time guarantee.
func ProbePassiveKeyObjects(ctx context.Context, opts PassiveProbeOptions) (report PassiveProbeReport, err error) {
	return probePassiveObjects(ctx, opts, nil)
}

func probePassiveObjects(ctx context.Context, opts PassiveProbeOptions, material *materialProbeSettings) (report PassiveProbeReport, err error) {
	start := time.Now()
	report.Route = "bounded-d0-raw-probe-v1"
	if material != nil {
		report.Route = "bounded-string-material-probe-v1"
		report.MaterialProfile = &material.profile.report
		report.Scope = "selected-db-on-miss; expand-to-all-source-dbs-only-after-selected-db-match"
		report.ProbeDBs = 1
	}
	report.Status = "failed"
	defer func() {
		report.ElapsedMS = time.Since(start).Milliseconds()
		if report.Status == "identity-or-source-invalidated" {
			return
		}
		if errors.Is(err, ErrPassiveProbeBudget) {
			report.Status = "budget-exhausted"
		} else if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			report.Status = "cancelled-or-timeout"
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if !filepath.IsAbs(opts.DBRoot) || !filepath.IsAbs(opts.ExePath) || opts.PID == 0 || opts.CreatedUTC.IsZero() {
		return report, errors.New("probe requires an explicit account and observed process identity")
	}
	base := filepath.Join(opts.DBRoot, "db_storage")
	source, err := openProbeRoot(base, os.OpenRoot)
	if err != nil {
		return report, err
	}
	if material != nil && material.retained != nil {
		// Transfer only the already-validated source root to the bounded
		// synchronous consumer. Its owner clears/closes it on every return.
		material.retained.source, material.retained.base = source, base
		material.retained.provenance = report.Route + "/module-sha256:" + material.profile.report.ModuleSHA256
	} else {
		defer source.Close()
	}
	dbs, err := probeSourceDBsInRoot(ctx, source, base)
	if err != nil {
		return report, err
	}
	report.SourceDBs = len(dbs)
	if len(dbs) == 0 || len(dbs) > 64 {
		return report, errors.New("probe requires 1..64 encrypted source databases")
	}
	var verifiers []*sqlcipherVerifier
	for _, db := range dbs {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		v := readProbeVerifier(source, db)
		if v == nil {
			return report, errors.New("cannot read a required source verification page")
		}
		verifiers = append(verifiers, v)
	}
	owners, err := accountFileOwners(dbs) // metadata only; never RmShutdown
	if err != nil {
		return report, err
	}
	var identity *accountProcess
	for i := range owners {
		p := &owners[i]
		if p.pid == opts.PID && time.Unix(0, p.start.Nanoseconds()).Equal(opts.CreatedUTC) {
			identity = p
		}
	}
	if identity == nil {
		return report, errors.New("observed process is not a verified owner of selected account files")
	}
	pin, exe, err := pinAccountProcess(*identity)
	if err != nil {
		return report, err
	}
	defer windows.CloseHandle(pin)
	if !strings.EqualFold(exe, opts.ExePath) {
		return report, errors.New("selected account process executable changed")
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ|windows.SYNCHRONIZE, false, opts.PID)
	if err != nil {
		return report, processAccessError(opts.PID, windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ|windows.SYNCHRONIZE, err)
	}
	defer windows.CloseHandle(h)
	checkIdentity := func() error {
		var c, e, k, u windows.Filetime
		state, waitErr := windows.WaitForSingleObject(h, 0)
		if waitErr != nil || state != uint32(windows.WAIT_TIMEOUT) {
			return errors.New("probe process exited")
		}
		if err := windows.GetProcessTimes(h, &c, &e, &k, &u); err != nil || c != identity.start {
			return errors.New("probe process identity changed")
		}
		return nil
	}
	if err := checkIdentity(); err != nil {
		return report, err
	}
	matched := make([]bool, len(dbs))
	verify := func(key []byte) (bool, error) {
		for i, v := range verifiers {
			if err := ctx.Err(); err != nil {
				return false, err
			}
			if matched[i] {
				continue
			}
			report.HMACChecks++
			if v.verify(key) {
				matched[i] = true
				report.VerifiedDBs++
			}
		}
		return report.VerifiedDBs == len(dbs), nil
	}
	scan := scanBoundedKeyObjects
	if material != nil {
		primary := -1
		for i, db := range dbs {
			if strings.EqualFold(filepath.Clean(db.rel), filepath.Clean(material.primaryDB)) {
				primary = i
			}
		}
		if primary < 0 {
			return report, errors.New("selected material probe database is not in the pinned account")
		}
		if err := checkProbeMaterialModule(ctx, h, opts.PID, material.profile, &report); err != nil {
			return report, err
		}
		var retain func(int, []byte)
		if material.retained != nil {
			retain = func(i int, key []byte) {
				material.retained.put(dbs[i].rel, key)
			}
		}
		verify = newMaterialProbeVerifierRetaining(ctx, verifiers, primary, material.profile.masks, &report, 512, retain)
		scan = scanBoundedStringObjects
	}
	err = scan(ctx, nativeObjectProbeMemory{h}, defaultObjectProbeLimits, &report, verify)
	err = errors.Join(err, ctx.Err())
	identityErr := checkIdentity()
	var sourceErr error
	report.SourceStable, sourceErr = recheckProbeSources(ctx, dbs, verifiers, func(db windowsSourceDB) *sqlcipherVerifier {
		return readProbeVerifier(source, db)
	})
	identityErr = errors.Join(identityErr, checkIdentity())
	var moduleErr error
	if material != nil && identityErr == nil {
		moduleErr = checkProbeMaterialModule(ctx, h, opts.PID, material.profile, &report)
		report.ModuleStable = moduleErr == nil
	}
	var accountErr error
	report.AccountStable, accountErr = recheckProbeAccount(ctx, dbs, *identity, accountFileOwners)
	identityErr = errors.Join(identityErr, checkIdentity())
	report.IdentityStable = identityErr == nil
	finalizeProbeStability(&report, material != nil, identityErr, sourceErr, moduleErr, accountErr)
	if err = errors.Join(err, identityErr, sourceErr, moduleErr, accountErr, ctx.Err()); err != nil {
		return report, err
	}
	if report.VerifiedDBs == len(dbs) {
		report.Status = "all-page1-hmac-verified"
	} else if report.VerifiedDBs > 0 {
		report.Status = "partial-page1-hmac-verified"
	} else {
		report.Status = "no-match-in-observed-regions"
	}
	return report, nil
}

// An unavailable post-check is not evidence of an identity change, but it
// cannot authorize success either. Cancellation is reported by the caller.
func finalizeProbeStability(report *PassiveProbeReport, material bool, identityErr, sourceErr, moduleErr, accountErr error) {
	if identityErr != nil || errors.Is(sourceErr, errProbeSourceChanged) || errors.Is(moduleErr, errProbeModuleChanged) ||
		errors.Is(accountErr, errProbeAccountChanged) {
		report.Status = "identity-or-source-invalidated"
	} else if !report.IdentityStable || !report.AccountStable || !report.SourceStable || (material && !report.ModuleStable) {
		report.Status = "validation-incomplete"
	} else {
		return
	}
	report.VerifiedDBs = 0
	report.RawDBs, report.DirectPassphraseDBs, report.NormalizedDBs = 0, 0, 0
}
