// Package export turns decrypted WeChat database snapshots into the chat-v1
// standard model plus Markdown/JSONL artifacts.
//
// Schema probing is evidence-based: table and column names are matched
// against candidate lists, never assumed. Anything unrecognized is preserved
// raw and flagged, not silently dropped.
package export

// SchemaVersion is the fixed contract version of the standard model. It does
// NOT follow WeChat versions.
const SchemaVersion = "chat-v1"

// Direction of a message.
const (
	DirectionOut     = "out"
	DirectionIn      = "in"
	DirectionSystem  = "system"
	DirectionUnknown = "unknown"
)

// Parse status.
const (
	ParseOK      = "parsed"
	ParsePartial = "partial"
	ParseUnknown = "unknown"
)

// Provenance ties a message back to its exact source record. Snapshot-level
// evidence (source size/mtime, WAL replay) lives in manifest.json per DB,
// so identical sources re-render byte-identical messages.
type Provenance struct {
	DB    string `json:"db"`    // DB path relative to the account db_storage root
	Table string `json:"table"` // source table
	RowID int64  `json:"rowid"` // source rowid
	Shard string `json:"shard"` // DB file name (msg_0.db, ...)
}

// AttachmentState records that attachments exist but were not extracted yet.
type AttachmentState struct {
	Kind   string `json:"kind"`
	Status string `json:"status"` // "missing" | "not-extracted" | "unsupported"
	Ref    string `json:"ref,omitempty"`
}

// Message is the chat-v1 standard message.
type Message struct {
	SchemaVersion    string `json:"schema_version"`
	AccountID        string `json:"account_id"`
	ConversationID   string `json:"conversation_id"`
	ConversationName string `json:"conversation_name,omitempty"`
	ConversationKind string `json:"conversation_kind"` // single|group|unknown
	MessageID        string `json:"message_id"`
	IDSource         string `json:"id_source"` // server_id|local_id|rowid_fallback
	SenderID         string `json:"sender_id,omitempty"`
	SenderName       string `json:"display_name,omitempty"`
	Direction        string `json:"direction"`
	Timestamp        string `json:"timestamp"` // RFC3339 with offset
	SourceTimestamp  int64  `json:"source_timestamp"`
	SourceTimeUnit   string `json:"source_time_unit"` // s|ms|unknown
	Type             string `json:"type"`             // normalized
	SourceType       int64  `json:"source_type"`
	Text             string `json:"text,omitempty"`
	// StructuredContent preserves unparsed raw fields (quoted/XML payloads,
	// unknown-type bodies) so nothing is silently lost.
	StructuredContent map[string]any    `json:"structured_content,omitempty"`
	Attachments       []AttachmentState `json:"attachments,omitempty"`
	Provenance        Provenance        `json:"provenance"`
	ParseStatus       string            `json:"parse_status"`
	Warnings          []string          `json:"warnings,omitempty"`
	Revision          int               `json:"revision"`
}

// Conversation summarizes one conversation.
type Conversation struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Kind      string `json:"kind"` // single|group|unknown
	Messages  int64  `json:"messages"`
	FirstTime string `json:"first_time,omitempty"`
	LastTime  string `json:"last_time,omitempty"`
}

// SkippedTable records a table the parser deliberately did not consume.
type SkippedTable struct {
	Table  string `json:"table"`
	Reason string `json:"reason"`
}
