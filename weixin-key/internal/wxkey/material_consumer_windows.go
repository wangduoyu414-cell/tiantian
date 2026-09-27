//go:build windows

package wxkey

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// VerifiedPassiveKeys is a synchronous, callback-scoped set of derived per-DB
// keys, never the observed or normalized passphrase. It is not serializable.
// Do not retain it or use it concurrently. WithKey's bytes are borrowed only
// until that callback returns. Consumers must not log or retain those bytes.
type VerifiedPassiveKeys struct {
	keys       map[string][32]byte
	ready      bool
	source     *probeRoot
	base       string
	provenance string
}

func (VerifiedPassiveKeys) Format(w fmt.State, _ rune) {
	_, _ = io.WriteString(w, "[redacted verified passive keys]")
}
func (VerifiedPassiveKeys) MarshalJSON() ([]byte, error) {
	return nil, errors.New("verified passive material must not be serialized")
}
func (k *VerifiedPassiveKeys) put(relativeDB string, key []byte) {
	if k.keys == nil {
		k.keys = map[string][32]byte{}
	}
	var copyKey [32]byte
	copy(copyKey[:], key)
	k.keys[strings.ToLower(filepath.Clean(relativeDB))] = copyKey
	clear(copyKey[:])
}
func (k *VerifiedPassiveKeys) clear() {
	k.ready = false
	for name := range k.keys {
		k.keys[name] = [32]byte{}
		delete(k.keys, name)
	}
	k.keys = nil
	if k.source != nil {
		_ = k.source.Close()
		k.source = nil
	}
	k.base = ""
	k.provenance = ""
}

// openConsumerSourceIdentity is read-only and excludes reparse traversal of
// the final component, as well as replacement while its handle is retained.
func openConsumerSourceIdentity(path string, directory bool) (windows.Handle, windows.ByHandleFileInformation, error) {
	var info windows.ByHandleFileInformation
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, info, err
	}
	flags, access := uint32(windows.FILE_FLAG_OPEN_REPARSE_POINT), uint32(windows.FILE_READ_ATTRIBUTES)
	if directory {
		flags |= windows.FILE_FLAG_BACKUP_SEMANTICS
	} else {
		access |= windows.GENERIC_READ
	}
	h, err := windows.CreateFile(name, access, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, flags, 0)
	if err != nil {
		return 0, info, err
	}
	err = windows.GetFileInformationByHandle(h, &info)
	if err != nil || info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 ||
		(info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0) != directory {
		windows.CloseHandle(h)
		return 0, info, errors.New("consumer source identity is invalid or redirected")
	}
	return h, info, nil
}

// WithSource binds native path-based consumption to the SAME rooted directory
// retained by the observation, not a newly opened account root. Both root and
// file identities are pinned without denying ordinary writers.
func (k *VerifiedPassiveKeys) WithSource(ctx context.Context, relativeDB string, consume func(string, []byte) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if k == nil || !k.ready || k.source == nil || consume == nil ||
		!filepath.IsLocal(relativeDB) || relativeDB == "." {
		return errors.New("verified source is unavailable or selection is invalid")
	}
	if _, ok := k.keys[strings.ToLower(filepath.Clean(relativeDB))]; !ok {
		return errors.New("no verified material for the selected source")
	}
	var original windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(k.source.pin, &original); err != nil {
		return err
	}
	rootPin, current, err := openConsumerSourceIdentity(k.base, true)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(rootPin)
	if !sameProbeDirectory(original, current) {
		return errors.New("captured source root identity changed")
	}
	// This open is relative to the original, retained root handle.
	f, err := k.source.Open(relativeDB)
	if err != nil {
		return err
	}
	defer f.Close()
	var opened windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(windows.Handle(f.Fd()), &opened); err != nil {
		return err
	}
	path := filepath.Join(k.base, relativeDB)
	filePin, pinned, err := openConsumerSourceIdentity(path, false)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(filePin)
	if !sameProbeDirectory(opened, pinned) {
		return errors.New("captured source file identity changed")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	err = k.WithKey(relativeDB, func(key []byte) error { return consume(path, key) })
	// Check the requested path once more while original handles are still
	// alive. Failure does not turn a consumer result into accepted success.
	afterPin, after, afterErr := openConsumerSourceIdentity(path, false)
	if afterPin != 0 {
		windows.CloseHandle(afterPin)
	}
	if afterErr == nil && !sameProbeDirectory(pinned, after) {
		afterErr = errors.New("consumer source path changed")
	}
	return errors.Join(err, afterErr, ctx.Err())
}

// WithKey permits one explicit DB's authenticated raw key to be consumed in
// process. It never falls back to another DB, a passphrase, config or env.
func (k *VerifiedPassiveKeys) WithKey(relativeDB string, consume func([]byte) error) error {
	if k == nil || !k.ready || consume == nil || !filepath.IsLocal(relativeDB) || relativeDB == "." {
		return errors.New("verified material is unavailable or selection is invalid")
	}
	key, ok := k.keys[strings.ToLower(filepath.Clean(relativeDB))]
	if !ok {
		return errors.New("no verified material for the selected database")
	}
	defer clear(key[:])
	return consume(key[:])
}

// WithVerifiedPassiveKeys is an explicit material-consuming alternative to
// the discard-only diagnostic. It performs the same bounded observation.
// Only ALL discovered DBs verified, with every post-check completed and no
// error/cancellation, can reach consume. Process handles close first; the
// original source root stays pinned through the synchronous consumer.
// This API itself never writes config, output or WeChat.
func WithVerifiedPassiveKeys(ctx context.Context, opts MaterialProbeOptions,
	consume func(*VerifiedPassiveKeys) error) (PassiveProbeReport, error) {
	var keys VerifiedPassiveKeys
	defer keys.clear()
	if consume == nil {
		return PassiveProbeReport{Status: "failed"}, errors.New("verified material consumer is required")
	}
	report, err := probePassiveMaterialObjects(ctx, opts, &keys)
	if err != nil {
		return report, err
	}
	err = consumeVerifiedPassiveKeys(ctx, report, &keys, consume)
	return report, err
}

func consumeVerifiedPassiveKeys(ctx context.Context, report PassiveProbeReport, keys *VerifiedPassiveKeys,
	consume func(*VerifiedPassiveKeys) error) error {
	defer keys.clear()
	if err := ctx.Err(); err != nil {
		return err
	}
	if report.Status != "all-page1-hmac-verified" || report.SourceDBs == 0 ||
		report.SourceDBs != report.VerifiedDBs || len(keys.keys) != report.SourceDBs ||
		!report.IdentityStable || !report.AccountStable || !report.SourceStable || !report.ModuleStable {
		return errors.New("complete stable material verification is required before consumption")
	}
	keys.ready = true
	return consume(keys)
}
