//go:build windows

package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"weixin-key/internal/wxkey"
)

func TestFrozenEncryptedSourceRefusesOpenWriter(t *testing.T) {
	acct, pp := buildFixtureAccount(t)
	path := filepath.Join(acct, "db_storage", "message", "msg_0.db")
	writer, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	_ = pp
	f, err := openFrozenSource(path)
	if f != nil {
		f.Close()
	}
	if err == nil {
		t.Fatal("unprotected main-then-WAL copy accepted while a writer held the encrypted source")
	}
}

func TestEncryptedSnapshotOnlineWithOpenWriter(t *testing.T) {
	acct, pp := buildFixtureAccount(t)
	path := filepath.Join(acct, "db_storage", "message", "msg_0.db")
	writer, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	plain, info, _, _, err := snapshotDB(sourceDB{rel: "message/msg_0.db", path: path}, t.TempDir(), wxkey.NewKeyResolver(), mustResolvePassphrase(t, pp))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(info.Method, "sqlite3mc-online-backup-read-transaction/") || !isPlaintextSQLite(plain) {
		t.Fatalf("wrong snapshot evidence: %+v", info)
	}
}
