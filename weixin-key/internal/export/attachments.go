package export

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// attachmentState values written into Message.Attachments.
const (
	attCopied          = "copied"           // plaintext bytes verified by size+sha256
	attDecrypted       = "decrypted"        // .dat container decoded to a plain image
	attStoredEncrypted = "stored-encrypted" // copied but still .dat-encrypted (no key supplied)
	attMissing         = "missing"          // source file/record not found
	attNotExtracted    = "not-extracted"    // known kind, no extractor yet
)

// attacher resolves message attachments from the account's on-disk msg/ tree
// and from VoiceInfo rows in media DB snapshots. It writes into the staging
// files dir (published with everything else) and never mutates sources.
type attacher struct {
	dbRoot   string            // account root containing msg/
	filesDir string            // staging files dir; attachments go under attachments/
	convUser map[string]string // message table name -> conversation username
	voice    map[string][]byte // "<tableLower>/<localID>" -> silk bytes (0x02+#!SILK_V3)
	resSize  map[string][]int64
	imgKey   []byte // optional WeChat v4 image AES-128 key; nil keeps .dat encrypted
	xorKey   byte   // account-global tail key, learned from the first JPEG decode
	hasXor   bool
}

// voiceKey joins a message table and local id for VoiceInfo lookup.
func voiceKey(table string, localID int64) string {
	return strings.ToLower(table) + "/" + fmtInt(localID)
}

func fmtInt(v int64) string {
	return strings.TrimSpace(fmt.Sprintf("%d", v))
}

// probeVoice reads VoiceInfo from a media DB snapshot: chat_name_id resolves
// through this DB's own Name2Id to the conversation username; local_id joins
// the conversation's message table. Only rows with non-empty payloads count.
func probeVoice(db *sql.DB) (map[string][]byte, []string) {
	out := map[string][]byte{}
	var warnings []string
	tables, err := listTables(db)
	if err != nil {
		return out, []string{"voice probe: " + err.Error()}
	}
	have := false
	for _, t := range tables {
		if strings.EqualFold(t, "VoiceInfo") {
			have = true
		}
	}
	if !have {
		return out, nil
	}
	catalog, w := probeName2ID(db)
	warnings = append(warnings, w...)
	idByName := map[string]string{}
	for rid, uname := range catalog {
		idByName[uname] = rid
	}
	rows, err := db.Query(`SELECT chat_name_id, local_id, voice_data FROM VoiceInfo LIMIT 200000`)
	if err != nil {
		return out, append(warnings, "VoiceInfo unreadable: "+err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		var chatID, localID sql.NullInt64
		var data []byte
		if err := rows.Scan(&chatID, &localID, &data); err != nil {
			break
		}
		if !chatID.Valid || !localID.Valid || len(data) == 0 {
			continue
		}
		uname := catalog[fmtInt(chatID.Int64)]
		if uname == "" {
			continue
		}
		out[voiceKey(convTableName(uname), localID.Int64)] = append([]byte(nil), data...)
	}
	return out, warnings
}

// probeResourceSizes reads message_resource.db-style snapshots and returns
// "<tableLower>/<localID>" -> resource sizes (thumb/image/file byte counts).
func probeResourceSizes(db *sql.DB) (map[string][]int64, []string) {
	out := map[string][]int64{}
	var warnings []string
	tables, err := listTables(db)
	if err != nil {
		return out, []string{"resource probe: " + err.Error()}
	}
	haveInfo, haveDetail, haveChat := false, false, false
	for _, t := range tables {
		switch {
		case strings.EqualFold(t, "MessageResourceInfo"):
			haveInfo = true
		case strings.EqualFold(t, "MessageResourceDetail"):
			haveDetail = true
		case strings.EqualFold(t, "ChatName2Id"):
			haveChat = true
		}
	}
	if !haveInfo || !haveDetail || !haveChat {
		return out, nil
	}
	chatRows, err := db.Query(`SELECT rowid, user_name FROM ChatName2Id LIMIT 100000`)
	if err != nil {
		return out, append(warnings, "ChatName2Id unreadable: "+err.Error())
	}
	chatByRowid := map[int64]string{}
	for chatRows.Next() {
		var rid int64
		var uname sql.NullString
		if err := chatRows.Scan(&rid, &uname); err != nil {
			break
		}
		if uname.Valid {
			chatByRowid[rid] = uname.String
		}
	}
	chatRows.Close()

	rows, err := db.Query(`SELECT i.chat_id, i.message_local_id, d.size
		FROM MessageResourceDetail d JOIN MessageResourceInfo i ON d.message_id = i.message_id
		WHERE d.size > 0 LIMIT 500000`)
	if err != nil {
		return out, append(warnings, "resource join unreadable: "+err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		var chatID, localID, size sql.NullInt64
		if err := rows.Scan(&chatID, &localID, &size); err != nil {
			break
		}
		if !chatID.Valid || !localID.Valid || !size.Valid || size.Int64 <= 0 {
			continue
		}
		uname := chatByRowid[chatID.Int64]
		if uname == "" {
			continue
		}
		k := voiceKey(convTableName(uname), localID.Int64)
		out[k] = append(out[k], size.Int64)
	}
	return out, warnings
}

// monthDir derives the msg/ bucket month (yyyy-MM) from the message time.
func monthDir(sourceTS int64) string {
	ts, unit := normalizeTime(sourceTS)
	if ts.IsZero() {
		return ""
	}
	if unit == "ms" {
		ts = ts.Local()
	}
	return ts.Local().Format("2006-01")
}

// copyVerified copies src to dst (fresh), returning the sha256; size must
// match exactly or the copy is refused.
func copyVerified(src, dst string, wantSize int64) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return "", err
	}
	if wantSize > 0 && info.Size() != wantSize {
		return "", fmt.Errorf("size mismatch: %d != %d", info.Size(), wantSize)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return "", err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(out, h), in); err != nil {
		out.Close()
		os.Remove(dst)
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// resolve fills Message.Attachments for media-type messages. It never fails
// the message: every outcome is an honest AttachmentState.
func (a *attacher) resolve(m *Message, table string, localID int64) {
	if a == nil {
		return
	}
	msgRef := sanitizeFileComponent(strings.ReplaceAll(m.MessageID, ":", "_"))
	convDir := sanitizeFileComponent(table)
	rel := func(name string) string { // path inside the export
		return "attachments/" + convDir + "/" + msgRef + "_" + name
	}
	switch m.Type {
	case "voice":
		data := a.voice[voiceKey(table, localID)]
		if len(data) == 0 {
			m.Attachments = []AttachmentState{{Kind: "voice", Status: attMissing}}
			return
		}
		name := rel("voice.silk")
		dst := filepath.Join(a.filesDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err == nil {
			if err := os.WriteFile(dst, data, 0o600); err == nil {
				m.Attachments = []AttachmentState{{Kind: "voice", Status: attCopied, Ref: name}}
				return
			}
		}
		m.Attachments = []AttachmentState{{Kind: "voice", Status: attNotExtracted}}

	case "image", "video":
		uname := a.convUser[table]
		month := monthDir(m.SourceTimestamp)
		if uname == "" || month == "" {
			m.Attachments = []AttachmentState{{Kind: m.Type, Status: attMissing}}
			return
		}
		// Images bucket per conversation+month; videos bucket per month only.
		var bucket string
		if m.Type == "video" {
			bucket = filepath.Join(a.dbRoot, "msg", "video", month)
		} else {
			bucket = filepath.Join(a.dbRoot, "msg", "attach", strings.TrimPrefix(convTableName(uname), "msg_"), month, "Img")
		}
		entries, err := os.ReadDir(bucket)
		if err != nil {
			m.Attachments = []AttachmentState{{Kind: m.Type, Status: attMissing}}
			return
		}
		// Link by exact byte size from the resource catalog; ambiguous or
		// absent sizes stay missing rather than guessing a file.
		wantSizes := a.resSize[voiceKey(table, localID)]
		var picked []string
		for _, ws := range wantSizes {
			var hits []string
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				if fi, err := e.Info(); err == nil && fi.Size() == ws {
					hits = append(hits, e.Name())
				}
			}
			if len(hits) == 1 {
				picked = append(picked, hits[0])
			}
		}
		if len(picked) == 0 {
			m.Attachments = []AttachmentState{{Kind: m.Type, Status: attMissing}}
			return
		}
		for _, p := range picked {
			srcPath := filepath.Join(bucket, p)
			// .dat containers are decoded in place when the image key is
			// available; otherwise the encrypted file is copied as evidence.
			if raw, err := os.ReadFile(srcPath); err == nil && isDatContainer(raw) {
				var fb *byte
				if a.hasXor {
					fb = &a.xorKey
				}
				if plain, ext, xk, derr := decodeDat(raw, a.imgKey, fb); derr == nil {
					if ext == "jpg" && !a.hasXor {
						a.xorKey, a.hasXor = xk, true
					}
					name := rel(sanitizeFileComponent(strings.TrimSuffix(p, ".dat")) + "." + ext)
					dst := filepath.Join(a.filesDir, filepath.FromSlash(name))
					if werr := os.MkdirAll(filepath.Dir(dst), 0o700); werr == nil {
						if werr := os.WriteFile(dst, plain, 0o600); werr == nil {
							m.Attachments = append(m.Attachments, AttachmentState{Kind: m.Type, Status: attDecrypted, Ref: name})
							continue
						}
					}
				}
			}
			name := rel(p)
			dst := filepath.Join(a.filesDir, filepath.FromSlash(name))
			if _, err := copyVerified(srcPath, dst, 0); err != nil {
				continue
			}
			status := attStoredEncrypted
			switch strings.ToLower(filepath.Ext(p)) {
			case ".mp4", ".jpg", ".jpeg", ".png":
				status = attCopied
			}
			m.Attachments = append(m.Attachments, AttachmentState{Kind: m.Type, Status: status, Ref: name})
		}

	case "app":
		// File shares: the XML title is the on-disk file name under
		// msg/file/<yyyy-MM>/, stored unencrypted.
		title := ""
		if raw, ok := m.StructuredContent["content_xml"].(string); ok {
			title = xmlTagValue(raw, "title")
		}
		month := monthDir(m.SourceTimestamp)
		if title == "" || month == "" || strings.ContainsAny(title, `/\`) {
			m.Attachments = []AttachmentState{{Kind: m.Type, Status: attNotExtracted}}
			return
		}
		src := filepath.Join(a.dbRoot, "msg", "file", month, title)
		if fi, err := os.Stat(src); err != nil || fi.IsDir() {
			m.Attachments = []AttachmentState{{Kind: m.Type, Status: attMissing}}
			return
		}
		name := rel(sanitizeFileComponent(title))
		dst := filepath.Join(a.filesDir, filepath.FromSlash(name))
		if _, err := copyVerified(src, dst, 0); err != nil {
			m.Attachments = []AttachmentState{{Kind: m.Type, Status: attNotExtracted}}
			return
		}
		m.Attachments = []AttachmentState{{Kind: m.Type, Status: attCopied, Ref: name}}

	default:
		m.Attachments = []AttachmentState{{Kind: m.Type, Status: attNotExtracted}}
	}
}
