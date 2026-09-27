package export

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// merger implements the incremental merge layer between the parser and the
// render sinks. Semantics (chat-v1):
//
//   - identity = (account, conversation, message_id); the run's current source
//     messages are the authority for what exists NOW;
//   - a message whose content hash changed since the previous export is a
//     REVISION (revision counter increments), not a new message;
//   - a previous message absent from the current source is CARRIED FORWARD
//     with a "missing-in-source" warning - it is never auto-deleted, because
//     absence from one snapshot is not evidence of deletion;
//   - duplicates of the same identity inside one run are dropped and counted;
//
// Memory stays bounded by one conversation at a time plus the previous run's
// index (key -> hash/revision), not the full message bodies.
type merger struct {
	enabled bool
	account string
	ctx     context.Context

	// prev: identity key -> previous state (hash for change detection,
	// revision to continue the chain).
	prev map[string]prevEntry
	// seen: identities emitted by the current run.
	seen map[string]bool
	// droppedDupes counts same-identity duplicates within the current run.
	droppedDupes int64

	// per-conversation buffer, flushed at conversation end.
	conv   string
	buf    []*Message
	bufRev map[string]int // id -> revision for buffered messages

	stats MergeStats
}

// prevEntry is one previously exported message's merge-relevant state.
type prevEntry struct {
	hash     string
	revision int
}

// MergeStats lands in the manifest.
type MergeStats struct {
	New       int64 `json:"new"`
	Unchanged int64 `json:"unchanged"`
	Revised   int64 `json:"revised"`
	Carried   int64 `json:"carried_forward"`
	Dupes     int64 `json:"duplicates_dropped"`
}

func mergeKey(m *Message) string {
	return m.AccountID + " " + m.ConversationID + " " + m.MessageID
}

// contentHash covers the fields whose change constitutes a revision.
func contentHash(m *Message) string {
	h := sha256.New()
	h.Write([]byte(m.Text))
	h.Write([]byte{0})
	h.Write([]byte(m.Type))
	h.Write([]byte{0})
	fmt.Fprintf(h, "%d %s %s %s", m.SourceType, m.SenderID, m.Direction, m.Timestamp)
	if m.StructuredContent != nil {
		b, _ := json.Marshal(m.StructuredContent)
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// loadPrevIndex streams the previous messages.jsonl, building the merge
// index. Only an absent file means an empty index. Malformed or cross-account
// history must not be silently discarded or carried into another account.
func loadPrevIndex(ctx context.Context, path, account string) (map[string]prevEntry, error) {
	idx := map[string]prevEntry{}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return idx, nil
		}
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
	for sc.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var m Message
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, fmt.Errorf("invalid published message history: %w", err)
		}
		if m.AccountID != account {
			return nil, fmt.Errorf("message history contains a different account")
		}
		idx[mergeKey(&m)] = prevEntry{hash: contentHash(&m), revision: m.Revision}
	}
	return idx, sc.Err()
}

func newMerger(ctx context.Context, prevPath string, enabled bool, account string) (*merger, error) {
	m := &merger{enabled: enabled, ctx: ctx, account: account, seen: map[string]bool{}, bufRev: map[string]int{}}
	if enabled {
		var err error
		m.prev, err = loadPrevIndex(ctx, prevPath, account)
		if err != nil {
			return nil, err
		}
	} else {
		m.prev = map[string]prevEntry{}
	}
	return m, nil
}

// beginConversation starts buffering one conversation's current messages.
func (m *merger) beginConversation(convID string) {
	m.conv = convID
	m.buf = nil
}

// emit buffers one current-source message after applying merge semantics.
func (m *merger) emit(msg *Message) {
	key := mergeKey(msg)
	if m.seen[key] {
		m.droppedDupes++
		return
	}
	m.seen[key] = true
	if m.enabled {
		if p, ok := m.prev[key]; ok {
			if p.hash == contentHash(msg) {
				msg.Revision = p.revision
				m.stats.Unchanged++
			} else {
				msg.Revision = p.revision + 1
				m.stats.Revised++
			}
		} else {
			m.stats.New++
		}
	} else {
		// Full (non-incremental) run: everything is "new" for accounting.
		m.stats.New++
	}
	m.buf = append(m.buf, msg)
}

// endConversation flushes the buffered conversation: current messages in
// their (already deterministic) order, then carried-forward previous messages
// for this conversation that the current source no longer contains.
func (m *merger) endConversation(emit func(*Message) error, prevPath string) error {
	for _, msg := range m.buf {
		if err := emit(msg); err != nil {
			return err
		}
	}
	m.buf = nil
	return nil
}

// flushCarried streams the previous JSONL a second time and emits every
// message whose identity the current run never produced. These are appended
// after all current messages, grouped by conversation, each marked with a
// "missing-in-source" warning. (Ordering note: carried messages keep their
// original timestamps; within a volume they follow current messages.)
func (m *merger) flushCarried(prevPath string, emit func(*Message) error) error {
	if !m.enabled {
		return nil
	}
	f, err := os.Open(prevPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
	for sc.Scan() {
		if err := m.ctx.Err(); err != nil {
			return err
		}
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var pm Message
		if err := json.Unmarshal(line, &pm); err != nil {
			return fmt.Errorf("invalid published message history: %w", err)
		}
		if pm.AccountID != m.account {
			return fmt.Errorf("carried history contains a different account")
		}
		if m.seen[mergeKey(&pm)] {
			continue
		}
		pm.Warnings = appendMissingWarning(pm.Warnings)
		if err := emit(&pm); err != nil {
			return err
		}
		m.seen[mergeKey(&pm)] = true
		m.stats.Carried++
	}
	return sc.Err()
}

func appendMissingWarning(w []string) []string {
	const marker = "missing-in-source"
	for _, x := range w {
		if x == marker {
			return w
		}
	}
	return append(append([]string{}, w...), marker)
}

// sortedKeys is a helper for deterministic tests/debugging.
func (m *merger) sortedKeys() []string {
	out := make([]string, 0, len(m.seen))
	for k := range m.seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

var _ = strings.TrimSpace
