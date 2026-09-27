package testutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"database/sql"
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/pbkdf2"
	_ "modernc.org/sqlite" // pure-Go SQLite engine for fixture creation

	"weixin-key/internal/wcdb"
)

// BuildSQLiteDB creates a REAL plaintext SQLite database at path and runs the
// given DDL/DML statements in order.
func BuildSQLiteDB(path string, statements ...string) error {
	_ = os.Remove(path)
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec("PRAGMA journal_mode=DELETE"); err != nil {
		return fmt.Errorf("journal_mode: %w", err)
	}
	for i, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("statement %d: %w", i, err)
		}
	}
	return db.Close()
}

// SQLite packs b-tree cells from the END of each page downward, and this
// pure-Go fixture path cannot set the SQLCipher reserve (80 bytes) that real
// WeChat DBs are written with. Without a reserve, the first records of every
// page would land in the [4016:4096] region that physical SQLCipher
// encryption replaces with IV+HMAC, silently truncating them.
//
// The fix mirrors what the reserve does: a sacrificial pad record is written
// FIRST so it occupies the page tail, real records land above 4016, and the
// pad is then deleted (leaving an unreferenced freeblock that may be
// truncated harmlessly).
const schemaPadCreate = `CREATE TABLE __schema_pad (c1 TEXT, c2 TEXT, c3 TEXT, c4 TEXT, c5 TEXT, c6 TEXT)`
const schemaPadDrop = `DROP TABLE __schema_pad`

// PadInsertSQL returns an INSERT that parks a sacrificial pad row at the
// message-table page tail; PadDeleteSQL removes it after real rows are in
// place. For other table shapes use PadInsertValuesSQL.
func PadInsertSQL(table string) string {
	return PadInsertValuesSQL(table,
		"(localId, serverId, createTime, localType, messageContent, isSender, talker)",
		"(-1, -1, 0, 1, '"+strings.Repeat("P", 200)+"', 0, '')")
}

// PadDeleteSQL deletes the sacrificial pad row (message tables).
func PadDeleteSQL(table string) string {
	return fmt.Sprintf(`DELETE FROM %s WHERE localId = -1`, table)
}

// PadInsertValuesSQL is the shape-agnostic pad insert: padWhere must match
// only the pad row, and the values must include a >=200-byte string so the
// record covers the vulnerable page tail.
func PadInsertValuesSQL(table, columns, values string) string {
	return fmt.Sprintf(`INSERT INTO %s %s VALUES %s`, table, columns, values)
}

// PadBlob returns a 200-char string suitable for pad payloads.
func PadBlob() string { return strings.Repeat("P", 200) }

// BuildEncryptedSQLiteDB creates a real SQLite DB whose referenced content
// stays inside the SQLCipher usable page area, encrypts it into the
// SQLCipher 4 / WeChat 4.1+ layout, then PROVES the round-trip: it decrypts
// the result and compares every table's row count against the plaintext
// original. A layout violation fails here, at fixture build time, instead of
// masquerading as an export bug.
func BuildEncryptedSQLiteDB(encPath string, encKey, salt []byte, statements ...string) error {
	plain := encPath + ".plain-tmp"
	defer os.Remove(plain)

	stmts := make([]string, 0, len(statements)+2)
	stmts = append(stmts, schemaPadCreate)
	stmts = append(stmts, statements...)
	stmts = append(stmts, schemaPadDrop)
	if err := BuildSQLiteDB(plain, stmts...); err != nil {
		return err
	}

	before, err := tableRowCounts(plain)
	if err != nil {
		return fmt.Errorf("fixture plaintext unreadable: %w", err)
	}
	if err := EncryptSQLiteDB(plain, encPath, encKey, salt); err != nil {
		return err
	}

	// Round-trip: decrypt and compare all table row counts.
	roundTrip := encPath + ".roundtrip-tmp"
	defer os.Remove(roundTrip)
	if _, err := wcdb.DecryptDB(encPath, roundTrip, encKey); err != nil {
		return fmt.Errorf("fixture round-trip decrypt: %w", err)
	}
	after, err := tableRowCounts(roundTrip)
	if err != nil {
		return fmt.Errorf("fixture round-trip unreadable: %w", err)
	}
	if len(before) != len(after) {
		return fmt.Errorf("fixture table count changed: %d -> %d", len(before), len(after))
	}
	for tbl, n := range before {
		if after[tbl] != n {
			return fmt.Errorf("fixture table %s row count changed: %d -> %d (content landed in the reserve region)", tbl, n, after[tbl])
		}
	}
	return nil
}

func tableRowCounts(dbPath string) (map[string]int, error) {
	db, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=query_only(1)")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	rows.Close()
	out := map[string]int{}
	for _, tbl := range tables {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM "` + strings.ReplaceAll(tbl, `"`, `""`) + `"`).Scan(&n); err != nil {
			return nil, fmt.Errorf("count %s: %w", tbl, err)
		}
		out[tbl] = n
	}
	return out, nil
}

// EncryptSQLiteDB physically converts a plaintext SQLite DB into the
// SQLCipher 4 / WeChat 4.1+ layout: page 1 carries the salt instead of the
// SQLite header, each page is AES-256-CBC encrypted with a per-page IV in the
// reserve, and each page gets an HMAC-SHA512. The plaintext tail region
// [4016:4096] of every page is replaced - callers must keep referenced
// content out of it (see BuildEncryptedSQLiteDB).
func EncryptSQLiteDB(plainPath, encPath string, encKey, salt []byte) error {
	plain, err := os.ReadFile(plainPath)
	if err != nil {
		return err
	}
	if len(plain) == 0 || len(plain)%wcdb.SQLCipherPageSize != 0 {
		return fmt.Errorf("plaintext size %d is not a multiple of %d", len(plain), wcdb.SQLCipherPageSize)
	}
	block, err := aes.NewCipher(encKey)
	if err != nil {
		return err
	}
	hmacSalt := make([]byte, wcdb.SaltLength)
	for i := range salt {
		hmacSalt[i] = salt[i] ^ wcdb.SQLCipherSaltMask
	}
	macKey := pbkdf2.Key(encKey, hmacSalt, wcdb.SQLCipherFastKDF, wcdb.SQLCipherMACKeyLen, sha512.New)
	ivOff := wcdb.SQLCipherPageSize - wcdb.SQLCipherReserve

	pageCount := len(plain) / wcdb.SQLCipherPageSize
	out := make([]byte, 0, len(plain))
	for pg := 0; pg < pageCount; pg++ {
		pgno := pg + 1
		src := plain[pg*wcdb.SQLCipherPageSize : (pg+1)*wcdb.SQLCipherPageSize]
		page := make([]byte, wcdb.SQLCipherPageSize)
		iv := make([]byte, wcdb.SQLCipherIVSize)
		if _, err := rand.Read(iv); err != nil {
			return err
		}
		start := 0
		if pgno == 1 {
			copy(page, salt)
			start = wcdb.SaltLength
		}
		cipher.NewCBCEncrypter(block, iv).CryptBlocks(page[start:ivOff], src[start:ivOff])
		copy(page[ivOff:ivOff+wcdb.SQLCipherIVSize], iv)

		var pgn [4]byte
		binary.LittleEndian.PutUint32(pgn[:], uint32(pgno))
		mac := hmac.New(sha512.New, macKey)
		mac.Write(page[start : wcdb.SQLCipherPageSize-wcdb.SQLCipherHMACSize])
		mac.Write(pgn[:])
		copy(page[wcdb.SQLCipherPageSize-wcdb.SQLCipherHMACSize:], mac.Sum(nil))
		out = append(out, page...)
	}
	return os.WriteFile(encPath, out, 0o600)
}
