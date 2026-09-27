//go:build windows

package wxkey

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/windows"
	"weixin-key/internal/testutil"
)

type fakeObjectProbeMemory struct {
	data      []byte
	readCalls int
	failRead  bool
}

const fakeProbeBase = uintptr(0x10000)

func (m *fakeObjectProbeMemory) query(addr uintptr) (windows.MemoryBasicInformation, error) {
	if addr < fakeProbeBase {
		return windows.MemoryBasicInformation{BaseAddress: 0, RegionSize: fakeProbeBase}, nil
	}
	if addr >= fakeProbeBase+uintptr(len(m.data)) {
		return windows.MemoryBasicInformation{}, windows.ERROR_INVALID_PARAMETER
	}
	return windows.MemoryBasicInformation{
		BaseAddress: fakeProbeBase, AllocationBase: fakeProbeBase, RegionSize: uintptr(len(m.data)),
		State: windows.MEM_COMMIT, Type: probePrivate, Protect: windows.PAGE_READWRITE,
	}, nil
}

func (m *fakeObjectProbeMemory) read(addr uintptr, b []byte) error {
	m.readCalls++
	if m.failRead {
		return windows.ERROR_PARTIAL_COPY
	}
	if addr < fakeProbeBase || addr+uintptr(len(b)) > fakeProbeBase+uintptr(len(m.data)) {
		return windows.ERROR_PARTIAL_COPY
	}
	copy(b, m.data[addr-fakeProbeBase:])
	return nil
}

func setProbeObject(m *fakeObjectProbeMemory, offset, keyOffset int, key []byte) {
	binary.LittleEndian.PutUint64(m.data[offset:], uint64(fakeProbeBase)+uint64(keyOffset))
	binary.LittleEndian.PutUint64(m.data[offset+8:], 32)
	copy(m.data[keyOffset:], key)
}

func TestBoundedObjectProbeVerifiesSyntheticHMAC(t *testing.T) {
	path, _, salt, keyHex := testutil.MustNewPassphraseDB(t.TempDir(), "probe.db", 1)
	key, _ := hex.DecodeString(keyHex)
	m := &fakeObjectProbeMemory{data: make([]byte, probeChunk+256)}
	setProbeObject(m, probeChunk-8, probeChunk+64, key)
	v := newSQLCipherVerifier(path, salt)
	if v == nil {
		t.Fatal("fixture verifier unavailable")
	}
	var report PassiveProbeReport
	calls := 0
	err := scanBoundedKeyObjects(context.Background(), m, defaultObjectProbeLimits, &report, func(b []byte) (bool, error) {
		calls++
		if !v.verify(b) {
			t.Fatal("cross-chunk candidate failed synthetic HMAC")
		}
		return true, nil
	})
	if err != nil || calls != 1 || report.Candidates != 1 || report.Objects != 1 {
		t.Fatalf("unexpected counters: calls=%d candidates=%d objects=%d err=%v", calls, report.Candidates, report.Objects, err)
	}
}

func TestBoundedObjectProbeCancellationInsideCandidates(t *testing.T) {
	m := &fakeObjectProbeMemory{data: make([]byte, 4096)}
	for i := range 8 {
		key := make([]byte, 32)
		for j := range key {
			key[j] = byte(j + i + 1)
		}
		setProbeObject(m, i*16, 1024+i*32, key)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var report PassiveProbeReport
	calls := 0
	err := scanBoundedKeyObjects(ctx, m, defaultObjectProbeLimits, &report, func(b []byte) (bool, error) {
		calls++
		cancel()
		return false, nil
	})
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("cancellation not respected: calls=%d err=%v", calls, err)
	}
}

func TestBoundedObjectProbeBudgets(t *testing.T) {
	for _, field := range []string{"read", "query", "object", "candidate"} {
		t.Run(field, func(t *testing.T) {
			m := &fakeObjectProbeMemory{data: make([]byte, 4096)}
			key := make([]byte, 32)
			for i := range key {
				key[i] = byte(i + 1)
			}
			setProbeObject(m, 0, 512, key)
			key[0]++
			setProbeObject(m, 16, 544, key)
			limits := defaultObjectProbeLimits
			switch field {
			case "read":
				limits.readBytes = 100
			case "query":
				limits.queries = 1
			case "object":
				limits.objects = 1
			case "candidate":
				limits.candidates = 1
			}
			var r PassiveProbeReport
			err := scanBoundedKeyObjects(context.Background(), m, limits, &r, func([]byte) (bool, error) { return false, nil })
			if !errors.Is(err, ErrPassiveProbeBudget) {
				t.Fatalf("expected explicit budget result, got %v", err)
			}
			if r.ReadBytes > limits.readBytes || r.Queries > limits.queries || r.Objects > limits.objects || r.Candidates > limits.candidates {
				t.Fatal("hard counter bound exceeded")
			}
		})
	}
}

func TestBoundedObjectProbeDedupAndReadFailure(t *testing.T) {
	m := &fakeObjectProbeMemory{data: make([]byte, 4096)}
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	setProbeObject(m, 0, 512, key)
	setProbeObject(m, 16, 544, key) // two pointers, same content
	var r PassiveProbeReport
	calls := 0
	verify := func([]byte) (bool, error) { calls++; return false, nil }
	if err := scanBoundedKeyObjects(context.Background(), m, defaultObjectProbeLimits, &r, verify); err != nil || calls != 1 {
		t.Fatalf("dedup: calls=%d err=%v", calls, err)
	}
	m.failRead = true
	r, calls = PassiveProbeReport{}, 0
	if err := scanBoundedKeyObjects(context.Background(), m, defaultObjectProbeLimits, &r, verify); err != nil || calls != 0 || r.ReadFailures != 1 {
		t.Fatalf("failed chunk was verified: calls=%d failures=%d err=%v", calls, r.ReadFailures, err)
	}
}

func TestProbeRejectsNonPrivateOrInvalidRanges(t *testing.T) {
	m := &fakeObjectProbeMemory{data: make([]byte, 64)}
	base, _ := m.query(fakeProbeBase)
	if !probeContains(base, fakeProbeBase, 32) {
		t.Fatal("valid private range rejected")
	}
	for _, mutate := range []func(*windows.MemoryBasicInformation){
		func(m *windows.MemoryBasicInformation) { m.Type = 0x1000000 },
		func(m *windows.MemoryBasicInformation) { m.Protect |= windows.PAGE_GUARD },
		func(m *windows.MemoryBasicInformation) { m.Protect = windows.PAGE_EXECUTE_READWRITE },
		func(m *windows.MemoryBasicInformation) { m.RegionSize = ^uintptr(0) },
	} {
		r := base
		mutate(&r)
		if probeContains(r, fakeProbeBase, 32) {
			t.Fatal("disallowed range accepted")
		}
	}
	if probeContains(base, ^uintptr(0)-4, 32) || probeContains(base, fakeProbeBase+48, 32) {
		t.Fatal("cross-region/overflow read accepted")
	}
}

func TestPassiveProbeCancelledBeforeAnyIO(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, err := ProbePassiveKeyObjects(ctx, PassiveProbeOptions{DBRoot: `Z:\must-not-open`})
	if !errors.Is(err, context.Canceled) || r.ReadBytes != 0 || r.Status != "cancelled-or-timeout" {
		t.Fatal("early cancel was ignored")
	}
	b, err := json.Marshal(r)
	if err != nil || bytes.Contains(b, []byte("key_hex")) || bytes.Contains(b, []byte("salt")) {
		t.Fatal("summary exposes material fields")
	}
}

func TestProbeSourceDatabaseLimit(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "db_storage")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for i := range 65 {
		name := filepath.Join(dir, hex.EncodeToString([]byte{byte(i)})+".db")
		if err := os.WriteFile(name, bytes.Repeat([]byte{1}, 16), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	dbs, err := probeSourceDBs(context.Background(), root)
	if !errors.Is(err, ErrPassiveProbeBudget) || len(dbs) != 64 {
		t.Fatalf("source limit: %d err=%v", len(dbs), err)
	}
}

type fakeProbeEntry struct{ dir bool }

func (f fakeProbeEntry) Name() string               { return "synthetic" }
func (f fakeProbeEntry) IsDir() bool                { return f.dir }
func (f fakeProbeEntry) Type() fs.FileMode          { return 0 }
func (f fakeProbeEntry) Info() (fs.FileInfo, error) { return nil, errors.New("not used") }

type fakeProbeDirectory struct {
	remaining int
	calls     int
	closed    bool
	directory bool
	afterRead func()
}

func (f *fakeProbeDirectory) ReadDir(n int) ([]fs.DirEntry, error) {
	f.calls++
	if n != 128 {
		return nil, errors.New("unbounded or unexpected directory batch size")
	}
	b := make([]fs.DirEntry, min(n, f.remaining))
	for i := range b {
		b[i] = fakeProbeEntry{f.directory}
	}
	f.remaining -= len(b)
	if f.afterRead != nil {
		f.afterRead()
	}
	if len(b) == 0 {
		return nil, io.EOF
	}
	return b, nil
}
func (f *fakeProbeDirectory) Close() error { f.closed = true; return nil }

func TestProbeDirectoryBatchEntryAndDepthLimits(t *testing.T) {
	f := &fakeProbeDirectory{remaining: 70000}
	visits := 0
	err := walkProbeDirectories(context.Background(), "synthetic", func(string) (probeDirectory, error) { return f, nil },
		func(string, fs.DirEntry) error { visits++; return nil })
	if !errors.Is(err, ErrPassiveProbeBudget) || visits != 65536 || f.calls != 513 || !f.closed {
		t.Fatalf("entry bound visits=%d calls=%d closed=%v err=%v", visits, f.calls, f.closed, err)
	}
	var opened []*fakeProbeDirectory
	err = walkProbeDirectories(context.Background(), "synthetic", func(string) (probeDirectory, error) {
		d := &fakeProbeDirectory{remaining: 1, directory: true}
		opened = append(opened, d)
		return d, nil
	}, func(string, fs.DirEntry) error { return nil })
	if !errors.Is(err, ErrPassiveProbeBudget) || len(opened) != 9 {
		t.Fatalf("depth bound opens=%d err=%v", len(opened), err)
	}
	for _, d := range opened {
		if !d.closed {
			t.Fatal("nested directory left open")
		}
	}
}

func TestProbeDirectoryCancellationAfterBatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f := &fakeProbeDirectory{remaining: 1000, afterRead: cancel}
	visits := 0
	err := walkProbeDirectories(ctx, "synthetic", func(string) (probeDirectory, error) { return f, nil },
		func(string, fs.DirEntry) error { visits++; return nil })
	if !errors.Is(err, context.Canceled) || f.calls != 1 || visits != 0 || !f.closed {
		t.Fatal("directory cancellation did not stop at the batch boundary")
	}
}

func TestProbeSourceRecheckCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbs := []windowsSourceDB{{}, {}}
	v := &sqlcipherVerifier{}
	reads := 0
	stable, err := recheckProbeSources(ctx, dbs, []*sqlcipherVerifier{v, v}, func(windowsSourceDB) *sqlcipherVerifier {
		reads++
		cancel()
		return v
	})
	if stable || !errors.Is(err, context.Canceled) || reads != 1 {
		t.Fatalf("cancelled source check falsely passed: stable=%v reads=%d err=%v", stable, reads, err)
	}
}

func TestProbeSourceRecheckRejectsChange(t *testing.T) {
	before := &sqlcipherVerifier{dbSalt: []byte{1}}
	stable, err := recheckProbeSources(context.Background(), []windowsSourceDB{{}}, []*sqlcipherVerifier{before},
		func(windowsSourceDB) *sqlcipherVerifier { return &sqlcipherVerifier{dbSalt: []byte{2}} })
	if stable || !errors.Is(err, errProbeSourceChanged) {
		t.Fatal("changed source was accepted")
	}
}

func makeProbeTestJunction(t *testing.T, link, target string) {
	t.Helper()
	// Directory junctions do not require enabling SeCreateSymbolicLinkPrivilege.
	// Both paths belong to t.TempDir fixtures; never refer to user data.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ps := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	cmd := exec.CommandContext(ctx, ps, "-NoProfile", "-NonInteractive", "-Command",
		`New-Item -ItemType Junction -Path $env:WX_PROBE_TEST_LINK -Target $env:WX_PROBE_TEST_TARGET -ErrorAction Stop | Out-Null`)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Dir = t.TempDir() // keep any shell cache fallback outside frozen sources
	cmd.Env = append(os.Environ(), "WX_PROBE_TEST_LINK="+link, "WX_PROBE_TEST_TARGET="+target)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create owned junction fixture: %v (%s)", err, b)
	}
}

func TestProbeRootAndChildJunctionRejectedBeforeDatabaseRead(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "must-not-read.db"), []byte("too short for a salt"), 0o600); err != nil {
		t.Fatal(err)
	}
	account := t.TempDir()
	base := filepath.Join(account, "db_storage")
	makeProbeTestJunction(t, base, outside)
	opens := 0
	r, err := openProbeRoot(base, func(path string) (*os.Root, error) {
		opens++
		return os.OpenRoot(path)
	})
	if r != nil {
		r.Close()
	}
	if err == nil || opens != 0 || !strings.Contains(err.Error(), "not a link") {
		t.Fatalf("linked root was opened: opens=%d err=%v", opens, err)
	}
	dbs, err := probeSourceDBs(context.Background(), account)
	if err == nil || len(dbs) != 0 || !strings.Contains(err.Error(), "not a link") {
		t.Fatal("linked root reached DB discovery")
	}
	account = t.TempDir()
	base = filepath.Join(account, "db_storage")
	if err := os.Mkdir(base, 0o700); err != nil {
		t.Fatal(err)
	}
	makeProbeTestJunction(t, filepath.Join(base, "linked-child"), outside)
	dbs, err = probeSourceDBs(context.Background(), account)
	if err == nil || len(dbs) != 0 || !strings.Contains(err.Error(), "contains a link") {
		entries, listErr := os.ReadDir(base)
		t.Logf("fixture base=%s listing=%d error=%v", base, len(entries), listErr)
		for _, d := range entries {
			info, e := os.Lstat(filepath.Join(base, d.Name()))
			t.Logf("fixture entry=%s type=%v lstat=%v error=%v", d.Name(), d.Type(), info, e)
		}
		guard, e := openProbeRoot(base, os.OpenRoot)
		if e == nil {
			defer guard.Close()
			f, e := guard.Open(".")
			if e == nil {
				defer f.Close()
				dirs, e := f.ReadDir(128)
				t.Logf("rooted listing=%d error=%v", len(dirs), e)
			}
		}
		t.Fatalf("child junction not rejected: dbs=%d err=%v", len(dbs), err)
	}
}

func TestProbeRootRejectsOpenIdentityReplacement(t *testing.T) {
	selected, substituted := t.TempDir(), t.TempDir()
	r, err := openProbeRoot(selected, func(string) (*os.Root, error) {
		// Deterministically simulate substitution between Lstat and OpenRoot.
		return os.OpenRoot(substituted)
	})
	if r != nil {
		r.Close()
	}
	if err == nil || !strings.Contains(err.Error(), "identity changed") {
		t.Fatal("different opened root was accepted")
	}
}

func TestProbeRootRejectsSamePathDirectoryReplacement(t *testing.T) {
	base := t.TempDir()
	selected := filepath.Join(base, "selected")
	replacement := filepath.Join(base, "replacement")
	moved := filepath.Join(base, "original-pinned")
	for _, p := range []string{selected, replacement} {
		if err := os.Mkdir(p, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	swapped := false
	r, err := openProbeRoot(selected, func(path string) (*os.Root, error) {
		// Both rename targets are fixed children of this test's own TempDir.
		if err := os.Rename(selected, moved); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(replacement, selected); err != nil {
			t.Fatal(err)
		}
		swapped = true
		return os.OpenRoot(path) // SAME path now denotes a different directory
	})
	if r != nil {
		r.Close()
	}
	if !swapped || r != nil || err == nil || !strings.Contains(err.Error(), "identity changed") {
		t.Fatalf("same-path replacement was accepted: swapped=%v err=%v", swapped, err)
	}
}

func TestProbeRootedVerifierUsesSyntheticHMAC(t *testing.T) {
	dir := t.TempDir()
	_, _, salt, keyHex := testutil.MustNewPassphraseDB(dir, "synthetic.db", 1)
	r, err := openProbeRoot(dir, os.OpenRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	v := readProbeVerifier(r, windowsSourceDB{rel: "synthetic.db", salt: salt})
	key, _ := hex.DecodeString(keyHex)
	if v == nil || !v.verify(key) {
		t.Fatal("root-relative verifier rejected synthetic key")
	}
	key[0] ^= 1
	if v.verify(key) {
		t.Fatal("root-relative verifier accepted wrong key")
	}
	if readProbeVerifier(r, windowsSourceDB{rel: "../outside.db", salt: salt}) != nil {
		t.Fatal("outside-root verifier path accepted")
	}
}
