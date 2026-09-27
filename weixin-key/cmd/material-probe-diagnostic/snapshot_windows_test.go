//go:build windows && amd64

package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
	"weixin-key/internal/testutil"
)

func TestMaterialSnapshotNativeSyntheticAndCleanup(t *testing.T) {
	root, scratch := t.TempDir(), t.TempDir()
	dbDir := filepath.Join(root, "db_storage", "message")
	if err := os.MkdirAll(dbDir, 0o700); err != nil {
		t.Fatal(err)
	}
	key, salt := bytes.Repeat([]byte{0x31}, 32), bytes.Repeat([]byte{0x72}, 16)
	path := filepath.Join(dbDir, "synthetic.db")
	if err := testutil.BuildEncryptedSQLiteDB(path, key, salt,
		"CREATE TABLE Msg_synthetic(id INTEGER PRIMARY KEY, body TEXT)",
		testutil.PadInsertValuesSQL("Msg_synthetic", "", "(999, '"+testutil.PadBlob()+"')"),
		"INSERT INTO Msg_synthetic VALUES(1,'synthetic only')", "DELETE FROM Msg_synthetic WHERE id=999"); err != nil {
		t.Fatal(err)
	}
	opts := diagnosticOptions{snapshot: true, scratch: scratch, schemaCheck: true}
	opts.probe.DBRoot, opts.probe.PrimaryDB = root, `message\synthetic.db`
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report, err := checkMaterialSnapshot(context.Background(), opts, path, key)
	if err != nil {
		t.Fatal(err)
	}
	if !report.IntegrityOK || !report.PrivateDirectoryVerified || !report.EphemeralDirectoryRemoved ||
		report.Pages < 2 || report.MessageTables != 1 || report.MessageTableRows != 1 {
		t.Fatalf("incomplete synthetic snapshot check: %+v", report)
	}
	if len(report.MessageSchemas) != 1 || report.MessageSchemas[0].Tables != 1 ||
		len(report.MessageSchemas[0].Columns) != 2 {
		t.Fatal("message column shapes missing")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("snapshot check changed source")
	}
	left, err := os.ReadDir(scratch)
	if err != nil || len(left) != 0 {
		t.Fatal("private snapshot material left behind")
	}
	key[0]++
	if _, err := checkMaterialSnapshot(context.Background(), opts, path, key); err == nil {
		t.Fatal("wrong key accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := checkMaterialSnapshot(ctx, opts, path, key); !errors.Is(err, context.Canceled) {
		t.Fatal("early cancellation ignored")
	}
}

func TestMaterialSnapshotDirectoryPrivateAtCreation(t *testing.T) {
	// The production creator uses SECURITY_ATTRIBUTES at CreateDirectory;
	// both its immediate handle check and a child's inherited ACL must agree.
	parent := t.TempDir()
	dir, pin, err := privateSnapshotDirectory(parent)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(pin)
	child := filepath.Join(dir, "synthetic.txt")
	f, err := os.Create(child)
	if err != nil {
		t.Fatal(err)
	}
	sd, err := windows.GetSecurityInfo(windows.Handle(f.Fd()), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		f.Close()
		t.Fatal(err)
	}
	acl, _, err := sd.DACL()
	if err != nil || acl == nil || acl.AceCount != 1 {
		f.Close()
		t.Fatal("child inherited a permissive ACL")
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(acl, 0, &ace); err != nil {
		f.Close()
		t.Fatal(err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		f.Close()
		t.Fatal(err)
	}
	sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	if !sid.Equals(user.User.Sid) || ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
		f.Close()
		t.Fatal("child is not current-user-only")
	}
	runtime.KeepAlive(sd)
	f.Close()
	if err := os.Remove(child); err != nil {
		t.Fatal(err)
	}
}
