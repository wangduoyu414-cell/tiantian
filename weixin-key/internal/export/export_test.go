package export

import (
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"weixin-key/internal/testutil"
	"weixin-key/internal/wcdb"
	"weixin-key/internal/wxkey"
)

// mustResolvePassphrase builds ResolveOptions with an explicit passphrase and
// neutralizes the env override so tests are hermetic.
func mustResolvePassphrase(t *testing.T, pp string) wxkey.ResolveOptions {
	t.Helper()
	t.Setenv("WECHAT_CLI_PASSPHRASE_HEX", "")
	return wxkey.ResolveOptions{PassphraseHex: pp, NoConfig: true}
}

// noMaterialOptions resolves nothing: no flags, no config.
func noMaterialOptions() wxkey.ResolveOptions {
	return wxkey.ResolveOptions{NoConfig: true}
}

// buildFixtureAccount creates a synthetic WeChat-like account directory:
//
//	root/wxid_test_1234/db_storage/
//	  message/msg_0.db   (encrypted; Msg_aaa + Msg_bbb + Session tables)
//	  contact/contact.db (encrypted; contact table)
//
// Returns the account root and passphrase hex.
func buildFixtureAccount(t *testing.T) (root, passphraseHex string) {
	t.Helper()
	pass := make([]byte, wcdb.PassphraseLength)
	if _, err := rand.Read(pass); err != nil {
		t.Fatal(err)
	}
	passphraseHex = hexEncode(pass)

	root = t.TempDir()
	acct := filepath.Join(root, "wxid_test_1234")
	storage := filepath.Join(acct, "db_storage")

	salt1 := make([]byte, wcdb.SaltLength)
	salt2 := make([]byte, wcdb.SaltLength)
	if _, err := rand.Read(salt1); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(salt2); err != nil {
		t.Fatal(err)
	}
	enc1, err := wcdb.DeriveEncKey(passphraseHex, hexEncode(salt1))
	if err != nil {
		t.Fatal(err)
	}
	enc2, err := wcdb.DeriveEncKey(passphraseHex, hexEncode(salt2))
	if err != nil {
		t.Fatal(err)
	}
	enc1b, _ := hexDecode(enc1)
	enc2b, _ := hexDecode(enc2)

	msgDir := filepath.Join(storage, "message")
	if err := os.MkdirAll(msgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// BuildEncryptedSQLiteDB proves the round-trip (decrypt -> row counts) at
	// fixture build time. Pad rows keep real content out of the reserve zone.
	if err := testutil.BuildEncryptedSQLiteDB(filepath.Join(msgDir, "msg_0.db"), enc1b, salt1,
		`CREATE TABLE Msg_aaa (localId INTEGER PRIMARY KEY, serverId INTEGER, createTime INTEGER, localType INTEGER, messageContent TEXT, isSender INTEGER, talker TEXT)`,
		testutil.PadInsertSQL("Msg_aaa"),
		`INSERT INTO Msg_aaa VALUES (1, 9001, 1715000000, 1, '你好，第一条', 0, 'wxid_friend')`,
		`INSERT INTO Msg_aaa VALUES (2, 9002, 1715000060, 1, '同文同秒不同消息', 0, 'wxid_friend')`,
		`INSERT INTO Msg_aaa VALUES (3, 9003, 1715003600, 3, '<msg><img src="x"/></msg>', 1, '')`,
		`INSERT INTO Msg_aaa VALUES (4, 9004, 1715007200, 98765, '未知类型内容体', 0, 'wxid_friend')`,
		testutil.PadDeleteSQL("Msg_aaa"),
		`CREATE TABLE Msg_bbb (localId INTEGER PRIMARY KEY, serverId INTEGER, createTime INTEGER, localType INTEGER, messageContent TEXT, isSender INTEGER, talker TEXT)`,
		testutil.PadInsertSQL("Msg_bbb"),
		`INSERT INTO Msg_bbb VALUES (1, 8001, 1715100000, 1, '另一个会话', 0, 'wxid_friend2')`,
		testutil.PadDeleteSQL("Msg_bbb"),
		`CREATE TABLE Session (username TEXT, tableName TEXT, nickname TEXT, type INTEGER)`,
		testutil.PadInsertValuesSQL("Session", "", `('__pad__', '__pad__', '`+testutil.PadBlob()+`', 0)`),
		`INSERT INTO Session VALUES ('wxid_friend', 'Msg_aaa', '朋友甲', 0)`,
		`INSERT INTO Session VALUES ('wxid_friend2', 'Msg_bbb', '群聊乙', 2)`,
		`DELETE FROM Session WHERE username = '__pad__'`,
	); err != nil {
		t.Fatal(err)
	}

	contactDir := filepath.Join(storage, "contact")
	if err := os.MkdirAll(contactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := testutil.BuildEncryptedSQLiteDB(filepath.Join(contactDir, "contact.db"), enc2b, salt2,
		`CREATE TABLE contact (username TEXT, nickname TEXT, remark TEXT)`,
		testutil.PadInsertValuesSQL("contact", "", `('__pad__', '`+testutil.PadBlob()+`', '__pad__')`),
		`INSERT INTO contact VALUES ('wxid_friend', '朋友昵称', '备注朋友')`,
		`DELETE FROM contact WHERE username = '__pad__'`,
	); err != nil {
		t.Fatal(err)
	}
	return acct, passphraseHex
}

func hexEncode(b []byte) string {
	const hexdig = "0123456789abcdef"
	var sb strings.Builder
	for _, c := range b {
		sb.WriteByte(hexdig[c>>4])
		sb.WriteByte(hexdig[c&0xf])
	}
	return sb.String()
}

func hexDecode(s string) ([]byte, error) {
	out := make([]byte, len(s)/2)
	for i := 0; i < len(out); i++ {
		hi := unhexByte(s[2*i])
		lo := unhexByte(s[2*i+1])
		out[i] = hi<<4 | lo
	}
	return out, nil
}

func unhexByte(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

// Full pipeline: encrypted fixture account -> export -> Markdown + JSONL +
// manifest, with per-salt derivation, session/contact enrichment, unknown
// type preservation and deterministic reruns.
func TestRunExportEndToEnd(t *testing.T) {
	acct, pp := buildFixtureAccount(t)
	out := t.TempDir()

	rep, err := Run(Options{
		DBRoot:  acct,
		OutDir:  out,
		Resolve: mustResolvePassphrase(t, pp),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Status != "partial" {
		b, _ := json.MarshalIndent(rep, "", "  ")
		t.Fatalf("status = %s, want partial because the fixture includes an unknown type\n%s", rep.Status, b)
	}
	if rep.TotalMessages != 5 {
		t.Fatalf("total messages = %d, want 5", rep.TotalMessages)
	}
	if rep.TotalConversations != 2 {
		t.Fatalf("conversations = %d, want 2", rep.TotalConversations)
	}

	// JSONL carries all 5 messages with provenance and types.
	data, err := os.ReadFile(filepath.Join(out, "data", "messages.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 5 {
		t.Fatalf("jsonl lines = %d, want 5", len(lines))
	}
	var msgs []Message
	for _, ln := range lines {
		var m Message
		if err := json.Unmarshal([]byte(ln), &m); err != nil {
			t.Fatalf("bad jsonl line: %v", err)
		}
		if m.SchemaVersion != "chat-v1" || m.AccountID != "wxid_test" {
			t.Fatalf("message identity wrong: %+v", m)
		}
		msgs = append(msgs, m)
	}

	// Distinct ids must survive even with identical text (no content dedup).
	found := map[string]int{}
	for _, m := range msgs {
		found[m.MessageID]++
	}
	if len(found) != 5 {
		t.Fatalf("expected 5 distinct message ids, got %d", len(found))
	}

	// Unknown type 98765 preserved with source_type and partial status.
	var unknown *Message
	for i := range msgs {
		if msgs[i].SourceType == 98765 {
			unknown = &msgs[i]
		}
	}
	if unknown == nil || unknown.Type != "unknown" || unknown.ParseStatus != ParsePartial {
		t.Fatalf("unknown type not preserved: %+v", unknown)
	}
	if unknown.StructuredContent == nil {
		t.Fatal("unknown type lost its raw content")
	}

	// Markdown volume exists with conversation display name from Session and
	// sender display name from contacts.
	vol := filepath.Join(out, "accounts", "wxid_test", "conversations", "Msg_aaa", "2024-05.md")
	md, err := os.ReadFile(vol)
	if err != nil {
		t.Fatalf("volume missing: %v", err)
	}
	body := string(md)
	if !strings.Contains(body, "朋友甲") {
		t.Fatal("volume header missing session nickname")
	}
	if !strings.Contains(body, "你好，第一条") {
		t.Fatal("volume missing message text")
	}
	// remark (user-set note) outranks nickname, matching WeChat UI.
	if !strings.Contains(body, "备注朋友") {
		t.Fatal("volume missing contact display name for sender")
	}
	if !strings.Contains(body, "meta: id=svr:9001") {
		t.Fatal("volume missing per-message provenance meta")
	}
	if strings.Contains(body, "另一个会话") {
		t.Fatal("cross-conversation bleed")
	}

	// manifest.json exists and is parseable, with per-DB evidence.
	manifest, err := os.ReadFile(filepath.Join(out, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var man Report
	if err := json.Unmarshal(manifest, &man); err != nil {
		t.Fatal(err)
	}
	if len(man.DBs) != 2 {
		t.Fatalf("manifest dbs = %d, want 2", len(man.DBs))
	}
	var msgDB *DBResult
	for i := range man.DBs {
		if strings.Contains(man.DBs[i].Path, "msg_0") {
			msgDB = &man.DBs[i]
		}
	}
	if msgDB == nil || msgDB.Snapshot == nil || msgDB.KeySource != "--passphrase" {
		t.Fatalf("msg db evidence missing: %+v", msgDB)
	}

	if _, err := os.Stat(filepath.Join(out, "index.md")); err != nil {
		t.Fatal("index.md missing")
	}
	if _, err := os.Stat(filepath.Join(out, ".export-state", "state.json")); err != nil {
		t.Fatal("publish state missing")
	}

	// Rerun: identical JSONL (deterministic, zero duplicates).
	rep2, err := Run(Options{DBRoot: acct, OutDir: out, Resolve: mustResolvePassphrase(t, pp)})
	if err != nil {
		t.Fatalf("rerun: %v", err)
	}
	data2, _ := os.ReadFile(filepath.Join(out, "data", "messages.jsonl"))
	if string(data2) != string(data) {
		t.Fatal("rerun produced different messages.jsonl")
	}
	if rep2.TotalMessages != 5 {
		t.Fatalf("rerun messages = %d, want 5 (no duplicates)", rep2.TotalMessages)
	}
}

// A DB whose material cannot be resolved must surface as partial, not hidden.
func TestRunExportPartialWhenKeyMissing(t *testing.T) {
	t.Setenv("WECHAT_CLI_PASSPHRASE_HEX", "")
	acct, _ := buildFixtureAccount(t)
	out := t.TempDir()

	rep, err := Run(Options{DBRoot: acct, OutDir: out, Resolve: noMaterialOptions()})
	if err != nil {
		t.Fatalf("Run returned hard error: %v", err)
	}
	if rep.Status != "failed" && rep.Status != "partial" {
		t.Fatalf("status = %s, want partial/failed", rep.Status)
	}
	if rep.TotalMessages != 0 {
		t.Fatalf("messages = %d, want 0", rep.TotalMessages)
	}
	for _, d := range rep.DBs {
		if d.Error == "" {
			t.Fatalf("db %s has no error recorded despite missing key", d.Path)
		}
	}
}

// Refusing to write into the source tree.
func TestRunExportRejectsOutputInsideSource(t *testing.T) {
	acct, pp := buildFixtureAccount(t)
	inside := filepath.Join(acct, "export-out")
	_, err := Run(Options{DBRoot: acct, OutDir: inside, Resolve: mustResolvePassphrase(t, pp)})
	if err == nil {
		t.Fatal("export into source tree was not rejected")
	}
}

// A pre-existing user file at a planned path is preserved; ours lands in a
// conflict sibling.
func TestPublisherPreservesUserFiles(t *testing.T) {
	out := t.TempDir()
	userFile := filepath.Join(out, "data", "messages.jsonl")
	if err := os.MkdirAll(filepath.Dir(userFile), 0o755); err != nil {
		t.Fatal(err)
	}
	userContent := []byte("my own notes, not generated\n")
	if err := os.WriteFile(userFile, userContent, 0o600); err != nil {
		t.Fatal(err)
	}

	acct, pp := buildFixtureAccount(t)
	rep, err := Run(Options{DBRoot: acct, OutDir: out, Resolve: mustResolvePassphrase(t, pp)})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got, err := os.ReadFile(userFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(userContent) {
		t.Fatal("user file was overwritten")
	}
	if len(rep.Conflicts) == 0 {
		t.Fatal("expected a recorded conflict for the user file")
	}
	matches, _ := filepath.Glob(filepath.Join(out, "data", "messages.jsonl.conflict-*"))
	if len(matches) == 0 {
		t.Fatal("conflict copy missing")
	}
}
