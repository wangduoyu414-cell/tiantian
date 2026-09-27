package export

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// renderVolumesFromJSONL re-renders per-conversation Markdown volumes from the
// merged JSONL, so carried-forward history appears in correct time order.
//
// Memory stays bounded by one conversation at a time: pass A records each
// line's conversation and byte offset; pass B reads one conversation's lines
// (via offsets), sorts by (time, id), and writes its yyyy-MM volumes.
func renderVolumesFromJSONL(jsonlPath, filesDir, accountID string, convs map[string]*Conversation) error {
	return renderVolumesFromJSONLContext(context.Background(), jsonlPath, filesDir, accountID, convs)
}
func renderVolumesFromJSONLContext(ctx context.Context, jsonlPath, filesDir, accountID string, convs map[string]*Conversation) error {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return err
	}
	// Pass A: index conversation -> line offsets.
	index := map[string][]int64{}
	{
		r := bufio.NewReader(f)
		var off int64
		for {
			if err := ctx.Err(); err != nil {
				f.Close()
				return err
			}
			line, err := r.ReadBytes('\n')
			if len(line) > 0 {
				var m Message
				if json.Unmarshal(line[:len(line)-trailingNL(line)], &m) == nil {
					index[m.ConversationID] = append(index[m.ConversationID], off)
				}
				off += int64(len(line))
			}
			if err != nil {
				if !errors.Is(err, io.EOF) {
					f.Close()
					return err
				}
				break
			}
		}
	}
	f.Close()

	// Deterministic conversation order.
	ids := make([]string, 0, len(index))
	for id := range index {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, convID := range ids {
		if err := renderOneConversation(ctx, jsonlPath, filesDir, accountID, convID, index[convID], convs[convID]); err != nil {
			return err
		}
	}
	return nil
}

func trailingNL(b []byte) int {
	n := 0
	for len(b)-n > 0 && (b[len(b)-1-n] == '\n' || b[len(b)-1-n] == '\r') {
		n++
	}
	return n
}

// renderOneConversation reads one conversation's messages by offset, sorts
// them by (time, id), and writes month volumes.
func renderOneConversation(ctx context.Context, jsonlPath, filesDir, accountID, convID string, offsets []int64, meta *Conversation) error {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return err
	}
	defer f.Close()

	msgs := make([]*Message, 0, len(offsets))
	for _, off := range offsets {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := f.Seek(off, 0); err != nil {
			return err
		}
		line, err := bufio.NewReader(f).ReadBytes('\n')
		if err != nil && len(line) == 0 {
			return err
		}
		var m Message
		if err := json.Unmarshal(line[:len(line)-trailingNL(line)], &m); err != nil {
			continue // tolerate truncated tail
		}
		msgs = append(msgs, &m)
	}

	sort.SliceStable(msgs, func(i, j int) bool {
		a, b := msgs[i], msgs[j]
		if a.Timestamp != b.Timestamp {
			return a.Timestamp < b.Timestamp
		}
		return a.MessageID < b.MessageID
	})

	name := convID
	kind := "unknown"
	if meta != nil {
		if meta.Name != "" {
			name = meta.Name
		}
		kind = meta.Kind
	}

	convDir := filepath.Join(filesDir, "accounts", sanitizeFileComponent(accountID), "conversations", sanitizeFileComponent(convID))
	if err := os.MkdirAll(convDir, 0o700); err != nil {
		return err
	}

	var cur *os.File
	curMonth := ""
	closeCur := func() error {
		if cur == nil {
			return nil
		}
		if err := cur.Sync(); err != nil {
			return err
		}
		return cur.Close()
	}
	defer func() {
		if cur != nil {
			cur.Close()
		}
	}()
	for _, m := range msgs {
		if err := ctx.Err(); err != nil {
			return err
		}
		month := "unknown-time"
		if len(m.Timestamp) >= 7 {
			month = m.Timestamp[:7]
		}
		if month != curMonth {
			if err := closeCur(); err != nil {
				return err
			}
			nf, err := os.OpenFile(filepath.Join(convDir, month+".md"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
			if err != nil {
				return err
			}
			cur = nf
			curMonth = month
			fmt.Fprintf(cur, "# %s\n\n", name)
			fmt.Fprintf(cur, "- 会话 ID：`%s`\n- 类型：%s\n- 账号：`%s`\n- 分卷：%s\n- 导出版本：%s\n\n",
				convID, kindLabel(kind), accountID, month, SchemaVersion)
			fmt.Fprintf(cur, "---\n\n")
		}
		if _, err := cur.WriteString(renderMessageMarkdown(m)); err != nil {
			return err
		}
	}
	return closeCur()
}
