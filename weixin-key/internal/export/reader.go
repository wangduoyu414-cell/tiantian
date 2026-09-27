package export

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/klauspost/compress/zstd"
)

// readTableMessages streams one message table in deterministic order and
// emits chat-v1 messages. Nothing is accumulated: the callback consumes each
// message as it is read.
func readTableMessages(db *sql.DB, ti tableInfo, ctx *parseContext, emit func(*Message) error) (int64, error) {
	queryCtx := ctx.context
	if queryCtx == nil {
		queryCtx = context.Background()
	}
	rows, err := db.QueryContext(queryCtx, ti.selectSQL())
	if err != nil {
		return 0, fmt.Errorf("query %s: %w", ti.name, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return 0, err
	}
	var count int64
	for rows.Next() {
		if err := queryCtx.Err(); err != nil {
			return count, err
		}
		raw := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return count, fmt.Errorf("scan %s: %w", ti.name, err)
		}
		msg := mapRowToMessage(ti, cols, raw, ctx)
		if err := emit(msg); err != nil {
			return count, err
		}
		count++
	}
	return count, rows.Err()
}

// parseContext carries account/snapshot identity for provenance.
type parseContext struct {
	context      context.Context
	accountID    string
	dbRel        string // db path relative to db_storage
	shard        string // file name
	snapshotID   string
	contacts     map[string]string // sender id -> display name
	convNames    map[string]string // conversation id (table name) -> display name
	convKinds    map[string]string
	accountName  string
	ownSenderIDs map[string]bool
	// name2id maps db path (relative to db_storage) to that DB's Name2Id
	// catalog (numeric row-id -> username). The catalog is per-database:
	// rowids are not meaningful across DBs.
	name2id map[string]map[string]string
	// allMsgTables accumulates every probed message table name across DBs
	// (lowercased -> actual case), so a session catalog in one DB can name
	// tables in another.
	allMsgTables map[string]string
	// convUsers maps message table name -> conversation username.
	convUsers map[string]string
	// attach resolves media attachments into the staging files dir; nil in
	// contexts without an output pipeline (tests, diagnostics).
	attach *attacher
}

// ownUsername is the account username with any WeChat-internal instance
// suffix ("_ee32" style) stripped, matching Name2Id user_name form.
func (ctx *parseContext) ownUsername() string {
	id := ctx.accountID
	if idx := strings.LastIndex(id, "_"); idx > 0 {
		suffix := id[idx+1:]
		if len(suffix) == 4 {
			hex := true
			for _, r := range suffix {
				if !('0' <= r && r <= '9' || 'a' <= r && r <= 'f' || 'A' <= r && r <= 'F') {
					hex = false
					break
				}
			}
			if hex {
				return id[:idx]
			}
		}
	}
	return id
}

func mapRowToMessage(ti tableInfo, cols []string, raw []any, ctx *parseContext) *Message {
	m := &Message{
		SchemaVersion:    SchemaVersion,
		AccountID:        ctx.accountID,
		ConversationID:   ti.name,
		ConversationKind: "unknown",
		Direction:        DirectionUnknown,
		Type:             "unknown",
		SourceTimeUnit:   "unknown",
		ParseStatus:      ParseOK,
		Revision:         0,
		Provenance: Provenance{
			DB:    ctx.dbRel,
			Table: ti.name,
			Shard: ctx.shard,
		},
	}
	if n := ctx.convNames[ti.name]; n != "" {
		m.ConversationName = n
	}
	if k := ctx.convKinds[ti.name]; k != "" {
		m.ConversationKind = k
	}

	var localID, serverID int64
	var haveID bool
	var content any
	var contentType, compressFlag, sourceStorageType int64
	var haveContentType, haveCompressFlag, haveSourceStorageType bool
	var realSender any
	extra := map[string]any{}

	for i, col := range cols {
		val := raw[i]
		switch strings.ToLower(col) {
		case "__rowid":
			m.Provenance.RowID = toInt64(val)
		case ti.colID:
			localID = toInt64(val)
			haveID = true
		case ti.colServer:
			serverID = toInt64(val)
		case ti.colTime:
			m.SourceTimestamp = toInt64(val)
		case ti.colType:
			m.SourceType = toInt64(val)
		case ti.colContent:
			content = val
		case ti.colSender:
			m.Direction = normalizeDirection(val)
		case ti.colTalker:
			if realSender == nil || semanticSenderEmpty(realSender) {
				m.SenderID = toText(val)
			}
		case ti.colRealSender:
			realSender = val
			if s, ok := safeText(val); ok {
				m.SenderID = s
			}
		case ti.colCompress:
			compressFlag = toInt64(val)
			haveCompressFlag = true
		case ti.colPacked:
			if b, ok := rawBytes(val); ok && len(b) > 0 {
				putRawBytes(m, "packed_info_data", b)
				setPartial(m, "packed_info_data retained as raw bytes")
			}
		case ti.colContentType:
			contentType = toInt64(val)
			haveContentType = true
			putNumber(m, "WCDB_CT_message_content", contentType)
		case ti.colSourceType:
			sourceStorageType = toInt64(val)
			haveSourceStorageType = true
			putNumber(m, "WCDB_CT_source", sourceStorageType)
		default:
			if s, ok := safeText(val); ok && s != "" {
				extra[col] = s
			} else if b, ok := rawBytes(val); ok && len(b) > 0 {
				extra[col] = map[string]any{
					"encoding": "base64",
					"value":    base64.StdEncoding.EncodeToString(b),
				}
			}
		}
	}
	_ = haveSourceStorageType // retained in StructuredContent for auditability

	// WCDB_CT_* stores both the compression category and the original SQLite
	// storage kind. A non-zero compression category means the bytes are not
	// ordinary UTF-8 text. WeChat 4.1 stores such payloads as dictionary-less
	// zstd frames (magic 28 B5 2F FD, no dict-ID); decode is attempted with a
	// hard output bound, and undecodable bytes are still retained losslessly.
	compressed := haveCompressFlag && compressFlag != 0
	if haveContentType && contentType/2 != 0 {
		compressed = true
	}
	if content != nil && compressed {
		if b, ok := rawBytes(content); ok && len(b) > 0 {
			if dec, ok := tryZstdDecode(b); ok {
				content = dec
				compressed = false
			}
		}
	}
	if content != nil {
		if compressed {
			if b, ok := rawBytes(content); ok && len(b) > 0 {
				putRawBytes(m, "raw_content", b)
			}
			setPartial(m, "compressed message_content retained; zstd decode failed")
		} else if s, ok := safeText(content); ok {
			m.Text = s
		} else if b, ok := rawBytes(content); ok && len(b) > 0 {
			putRawBytes(m, "raw_content", b)
			setPartial(m, "non-UTF-8 message_content retained as base64")
		}
	}

	// Stable message identity: server id first, then local id, then rowid.
	switch {
	case serverID > 0:
		m.MessageID = fmt.Sprintf("svr:%d", serverID)
		m.IDSource = "server_id"
	case haveID && localID > 0:
		m.MessageID = fmt.Sprintf("loc:%d", localID)
		m.IDSource = "local_id"
	default:
		m.MessageID = fmt.Sprintf("row:%d", m.Provenance.RowID)
		m.IDSource = "rowid_fallback"
		m.Warnings = append(m.Warnings, "no stable source id; rowid fallback is shard-local")
	}

	ts, unit := normalizeTime(m.SourceTimestamp)
	m.SourceTimeUnit = unit
	if !ts.IsZero() {
		m.Timestamp = ts.Local().Format(time.RFC3339)
	} else {
		m.Warnings = append(m.Warnings, "missing or unusable source timestamp")
		m.ParseStatus = ParsePartial
	}

	m.Type = normalizeType(m.SourceType)
	if m.Type == "unknown" {
		m.ParseStatus = ParsePartial
		m.Warnings = append(m.Warnings, fmt.Sprintf("unrecognized source type %d", m.SourceType))
	}
	if m.Type != "text" && m.Text != "" {
		// Rich/quoted payloads (XML for image/emoji/app/video) are kept raw in
		// JSONL for audit; the Markdown body gets a bounded human summary.
		if m.StructuredContent == nil {
			m.StructuredContent = map[string]any{}
		}
		raw := m.Text
		if strings.HasPrefix(strings.TrimSpace(raw), "<") {
			m.StructuredContent["content_xml"] = raw
		} else {
			m.StructuredContent["raw_content"] = raw
		}
		m.Text = summarizeRich(m.Type, raw)
		if m.ParseStatus == ParseOK {
			m.ParseStatus = ParsePartial
		}
	}
	if realSender != nil {
		if s, ok := safeText(realSender); ok && s != "" && !isNumericValue(realSender) {
			if strings.EqualFold(s, ctx.ownUsername()) {
				m.Direction = DirectionOut
			} else {
				m.Direction = DirectionIn
			}
		} else if isNumericValue(realSender) {
			// In the current 17-column schema this is an internal numeric
			// identifier: a rowid into this DB's own Name2Id catalog.
			// Resolve it there first; the catalog is per-database, so a
			// miss must not fall back to any other DB's numbering.
			senderKey := toText(realSender)
			uname := ""
			catalogHit := false
			if catalog := ctx.name2id[ctx.dbRel]; catalog != nil {
				uname, catalogHit = catalog[senderKey]
			}
			switch {
			case uname != "":
				m.SenderID = uname
				if strings.EqualFold(uname, ctx.ownUsername()) {
					m.Direction = DirectionOut
				} else {
					m.Direction = DirectionIn
				}
			case catalogHit:
				// Name2Id knows this id but maps it to an empty username
				// (WeChat's system/placeholder entry, observed as rowid 35).
				// Not a resolvable person; leave direction to type fallback
				// and drop the bare numeric id from display.
				m.SenderID = ""
			case ctx.ownSenderIDs[senderKey]:
				m.Direction = DirectionOut
			case ctx.contacts[senderKey] != "":
				m.Direction = DirectionIn
			default:
				m.Direction = DirectionUnknown
				setPartial(m, "numeric real_sender_id retained; sender directory mapping unresolved")
			}
		}
	}
	// System-type messages with no resolved sender are system events, not
	// "unknown direction" evidence.
	if m.Direction == DirectionUnknown && m.Type == "system" {
		m.Direction = DirectionSystem
	}
	// Media-type messages resolve real attachments when an attacher is wired;
	// otherwise they carry the honest not-extracted placeholder.
	switch m.Type {
	case "image", "voice", "video", "app", "emoji":
		if ctx.attach != nil {
			ctx.attach.resolve(m, ti.name, localID)
		}
		if len(m.Attachments) == 0 {
			m.Attachments = []AttachmentState{{Kind: m.Type, Status: attNotExtracted}}
		}
	}
	if len(extra) > 0 {
		if m.StructuredContent == nil {
			m.StructuredContent = map[string]any{}
		}
		m.StructuredContent["unparsed_columns"] = extra
	}

	// Sender display name via contacts map (best effort).
	if m.SenderID != "" {
		if n, ok := ctx.contacts[m.SenderID]; ok {
			m.SenderName = n
		}
	} else if m.Direction == DirectionOut {
		m.SenderID = ctx.accountID
		m.SenderName = ctx.accountName
	}

	return m
}

// zstdDecoder is shared for DecodeAll, which is safe for concurrent use.
// The 32 MiB window cap bounds memory on hostile or corrupt frames.
var zstdDecoder = func() *zstd.Decoder {
	d, err := zstd.NewReader(nil, zstd.WithDecoderMaxWindow(32<<20))
	if err != nil {
		return nil
	}
	return d
}()

// tryZstdDecode decodes a dictionary-less zstd frame. It returns false unless
// the input carries the zstd magic and decodes within the bound.
func tryZstdDecode(b []byte) ([]byte, bool) {
	if len(b) < 4 || b[0] != 0x28 || b[1] != 0xB5 || b[2] != 0x2F || b[3] != 0xFD {
		return nil, false
	}
	if zstdDecoder == nil {
		return nil, false
	}
	out, err := zstdDecoder.DecodeAll(b, nil)
	if err != nil || len(out) == 0 {
		return nil, false
	}
	return out, true
}

// summarizeRich extracts a bounded human-readable summary from a rich-message
// XML body (image/emoji/app/video). It never fails: unknown shapes return "".
func summarizeRich(msgType, body string) string {
	title := xmlTagValue(body, "title")
	switch msgType {
	case "app":
		// File/link/mini-program shares carry the file or page name in title.
		if title != "" {
			return title
		}
		if fn := xmlAttrValue(body, "filename"); fn != "" {
			return fn
		}
	case "emoji":
		if m := xmlAttrValue(body, "md5"); m != "" {
			return "[表情] md5=" + m
		}
		return "[表情]"
	case "image":
		if m := xmlAttrValue(body, "md5"); m != "" {
			return "[图片] md5=" + m
		}
		return "[图片]"
	case "video":
		if m := xmlAttrValue(body, "md5"); m != "" {
			return "[视频] md5=" + m
		}
		return "[视频]"
	case "voice":
		if l := xmlAttrValue(body, "voicelength"); l != "" {
			return "[语音] " + l + "ms"
		}
		return "[语音]"
	}
	if title != "" {
		return title
	}
	return ""
}

// xmlTagValue returns the text of the first <tag> element, bounded scan.
func xmlTagValue(body, tag string) string {
	open := "<" + tag + ">"
	close := "</" + tag + ">"
	i := strings.Index(body, open)
	if i < 0 {
		return ""
	}
	rest := body[i+len(open):]
	j := strings.Index(rest, close)
	if j < 0 || j > 4096 {
		return ""
	}
	return strings.TrimSpace(rest[:j])
}

// xmlAttrValue returns the first attr="value" occurrence, bounded scan.
func xmlAttrValue(body, attr string) string {
	needle := attr + `="`
	i := strings.Index(body, needle)
	if i < 0 {
		return ""
	}
	rest := body[i+len(needle):]
	j := strings.Index(rest, `"`)
	if j < 0 || j > 4096 {
		return ""
	}
	return rest[:j]
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case string:
		var n int64
		fmt.Sscanf(t, "%d", &n)
		return n
	case []byte:
		var n int64
		fmt.Sscanf(string(t), "%d", &n)
		return n
	}
	return 0
}

func toText(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}

func normalizeDirection(v any) string {
	switch t := v.(type) {
	case int64:
		switch t {
		case 1:
			return DirectionOut
		case 0:
			return DirectionIn
		}
	case int:
		return normalizeDirection(int64(t))
	case float64:
		return normalizeDirection(int64(t))
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "1", "out", "outgoing", "send", "sent":
			return DirectionOut
		case "0", "in", "incoming", "receive", "received":
			return DirectionIn
		}
	case []byte:
		return normalizeDirection(string(t))
	}
	return DirectionUnknown
}

func safeText(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, utf8.ValidString(t)
	case []byte:
		if !utf8.Valid(t) {
			return "", false
		}
		return string(t), true
	case nil:
		return "", true
	default:
		return fmt.Sprintf("%v", t), true
	}
}

func rawBytes(v any) ([]byte, bool) {
	switch t := v.(type) {
	case []byte:
		return append([]byte(nil), t...), true
	case string:
		return []byte(t), true
	default:
		return nil, false
	}
}

func isNumericValue(v any) bool {
	switch t := v.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	case string:
		_, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return err == nil
	case []byte:
		_, err := strconv.ParseInt(strings.TrimSpace(string(t)), 10, 64)
		return err == nil
	default:
		return false
	}
}

func semanticSenderEmpty(v any) bool {
	if v == nil {
		return true
	}
	if isNumericValue(v) {
		return false
	}
	s, ok := safeText(v)
	return !ok || strings.TrimSpace(s) == ""
}

func putRawBytes(m *Message, key string, b []byte) {
	if m.StructuredContent == nil {
		m.StructuredContent = map[string]any{}
	}
	m.StructuredContent[key+"_base64"] = base64.StdEncoding.EncodeToString(b)
	m.StructuredContent[key+"_encoding"] = "base64"
}

func putNumber(m *Message, key string, n int64) {
	if m.StructuredContent == nil {
		m.StructuredContent = map[string]any{}
	}
	m.StructuredContent[key] = n
}

func setPartial(m *Message, warning string) {
	if m.ParseStatus == ParseOK {
		m.ParseStatus = ParsePartial
	}
	m.Warnings = append(m.Warnings, warning)
}

// probeConversations extracts conversation identity from session-like tables
// (name containing "session"). A row is linked to a message table when a
// string column equals the table name, or — WeChat 4.1 rule — when md5 of the
// column value matches the table's Msg_ hash suffix. The msgSet maps
// lowercased table name -> actual-case name and must be the global set
// accumulated across DBs: session catalogs and message tables live in
// different files.
func probeConversations(db *sql.DB, msgSet map[string]string) (names map[string]string, kinds map[string]string, convUsers map[string]string, warnings []string) {
	names = map[string]string{}
	kinds = map[string]string{}
	convUsers = map[string]string{}

	tables, err := listTables(db)
	if err != nil {
		return names, kinds, convUsers, []string{"session probe: " + err.Error()}
	}
	for _, t := range tables {
		if !strings.Contains(strings.ToLower(t), "session") {
			continue
		}
		rows, err := db.Query(`SELECT * FROM ` + quoteIdent(t) + ` LIMIT 10000`)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("session table %s unreadable: %v", t, err))
			continue
		}
		cols, _ := rows.Columns()
		for rows.Next() {
			raw := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range raw {
				ptrs[i] = &raw[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				break
			}
			row := map[string]string{}
			match := ""
			username := ""
			for i, c := range cols {
				s := toText(raw[i])
				row[strings.ToLower(c)] = s
				if actual, ok := msgSet[strings.ToLower(s)]; ok && s != "" {
					match = actual
				} else if actual, ok := msgSet[convTableName(s)]; ok && s != "" {
					match = actual
					username = s
				}
			}
			if match == "" {
				continue
			}
			if username != "" {
				convUsers[match] = username
				if strings.HasSuffix(username, "@chatroom") {
					kinds[match] = "group"
				} else {
					kinds[match] = "single"
				}
			}
			if n := firstNonEmpty(row, "nickname", "nick_name", "remark", "name", "displayname", "title", "session_title"); n != "" {
				names[match] = n
			}
			if kinds[match] == "" {
				if v := firstNonEmpty(row, "type", "kind", "chatroomtype"); v != "" && v != "0" {
					kinds[match] = "group"
				} else {
					kinds[match] = "single"
				}
			}
		}
		rows.Close()
	}
	return names, kinds, convUsers, warnings
}

// probeName2ID reads this DB's own Name2Id catalog: numeric rowid -> username.
// Every WeChat 4.1 DB carries an independent catalog, so callers must key the
// result by database and never merge catalogs across DBs.
func probeName2ID(db *sql.DB) (map[string]string, []string) {
	out := map[string]string{}
	var warnings []string
	tables, err := listTables(db)
	if err != nil {
		return out, []string{"name2id probe: " + err.Error()}
	}
	for _, t := range tables {
		if !strings.EqualFold(t, "Name2Id") {
			continue
		}
		cols, err := tableColumns(db, t)
		if err != nil || cols["user_name"] == "" {
			continue
		}
		rows, err := db.Query(`SELECT rowid, user_name FROM "Name2Id" LIMIT 100000`)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Name2Id unreadable: %v", err))
			continue
		}
		for rows.Next() {
			var rid sql.NullInt64
			var uname sql.NullString
			if err := rows.Scan(&rid, &uname); err != nil {
				break
			}
			if rid.Valid && uname.Valid && uname.String != "" {
				out[strconv.FormatInt(rid.Int64, 10)] = uname.String
			}
		}
		rows.Close()
	}
	return out, warnings
}

// convTableName is the WeChat 4.1 message-table naming rule: "Msg_" + md5 of
// the conversation username (single chats and chatrooms alike).
func convTableName(username string) string {
	sum := md5.Sum([]byte(username))
	return "msg_" + hex.EncodeToString(sum[:])
}

func firstNonEmpty(row map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(row[k]); v != "" {
			return v
		}
	}
	return ""
}

// probeContacts extracts username -> display name from contact-like tables.
func probeContacts(db *sql.DB) (map[string]string, []string) {
	out := map[string]string{}
	var warnings []string
	tables, err := listTables(db)
	if err != nil {
		return out, []string{"contacts probe: " + err.Error()}
	}
	for _, t := range tables {
		lt := strings.ToLower(t)
		if !strings.Contains(lt, "contact") && !strings.Contains(lt, "stranger") && !strings.Contains(lt, "friend") {
			continue
		}
		cols, err := tableColumns(db, t)
		if err != nil {
			continue
		}
		userCol := pickCol(cols, []string{"username", "userid", "user_id", "wxid", "id"})
		if userCol == "" {
			continue
		}
		// WeChat 4.1 contact rows: remark (user-set) outranks nick_name.
		nameCols := []string{}
		for _, nc := range []string{"remark", "nick_name", "nickname", "displayname", "name", "alias"} {
			if _, ok := cols[nc]; ok {
				nameCols = append(nameCols, nc)
			}
		}
		if len(nameCols) == 0 {
			continue
		}
		sel := []string{quoteIdent(userCol)}
		for _, nc := range nameCols {
			sel = append(sel, quoteIdent(nc))
		}
		rows, err := db.Query(fmt.Sprintf(`SELECT %s FROM %s LIMIT 100000`,
			strings.Join(sel, ", "), quoteIdent(t)))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("contact table %s unreadable: %v", t, err))
			continue
		}
		for rows.Next() {
			vals := make([]sql.NullString, len(nameCols)+1)
			ptrs := make([]any, len(vals))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				break
			}
			u := vals[0]
			if !u.Valid || u.String == "" {
				continue
			}
			display := ""
			for _, v := range vals[1:] {
				if v.Valid && strings.TrimSpace(v.String) != "" {
					display = v.String
					break
				}
			}
			if display == "" {
				continue
			}
			if _, exists := out[u.String]; !exists {
				out[u.String] = display
			}
		}
		rows.Close()
	}
	return out, warnings
}

// probeOwnSenders reads only explicitly self-user tables. A generic numeric
// id from an unrelated table must never be treated as the account identity.
func probeOwnSenders(db *sql.DB) (map[string]bool, string, []string) {
	out := map[string]bool{}
	var accountName string
	var warnings []string
	tables, err := listTables(db)
	if err != nil {
		return out, "", []string{"self-user probe: " + err.Error()}
	}
	for _, t := range tables {
		lt := strings.ToLower(t)
		if !strings.Contains(lt, "my_user") && !strings.Contains(lt, "self") {
			continue
		}
		cols, err := tableColumns(db, t)
		if err != nil {
			continue
		}
		idCol := pickCol(cols, []string{"userid", "user_id", "username", "wxid", "id", "localid", "local_id"})
		nameCol := pickCol(cols, []string{"nickname", "remark", "displayname", "name", "alias"})
		if idCol == "" {
			continue
		}
		queryCols := []string{"rowid", quoteIdent(idCol)}
		if nameCol != "" && !strings.EqualFold(nameCol, idCol) {
			queryCols = append(queryCols, quoteIdent(nameCol))
		}
		rows, err := db.Query("SELECT " + strings.Join(queryCols, ",") + " FROM " + quoteIdent(t) + " LIMIT 100")
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("self-user table %s unreadable: %v", t, err))
			continue
		}
		for rows.Next() {
			var rowID sql.NullInt64
			var id any
			var name sql.NullString
			dest := []any{&rowID, &id}
			if len(queryCols) == 3 {
				dest = append(dest, &name)
			}
			if err := rows.Scan(dest...); err != nil {
				break
			}
			if s, ok := safeText(id); ok && strings.TrimSpace(s) != "" {
				out[strings.TrimSpace(s)] = true
			}
			if rowID.Valid {
				out[strconv.FormatInt(rowID.Int64, 10)] = true
			}
			if accountName == "" && name.Valid && strings.TrimSpace(name.String) != "" {
				accountName = name.String
			}
		}
		rows.Close()
	}
	return out, accountName, warnings
}
