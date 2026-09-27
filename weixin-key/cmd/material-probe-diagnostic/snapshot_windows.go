//go:build windows

package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	_ "modernc.org/sqlite"
	"weixin-key/internal/pathguard"
	"weixin-key/internal/sqliteengine"
	"weixin-key/internal/wxkey"
)

type materialSnapshotReport struct {
	Status                    string                 `json:"status"`
	Method                    string                 `json:"method"`
	Pages                     int64                  `json:"pages"`
	Tables                    int                    `json:"tables"`
	MessageTables             int                    `json:"message_tables"`
	MessageTableRows          int64                  `json:"message_table_rows"`
	IntegrityOK               bool                   `json:"integrity_ok"`
	PrivateDirectoryVerified  bool                   `json:"private_directory_verified"`
	EphemeralDirectoryRemoved bool                   `json:"ephemeral_directory_removed"`
	MessageSchemas            []messageSchemaShape   `json:"message_schemas,omitempty"`
	MessageSemantics          *messageSemanticReport `json:"message_semantics,omitempty"`
}

// Create the directory with its private DACL atomically. A later ACL change
// would not revoke handles acquired during an initially permissive interval.
func privateSnapshotDirectory(parent string) (string, windows.Handle, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", 0, err
	}
	dir := filepath.Join(parent, ".material-check-"+hex.EncodeToString(nonce[:]))
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", 0, err
	}
	sd, err := windows.SecurityDescriptorFromString("O:" + user.User.Sid.String() + "D:P(A;OICI;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return "", 0, err
	}
	name, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return "", 0, err
	}
	attrs := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	err = windows.CreateDirectory(name, &attrs)
	runtime.KeepAlive(sd)
	if err != nil {
		// Includes a random-name collision: never remove an existing directory.
		return "", 0, err
	}
	fail := func(err error) (string, windows.Handle, error) { _ = os.Remove(dir); return "", 0, err }
	h, err := windows.CreateFile(name, windows.READ_CONTROL|windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return fail(err)
	}
	closeFail := func(err error) (string, windows.Handle, error) { windows.CloseHandle(h); return fail(err) }
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &info); err != nil {
		return closeFail(err)
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 ||
		info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 {
		return closeFail(errors.New("scratch directory identity is invalid"))
	}
	// Query the resolved handle; do not treat chmod/requested SDDL as proof.
	actual, err := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return closeFail(err)
	}
	owner, _, err := actual.Owner()
	if err != nil || !owner.Equals(user.User.Sid) {
		return closeFail(errors.New("scratch directory owner identity is invalid"))
	}
	control, _, err := actual.Control()
	if err != nil {
		return closeFail(err)
	}
	acl, _, err := actual.DACL()
	if err != nil || acl == nil || acl.AceCount != 1 || control&windows.SE_DACL_PROTECTED == 0 {
		return closeFail(errors.New("scratch directory DACL is not protected/private"))
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(acl, 0, &ace); err != nil {
		return closeFail(err)
	}
	sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	private := ace.Header.AceType == windows.ACCESS_ALLOWED_ACE_TYPE && ace.Header.AceFlags == 3 &&
		ace.Mask == windows.ACCESS_MASK(0x001F01FF) && sid.Equals(user.User.Sid)
	runtime.KeepAlive(actual)
	if !private {
		return closeFail(errors.New("scratch directory is not current-user-only with child inheritance"))
	}
	return dir, h, nil
}

// Check only the explicitly selected database. No row text, table names, salt,
// material, config writes, exports or persistent decrypted DB are returned.
func checkMaterialSnapshot(ctx context.Context, opts diagnosticOptions, source string, key []byte) (report materialSnapshotReport, retErr error) {
	report.Status = "failed"
	if err := ctx.Err(); err != nil {
		return report, err
	}
	scratch, err := pathguard.ResolveOutput(opts.scratch, opts.probe.DBRoot)
	if err != nil {
		return report, err
	}
	if !filepath.IsLocal(opts.probe.PrimaryDB) || opts.probe.PrimaryDB == "." ||
		!strings.EqualFold(filepath.Clean(source), filepath.Join(opts.probe.DBRoot, "db_storage", opts.probe.PrimaryDB)) {
		return report, errors.New("invalid snapshot database selection")
	}
	// Production invokes this only inside VerifiedPassiveKeys.WithSource,
	// which retains the original observation root and pins the exact file.
	// ResolveForDB's explicit raw key re-verifies the CURRENT source. It
	// never consults env/config when this explicit material is supplied.
	resolved, err := wxkey.NewKeyResolver().ResolveForDB(source,
		wxkey.ResolveOptions{EncKeyHex: hex.EncodeToString(key), NoConfig: true})
	if err != nil {
		return report, err
	}
	defer func() { resolved.EncKeyHex = "" }() // strings cannot guarantee zeroization
	dir, pin, err := privateSnapshotDirectory(scratch)
	if err != nil {
		return report, err
	}
	report.PrivateDirectoryVerified = true
	plain := filepath.Join(dir, "snapshot.db")
	defer func() {
		// Only this run's exact plaintext file and now-empty private directory.
		// Never recursively remove or enumerate a user-controlled tree.
		removeErr := os.Remove(plain)
		if errors.Is(removeErr, os.ErrNotExist) {
			removeErr = nil
		}
		closeErr := windows.CloseHandle(pin)
		dirErr := os.Remove(dir)
		report.EphemeralDirectoryRemoved = removeErr == nil && closeErr == nil && dirErr == nil
		if !report.EphemeralDirectoryRemoved {
			report.Status = "cleanup-failed"
			retErr = errors.Join(retErr, errors.New("private ephemeral snapshot cleanup failed"))
		}
	}()
	if err := sqliteengine.Backup(ctx, source, plain, resolved.EncKeyHex, resolved.SaltHex); err != nil {
		return report, err
	}
	report.Method = "native-read-transaction-backup-and-go-all-page-authentication/" + sqliteengine.Version
	info, err := os.Stat(plain)
	if err != nil {
		return report, err
	}
	report.Pages = info.Size() / 4096
	path := filepath.ToSlash(plain)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u := url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro&immutable=1"}
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return report, err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	var integrity string
	if err := db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		return report, errors.New("snapshot SQLite integrity check failed")
	}
	report.IntegrityOK = true
	rows, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name LIMIT 4097")
	if err != nil {
		return report, errors.New("cannot enumerate snapshot schema")
	}
	var messageTables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return report, err
		}
		report.Tables++
		if strings.HasPrefix(strings.ToLower(name), "msg_") {
			messageTables = append(messageTables, name)
		}
	}
	err = errors.Join(rows.Err(), rows.Close())
	if err != nil || report.Tables > 4096 {
		return report, errors.New("snapshot schema failed or exceeded its limit")
	}
	for _, name := range messageTables {
		var count int64
		quoted := `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM "+quoted).Scan(&count); err != nil {
			return report, errors.New("cannot count snapshot message table")
		}
		report.MessageTables++
		report.MessageTableRows += count
	}
	if opts.schemaCheck {
		report.MessageSchemas, err = inspectMessageSchemaShapes(ctx, db, messageTables)
		if err != nil {
			return report, err
		}
	}
	if opts.semanticCheck {
		semantics, e := inspectMessageSemantics(ctx, db, messageTables)
		if e != nil {
			return report, e
		}
		report.MessageSemantics = &semantics
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	report.Status = "full-database-verified-and-queryable"
	return report, nil
}
