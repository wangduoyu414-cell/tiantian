package export

import (
	"bufio"
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

// incrAccount is a mutable fixture account for incremental tests.
type incrAccount struct {
	root string
	pp   string
	salt []byte
}

// newIncrAccount builds an encrypted one-table account; msgRows are the
// current rows of Msg_a (values ready for INSERT).
func newIncrAccount(t *testing.T, msgRows []string) *incrAccount {
	t.Helper()
	root := t.TempDir()
	acct := filepath.Join(root, "wxid_incr_1")
	a := &incrAccount{root: acct}
	pass := make([]byte, 32)
	if _, err := rand.Read(pass); err != nil {
		t.Fatal(err)
	}
	a.pp = hexEncode(pass)
	a.salt = make([]byte, 16)
	if _, err := rand.Read(a.salt); err != nil {
		t.Fatal(err)
	}
	a.writeMsgDB(t, msgRows)
	return a
}

func (a *incrAccount) writeMsgDB(t *testing.T, msgRows []string) {
	t.Helper()
	enc, err := wcdb.DeriveEncKey(a.pp, hexEncode(a.salt))
	if err != nil {
		t.Fatal(err)
	}
	stmts := []string{
		`CREATE TABLE Msg_a (localId INTEGER PRIMARY KEY, serverId INTEGER, createTime INTEGER, localType INTEGER, messageContent TEXT, isSender INTEGER, talker TEXT)`,
		testutil.PadInsertSQL("Msg_a"),
	}
	stmts = append(stmts, msgRows...)
	stmts = append(stmts, testutil.PadDeleteSQL("Msg_a"))
	msgDir := filepath.Join(a.root, "db_storage", "message")
	if err := os.MkdirAll(msgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	encB, _ := hexDecode(enc)
	if err := testutil.BuildEncryptedSQLiteDB(filepath.Join(msgDir, "msg_0.db"), encB, a.salt, stmts...); err != nil {
		t.Fatal(err)
	}
}

func (a *incrAccount) options(out string) Options {
	return Options{DBRoot: a.root, OutDir: out, Resolve: wxkey.ResolveOptions{PassphraseHex: a.pp, NoConfig: true}}
}

// readJSONL returns the published messages keyed by message_id.
func readJSONL(t *testing.T, out string) map[string]Message {
	t.Helper()
	f, err := os.Open(filepath.Join(out, "data", "messages.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
	out_ := map[string]Message{}
	for sc.Scan() {
		var m Message
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatal(err)
		}
		out_[m.MessageID] = m
	}
	return out_
}

var baseRows = []string{
	`INSERT INTO Msg_a VALUES (1, 7001, 1715000000, 1, '第一条', 0, 'wxid_a')`,
	`INSERT INTO Msg_a VALUES (2, 7002, 1715000060, 1, '第二条', 1, '')`,
}

// Zero-change rerun: everything unchanged, nothing duplicated.
func TestIncrementalRerunUnchanged(t *testing.T) {
	a := newIncrAccount(t, baseRows)
	out := t.TempDir()
	rep1, err := Run(a.options(out))
	if err != nil {
		t.Fatal(err)
	}
	if rep1.Merge == nil || rep1.Merge.New != 2 {
		t.Fatalf("first run merge = %+v", rep1.Merge)
	}
	rep2, err := Run(a.options(out))
	if err != nil {
		t.Fatal(err)
	}
	if rep2.Merge.Unchanged != 2 || rep2.Merge.New != 0 || rep2.Merge.Revised != 0 || rep2.Merge.Carried != 0 {
		t.Fatalf("rerun merge = %+v, want 2 unchanged only", rep2.Merge)
	}
	msgs := readJSONL(t, out)
	if len(msgs) != 2 {
		t.Fatalf("jsonl has %d messages, want 2", len(msgs))
	}
}

// One appended message lands as exactly one new message.
func TestIncrementalAddsOnlyNew(t *testing.T) {
	a := newIncrAccount(t, baseRows)
	out := t.TempDir()
	if _, err := Run(a.options(out)); err != nil {
		t.Fatal(err)
	}
	a.writeMsgDB(t, append(append([]string{}, baseRows...),
		`INSERT INTO Msg_a VALUES (3, 7003, 1715000900, 1, '第三条来了', 0, 'wxid_a')`))
	rep, err := Run(a.options(out))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Merge.New != 1 || rep.Merge.Unchanged != 2 {
		t.Fatalf("merge = %+v, want +1 new, 2 unchanged", rep.Merge)
	}
	msgs := readJSONL(t, out)
	if len(msgs) != 3 {
		t.Fatalf("jsonl has %d, want 3", len(msgs))
	}
	if msgs["svr:7003"].Text != "第三条来了" {
		t.Fatal("new message content wrong")
	}
}

// Content change under a stable id is a revision, not a new message.
func TestIncrementalRevisionOnChange(t *testing.T) {
	a := newIncrAccount(t, baseRows)
	out := t.TempDir()
	if _, err := Run(a.options(out)); err != nil {
		t.Fatal(err)
	}
	a.writeMsgDB(t, []string{
		`INSERT INTO Msg_a VALUES (1, 7001, 1715000000, 1, '第一条(已编辑)', 0, 'wxid_a')`,
		`INSERT INTO Msg_a VALUES (2, 7002, 1715000060, 1, '第二条', 1, '')`,
	})
	rep, err := Run(a.options(out))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Merge.Revised != 1 || rep.Merge.Unchanged != 1 {
		t.Fatalf("merge = %+v, want 1 revised + 1 unchanged", rep.Merge)
	}
	msgs := readJSONL(t, out)
	m := msgs["svr:7001"]
	if m.Revision != 1 || m.Text != "第一条(已编辑)" {
		t.Fatalf("revision = %d text = %q", m.Revision, m.Text)
	}
	if msgs["svr:7002"].Revision != 0 {
		t.Fatal("unchanged message gained a revision")
	}
}

// A message whose shard disappears must be carried forward, not deleted.
func TestIncrementalCarriesForwardMissingMessages(t *testing.T) {
	a := newIncrAccount(t, baseRows)
	out := t.TempDir()
	if _, err := Run(a.options(out)); err != nil {
		t.Fatal(err)
	}
	// Source loses svr:7002 (e.g. shard rotated away): it must survive.
	a.writeMsgDB(t, baseRows[:1])
	rep, err := Run(a.options(out))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Merge.Carried != 1 {
		t.Fatalf("merge = %+v, want 1 carried", rep.Merge)
	}
	msgs := readJSONL(t, out)
	carried, ok := msgs["svr:7002"]
	if !ok {
		t.Fatal("missing message was dropped instead of carried forward")
	}
	hasMarker := false
	for _, w := range carried.Warnings {
		if w == "missing-in-source" {
			hasMarker = true
		}
	}
	if !hasMarker {
		t.Fatalf("carried message missing marker: %v", carried.Warnings)
	}
}

// Same identity arriving from two shards is kept once.
func TestIncrementalDedupAcrossShards(t *testing.T) {
	a := newIncrAccount(t, baseRows)
	// Second shard with a duplicate of svr:7001.
	enc, err := wcdb.DeriveEncKey(a.pp, hexEncode(a.salt))
	if err != nil {
		t.Fatal(err)
	}
	encB, _ := hexDecode(enc)
	if err := testutil.BuildEncryptedSQLiteDB(filepath.Join(a.root, "db_storage", "message", "msg_1.db"), encB, a.salt,
		`CREATE TABLE Msg_a (localId INTEGER PRIMARY KEY, serverId INTEGER, createTime INTEGER, localType INTEGER, messageContent TEXT, isSender INTEGER, talker TEXT)`,
		testutil.PadInsertSQL("Msg_a"),
		`INSERT INTO Msg_a VALUES (9, 7001, 1715000000, 1, '第一条', 0, 'wxid_a')`,
		testutil.PadDeleteSQL("Msg_a"),
	); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	rep, err := Run(a.options(out))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Merge.Dupes != 1 {
		t.Fatalf("dupes = %d, want 1", rep.Merge.Dupes)
	}
	if rep.TotalMessages != 2 {
		t.Fatalf("total = %d, want 2 after dedup", rep.TotalMessages)
	}
	msgs := readJSONL(t, out)
	if len(msgs) != 2 {
		t.Fatalf("jsonl has %d, want 2", len(msgs))
	}
}

// --full ignores previous exports entirely.
func TestFullExportIgnoresPrevious(t *testing.T) {
	a := newIncrAccount(t, baseRows)
	out := t.TempDir()
	if _, err := Run(a.options(out)); err != nil {
		t.Fatal(err)
	}
	// Shrink the source.
	a.writeMsgDB(t, baseRows[:1])
	opts := a.options(out)
	opts.Full = true
	rep, err := Run(opts)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Merge.New != 1 || rep.Merge.Carried != 0 {
		t.Fatalf("full merge = %+v, want all-new 1, no carried", rep.Merge)
	}
	msgs := readJSONL(t, out)
	if len(msgs) != 1 {
		t.Fatalf("full export kept %d messages, want 1", len(msgs))
	}
}

// A carried message with an EARLIER timestamp than a current message must
// appear before it in the conversation volume (volumes are re-rendered from
// the merged JSONL, not appended blindly).
func TestCarriedMessagesLandInTimeOrder(t *testing.T) {
	a := newIncrAccount(t, baseRows)
	out := t.TempDir()
	if _, err := Run(a.options(out)); err != nil {
		t.Fatal(err)
	}
	// Keep only the LATER message in the source; the earlier one is carried.
	a.writeMsgDB(t, baseRows[1:])
	rep, err := Run(a.options(out))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Merge == nil || rep.Merge.Carried != 1 {
		t.Fatalf("merge = %+v, want 1 carried", rep.Merge)
	}
	md, err := os.ReadFile(filepath.Join(out, "accounts", "wxid_incr", "conversations", "Msg_a", "2024-05.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(md)
	i1 := strings.Index(body, "第一条")
	i2 := strings.Index(body, "第二条")
	if i1 < 0 || i2 < 0 {
		t.Fatal("volume missing messages")
	}
	if i1 > i2 {
		t.Fatal("carried earlier message must precede the current later message in the volume")
	}
	// The carried message keeps its marker.
	if !strings.Contains(body, "missing-in-source") {
		t.Fatal("carried marker missing from volume")
	}
}
