package export

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// renderSinks owns the streamed JSONL channel plus conversation metadata for
// one run. Markdown volumes are NOT written here: they are re-rendered from
// the merged JSONL afterwards (renderVolumesFromJSONL), so carried-forward
// history lands in correct time order instead of being appended at the end.
type renderSinks struct {
	cancelled func() bool
	dir       string
	accountID string

	jsonl   *os.File
	jsonlEn *json.Encoder

	conversations map[string]*Conversation
	totalMessages int64
}

func newRenderSinks(dir, accountID string) (*renderSinks, error) {
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	jf, err := os.OpenFile(filepath.Join(dataDir, "messages.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	return &renderSinks{
		dir:           dir,
		accountID:     accountID,
		jsonl:         jf,
		jsonlEn:       json.NewEncoder(jf),
		conversations: map[string]*Conversation{},
	}, nil
}

// emit streams one message to JSONL and tracks conversation metadata.
func (s *renderSinks) emit(m *Message) error {
	if s.cancelled != nil && s.cancelled() {
		return context.Canceled
	}
	if err := s.jsonlEn.Encode(m); err != nil {
		return fmt.Errorf("write jsonl: %w", err)
	}

	conv := s.conversations[m.ConversationID]
	if conv == nil {
		conv = &Conversation{
			ID:   m.ConversationID,
			Name: m.ConversationName,
			Kind: m.ConversationKind,
		}
		s.conversations[m.ConversationID] = conv
	}
	conv.Messages++
	if m.Timestamp != "" {
		if conv.FirstTime == "" || m.Timestamp < conv.FirstTime {
			conv.FirstTime = m.Timestamp
		}
		if conv.LastTime == "" || m.Timestamp > conv.LastTime {
			conv.LastTime = m.Timestamp
		}
	}
	s.totalMessages++
	return nil
}

// Close flushes and closes the JSONL sink.
func (s *renderSinks) Close() error {
	if s.jsonl == nil {
		return nil
	}
	f := s.jsonl
	s.jsonl = nil
	err := f.Sync()
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

// jsonlPath returns the staged JSONL path.
func (s *renderSinks) jsonlPath() string {
	return filepath.Join(s.dir, "data", "messages.jsonl")
}

// writeConversationsFile writes data/conversations.json.
func writeConversationsFile(dir string, s *renderSinks) (string, error) {
	list := make([]*Conversation, 0, len(s.conversations))
	for _, c := range s.conversations {
		list = append(list, c)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	b, err := json.MarshalIndent(map[string]any{
		"schema":        SchemaVersion,
		"conversations": list,
	}, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "data", "conversations.json")
	return path, os.WriteFile(path, b, 0o600)
}

// writeContactsFile writes data/contacts.json.
func writeContactsFile(dir string, contacts map[string]string) (string, error) {
	keys := make([]string, 0, len(contacts))
	for k := range contacts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := make(map[string]string, len(contacts))
	for _, k := range keys {
		ordered[k] = contacts[k]
	}
	b, err := json.MarshalIndent(map[string]any{
		"schema":   SchemaVersion,
		"contacts": ordered,
	}, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "data", "contacts.json")
	return path, os.WriteFile(path, b, 0o600)
}

// renderIndex renders the human-facing index.md.
func renderIndex(rep *Report, s *renderSinks) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# 微信聊天记录导出\n\n")
	fmt.Fprintf(&b, "- 账号：`%s`\n", rep.AccountID)
	fmt.Fprintf(&b, "- 导出时间：%s\n", rep.Started)
	fmt.Fprintf(&b, "- 导出版本：%s\n", rep.Schema)
	fmt.Fprintf(&b, "- 完整性状态：**%s**\n", rep.Status)
	fmt.Fprintf(&b, "- 消息总数：%d；会话数：%d\n\n", rep.TotalMessages, rep.TotalConversations)

	fmt.Fprintf(&b, "## 会话\n\n")
	ids := make([]string, 0, len(s.conversations))
	for id := range s.conversations {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		c := s.conversations[id]
		name := c.Name
		if name == "" {
			name = c.ID
		}
		dir := sanitizeFileComponent(c.ID)
		fmt.Fprintf(&b, "- [%s](accounts/%s/conversations/%s/)：%d 条（%s ~ %s）\n",
			name, sanitizeFileComponent(rep.AccountID), dir, c.Messages, c.FirstTime, c.LastTime)
	}

	fmt.Fprintf(&b, "\n## 数据源\n\n")
	for _, d := range rep.DBs {
		line := fmt.Sprintf("- `%s`：%d 条消息", d.Path, d.Messages)
		if d.Error != "" {
			line += "；**失败**：" + d.Error
		}
		fmt.Fprintf(&b, "%s\n", line)
	}
	fmt.Fprintf(&b, "\n详细审计信息见 [manifest.json](manifest.json)；结构化数据见 `data/`。\n")
	return b.String()
}

func kindLabel(kind string) string {
	switch kind {
	case "single":
		return "单聊"
	case "group":
		return "群聊"
	default:
		return "未知"
	}
}

// enrichContext runs pass 1 over one plaintext snapshot: contact display
// names, this DB's Name2Id catalog, and conversation names/kinds are merged
// into the shared context. dbRel keys the per-database Name2Id catalog.
func enrichContext(plainPath string, dbRel string, ctx *parseContext) ([]string, error) {
	db, err := openSnapshot(plainPath)
	if err != nil {
		return nil, fmt.Errorf("open snapshot: %w", err)
	}
	defer db.Close()

	if ctx.convUsers == nil {
		ctx.convUsers = map[string]string{}
	}

	var warnings []string
	contacts, w := probeContacts(db)
	for k, v := range contacts {
		if _, ok := ctx.contacts[k]; !ok {
			ctx.contacts[k] = v
		}
	}
	warnings = append(warnings, w...)
	ownIDs, accountName, w3 := probeOwnSenders(db)
	for id := range ownIDs {
		ctx.ownSenderIDs[id] = true
	}
	if ctx.accountName == "" {
		ctx.accountName = accountName
	}
	warnings = append(warnings, w3...)

	if catalog, wn := probeName2ID(db); len(catalog) > 0 {
		if ctx.name2id == nil {
			ctx.name2id = map[string]map[string]string{}
		}
		ctx.name2id[dbRel] = catalog
		// The account's own Name2Id rowid in this DB is authoritative
		// self-evidence for direction classification.
		for numID, uname := range catalog {
			if strings.EqualFold(uname, ctx.ownUsername()) {
				ctx.ownSenderIDs[numID] = true
			}
		}
		warnings = append(warnings, wn...)
	}

	msgTables, _, err := probeMessageTables(db)
	if err != nil {
		return warnings, err
	}
	if ctx.allMsgTables == nil {
		ctx.allMsgTables = map[string]string{}
	}
	for _, ti := range msgTables {
		ctx.allMsgTables[strings.ToLower(ti.name)] = ti.name
	}
	// Name2Id reverse lookup: md5(catalog username) matching a message table
	// hash identifies the conversation even when SessionTable has lost the
	// row (deleted/inactive sessions). Session rows, when present, win.
	if catalog := ctx.name2id[dbRel]; catalog != nil {
		for _, uname := range catalog {
			if uname == "" {
				continue
			}
			if actual, ok := ctx.allMsgTables[convTableName(uname)]; ok {
				ctx.convUsers[actual] = uname
				if ctx.convKinds[actual] == "" {
					if strings.HasSuffix(uname, "@chatroom") {
						ctx.convKinds[actual] = "group"
					} else {
						ctx.convKinds[actual] = "single"
					}
				}
				if ctx.convNames[actual] == "" {
					if n := ctx.contacts[uname]; n != "" {
						ctx.convNames[actual] = n
					} else {
						ctx.convNames[actual] = uname
					}
				}
			}
		}
	}
	names, kinds, convUsers, w2 := probeConversations(db, ctx.allMsgTables)
	for k, v := range names {
		ctx.convNames[k] = v
	}
	for k, v := range kinds {
		ctx.convKinds[k] = v
	}
	if ctx.convUsers == nil {
		ctx.convUsers = map[string]string{}
	}
	for k, v := range convUsers {
		ctx.convUsers[k] = v
	}
	// Conversation display name: contact catalog (remark > nickname) beats
	// session-row fields; username is the final fallback.
	for table, uname := range convUsers {
		if ctx.convNames[table] == "" {
			if n := ctx.contacts[uname]; n != "" {
				ctx.convNames[table] = n
			} else {
				ctx.convNames[table] = uname
			}
		}
	}
	warnings = append(warnings, w2...)
	return warnings, nil
}

// streamDBMessages runs pass 2 over one plaintext snapshot: message tables
// are probed and streamed through the merger (per-conversation boundaries)
// into the sinks in deterministic order.
func streamDBMessages(plainPath string, src sourceDB, ctx *parseContext, m *merger, sinks *renderSinks) (int64, int64, []SkippedTable, []string, error) {
	db, err := openSnapshot(plainPath)
	if err != nil {
		return 0, 0, nil, nil, fmt.Errorf("open snapshot: %w", err)
	}
	defer db.Close()

	msgTables, skipped, err := probeMessageTables(db)
	if err != nil {
		return 0, 0, skipped, nil, err
	}

	sub := *ctx
	sub.dbRel = src.rel
	sub.shard = filepath.Base(src.path)

	var count int64
	var partial int64
	for _, ti := range msgTables {
		m.beginConversation(ti.name)
		n, err := readTableMessages(db, ti, &sub, func(msg *Message) error {
			if msg.ParseStatus != ParseOK {
				partial++
			}
			m.emit(msg)
			return nil
		})
		if err != nil {
			return count, partial, skipped, nil, err
		}
		if err := m.endConversation(sinks.emit, ""); err != nil {
			return count, partial, skipped, nil, err
		}
		count += n
	}
	return count, partial, skipped, nil, nil
}
