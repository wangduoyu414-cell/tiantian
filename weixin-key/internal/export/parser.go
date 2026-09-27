package export

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (pinned in go.mod)
)

// Column-name candidate lists. WeChat versions rename columns; we probe by
// evidence instead of hardcoding one generation's names.
var (
	idCandidates          = []string{"localid", "local_id", "id", "msgid", "msg_id"}
	serverCandidates      = []string{"serverid", "server_id", "svrid", "svr_id", "msgserverid", "msgserver_id"}
	timeCandidates        = []string{"createtime", "create_time", "timestamp", "msgtime", "msg_time"}
	typeCandidates        = []string{"localtype", "local_type", "type", "msgtype", "msg_type"}
	contentCandidates     = []string{"messagecontent", "message_content", "content", "msg_content", "msgcontent"}
	senderCandidates      = []string{"issender", "is_sender", "issend", "is_send", "direction"}
	talkerCandidates      = []string{"talker", "username", "fromusername", "from_username", "conversationid"}
	realSenderCandidates  = []string{"realsenderid", "real_sender_id", "senderid", "sender_id"}
	compressCandidates    = []string{"compresscontent", "compress_content"}
	packedCandidates      = []string{"packedinfodata", "packed_info_data"}
	contentTypeCandidates = []string{"wcdb_ct_message_content"}
	sourceTypeCandidates  = []string{"wcdb_ct_source"}
)

// tableInfo is one probed message table with its resolved column mapping.
type tableInfo struct {
	name           string
	cols           map[string]string // lower-case column -> declared type
	colID          string
	colServer      string
	colTime        string
	colType        string
	colContent     string
	colSender      string
	colTalker      string
	colRealSender  string
	colCompress    string
	colPacked      string
	colContentType string
	colSourceType  string
}

func pickCol(cols map[string]string, candidates []string) string {
	for _, c := range candidates {
		if _, ok := cols[c]; ok {
			return c
		}
	}
	return ""
}

// openSnapshot opens a plaintext SQLite snapshot read-only.
func openSnapshot(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=query_only(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	return db, nil
}

// listTables returns all user tables in the database.
func listTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// tableColumns probes PRAGMA table_info.
func tableColumns(db *sql.DB, table string) (map[string]string, error) {
	// PRAGMA does not support bind parameters for the table name; quote it.
	rows, err := db.Query(`PRAGMA table_info(` + quoteIdent(table) + `)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols := map[string]string{}
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		cols[strings.ToLower(name)] = ctype
	}
	return cols, rows.Err()
}

// quoteIdent quotes a SQLite identifier.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// probeMessageTables finds tables that look like message tables: either the
// WeChat Msg_* naming pattern, or a table carrying id+time+content evidence.
func probeMessageTables(db *sql.DB) ([]tableInfo, []SkippedTable, error) {
	tables, err := listTables(db)
	if err != nil {
		return nil, nil, err
	}
	var infos []tableInfo
	var skipped []SkippedTable
	for _, t := range tables {
		cols, err := tableColumns(db, t)
		if err != nil {
			skipped = append(skipped, SkippedTable{Table: t, Reason: "columns unreadable: " + err.Error()})
			continue
		}
		ti := tableInfo{
			name:           t,
			cols:           cols,
			colID:          pickCol(cols, idCandidates),
			colServer:      pickCol(cols, serverCandidates),
			colTime:        pickCol(cols, timeCandidates),
			colType:        pickCol(cols, typeCandidates),
			colContent:     pickCol(cols, contentCandidates),
			colSender:      pickCol(cols, senderCandidates),
			colTalker:      pickCol(cols, talkerCandidates),
			colRealSender:  pickCol(cols, realSenderCandidates),
			colCompress:    pickCol(cols, compressCandidates),
			colPacked:      pickCol(cols, packedCandidates),
			colContentType: pickCol(cols, contentTypeCandidates),
			colSourceType:  pickCol(cols, sourceTypeCandidates),
		}
		nameLooksMsg := strings.HasPrefix(strings.ToLower(t), "msg_") || strings.EqualFold(t, "message") || strings.EqualFold(t, "messages")
		schemaLooksMsg := ti.colID != "" && ti.colTime != "" && ti.colContent != ""
		if nameLooksMsg && ti.colTime == "" && ti.colContent == "" {
			skipped = append(skipped, SkippedTable{Table: t, Reason: "message-like name but no recognized time/content columns"})
			continue
		}
		if !nameLooksMsg && !schemaLooksMsg {
			skipped = append(skipped, SkippedTable{Table: t, Reason: "not a message table (no Msg_* name and no id+time+content columns)"})
			continue
		}
		if ti.colID == "" {
			// rowid fallback keeps the message addressable.
			ti.colID = "rowid"
		}
		infos = append(infos, ti)
	}
	return infos, skipped, nil
}

// normalized type mapping for the WeChat local types we have evidence for.
// Unknown codes are preserved as source_type with type "unknown".
func normalizeType(localType int64) string {
	// Newer message rows pack status/transport bits above the legacy local
	// type. Keep the raw value in SourceType, but normalize from the low
	// 32-bit type field so values such as 0x500000031 still classify as 49
	// (app/file/link) instead of becoming a false unknown.
	baseType := localType & 0xffffffff
	switch baseType {
	case 1:
		return "text"
	case 3:
		return "image"
	case 34:
		return "voice"
	case 43:
		return "video"
	case 47:
		return "emoji"
	case 49:
		return "app" // file/link/mini-program share
	case 10000:
		return "system"
	default:
		return "unknown"
	}
}

// normalizeTime interprets a raw WeChat timestamp, which is unix seconds in
// the versions we have evidence for; absurdly large values are treated as
// milliseconds. The raw value and detected unit are always preserved.
func normalizeTime(raw int64) (ts time.Time, unit string) {
	switch {
	case raw <= 0:
		return time.Time{}, "unknown"
	case raw > 1e12: // beyond ~ year 3366 in seconds: must be ms
		return time.UnixMilli(raw), "ms"
	default:
		return time.Unix(raw, 0), "s"
	}
}

// selectSQL builds a deterministic streaming SELECT for a message table.
// Rows come back ordered by (time, id) so output is stable across runs.
// Every column is aliased to its lowercase role name, because SQLite reports
// declared-name casing (and rowid can report the INTEGER PRIMARY KEY's name).
func (ti tableInfo) selectSQL() string {
	cols := []string{"rowid AS __rowid"}
	seen := map[string]bool{"__rowid": true}
	for _, c := range []string{
		ti.colID, ti.colServer, ti.colTime, ti.colType, ti.colContent,
		ti.colSender, ti.colTalker, ti.colRealSender, ti.colCompress,
		ti.colPacked, ti.colContentType, ti.colSourceType,
	} {
		if c == "" || c == "rowid" || seen[c] {
			continue
		}
		seen[c] = true
		cols = append(cols, quoteIdent(c)+" AS "+quoteIdent(c))
	}
	var order []string
	if ti.colTime != "" {
		order = append(order, quoteIdent(ti.colTime))
	}
	order = append(order, "rowid")
	return fmt.Sprintf("SELECT %s FROM %s ORDER BY %s", strings.Join(cols, ", "), quoteIdent(ti.name), strings.Join(order, ", "))
}
