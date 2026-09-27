//go:build windows && amd64

package sqliteengine

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
	"weixin-key/internal/testutil"
	"weixin-key/internal/wcdb"
)

func TestNativeBackupQueriesIndependentlyEncryptedFixture(t *testing.T) {
	key, salt := make([]byte, 32), make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "source.db")
	if err := testutil.BuildEncryptedSQLiteDB(source, key, salt,
		"CREATE TABLE test_messages(id INTEGER PRIMARY KEY, body TEXT)",
		testutil.PadInsertValuesSQL("test_messages", "", "(999, '"+testutil.PadBlob()+"')"),
		"INSERT INTO test_messages VALUES(1, 'synthetic message')",
		"DELETE FROM test_messages WHERE id=999",
	); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "out.db")
	if err := Backup(context.Background(), source, out, hex.EncodeToString(key), hex.EncodeToString(salt)); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", out)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow("SELECT count(*) FROM test_messages").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count=%d", count)
	}
}

func nativeFixture(t *testing.T) (string, string, string, uintptr) {
	t.Helper()
	if err := initialize(); err != nil {
		t.Fatal(err)
	}
	key, salt := strings.Repeat("31", 32), strings.Repeat("72", 16)
	path := filepath.Join(t.TempDir(), "native-消息.db")
	h, err := openEncrypted(path, key, salt, 2|4)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { api.close(h) })
	nativeExec(t, h, "PRAGMA journal_mode=WAL; PRAGMA wal_autocheckpoint=0; CREATE TABLE messages(id INTEGER PRIMARY KEY, body TEXT); INSERT INTO messages VALUES(1,'committed')")
	return path, key, salt, h
}
func nativeExec(t *testing.T, h uintptr, sql string) {
	t.Helper()
	if rc := api.exec(h, sql, 0, 0, 0); rc != 0 {
		t.Fatalf("synthetic SQL rc=%d: %s", rc, api.errmsg(h))
	}
}
func queryCount(t *testing.T, path string) int {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow("SELECT count(*) FROM messages").Scan(&count); err != nil {
		t.Fatal(err)
	}
	var check string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&check); err != nil || check != "ok" {
		t.Fatalf("integrity=%s err=%v", check, err)
	}
	return count
}
func digest(t *testing.T, path string) [32]byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(b)
}
func TestNativeBackupLiveWALAndSourcePreserved(t *testing.T) {
	path, key, salt, writer := nativeFixture(t)
	nativeExec(t, writer, "BEGIN; INSERT INTO messages VALUES(2,'uncommitted')")
	mainBefore, walBefore := digest(t, path), digest(t, path+"-wal")
	out := filepath.Join(t.TempDir(), "plain.db")
	if err := Backup(context.Background(), path, out, key, salt); err != nil {
		t.Fatal(err)
	}
	if count := queryCount(t, out); count != 1 {
		t.Fatalf("uncommitted row leaked: %d", count)
	}
	if digest(t, path) != mainBefore || digest(t, path+"-wal") != walBefore {
		t.Fatal("backup changed source DB or WAL")
	}
	nativeExec(t, writer, "COMMIT")
	if err := Backup(context.Background(), path, out, key, salt); err != nil {
		t.Fatal(err)
	}
	if count := queryCount(t, out); count != 2 {
		t.Fatalf("latest committed WAL row missing: %d", count)
	}
	// A separate pure-Go page/WAL reader must agree on the native fixture.
	k, _ := hex.DecodeString(key)
	independent := filepath.Join(t.TempDir(), "pure-go.db")
	report, err := wcdb.DecryptDBWithWAL(path, path+"-wal", independent, k)
	if err != nil {
		t.Fatal(err)
	}
	if report.WALFramesApplied == 0 {
		t.Fatalf("native WAL compatibility: report=%+v", report)
	}
	if count := queryCount(t, independent); count != 2 {
		t.Fatalf("native WAL compatibility: report=%+v count=%d", report, count)
	}
}
func TestNativeBackupPinsOneTransaction(t *testing.T) {
	path, key, salt, writer := nativeFixture(t)
	stage := filepath.Join(t.TempDir(), "encrypted.db")
	err := backupEncrypted(context.Background(), path, stage, key, salt, func() error {
		nativeExec(t, writer, "INSERT INTO messages VALUES(2,'later commit')")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	k, _ := hex.DecodeString(key)
	out := filepath.Join(t.TempDir(), "plain.db")
	if _, err := wcdb.DecryptDB(stage, out, k); err != nil {
		t.Fatal(err)
	}
	if count := queryCount(t, out); count != 1 {
		t.Fatalf("backup crossed source transaction: %d", count)
	}
}
func TestNativeBackupFailurePreservesDestination(t *testing.T) {
	for _, kind := range []string{"wrong-key", "later-page-corruption", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			path, key, salt, writer := nativeFixture(t)
			nativeExec(t, writer, "PRAGMA wal_checkpoint(TRUNCATE)")
			if kind == "later-page-corruption" {
				f, err := os.OpenFile(path, os.O_RDWR, 0)
				if err != nil {
					t.Fatal(err)
				}
				b := []byte{0}
				if _, err = f.ReadAt(b, 4096+120); err != nil {
					t.Fatal(err)
				}
				b[0] ^= 0x80
				if _, err = f.WriteAt(b, 4096+120); err != nil {
					t.Fatal(err)
				}
				f.Close()
			}
			if kind == "wrong-key" {
				key = strings.Repeat("00", 32)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if kind == "cancelled" {
				cancel()
			}
			out := filepath.Join(t.TempDir(), "out.db")
			if err := os.WriteFile(out, []byte("previous result"), 0o600); err != nil {
				t.Fatal(err)
			}
			err := Backup(ctx, path, out, key, salt)
			if err == nil {
				t.Fatal("unsafe snapshot accepted")
			}
			if kind == "cancelled" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			b, _ := os.ReadFile(out)
			if string(b) != "previous result" {
				t.Fatal("failure changed previous output")
			}
			files, _ := filepath.Glob(filepath.Join(filepath.Dir(out), ".cipher-snapshot-*"))
			if len(files) != 0 {
				t.Fatal("private intermediate left behind")
			}
		})
	}
}
func TestNativeBackupCancellationAfterPin(t *testing.T) {
	path, key, salt, _ := nativeFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := backupEncrypted(ctx, path, filepath.Join(t.TempDir(), "out.db"), key, salt, func() error { cancel(); return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel after pin: %v", err)
	}
}
