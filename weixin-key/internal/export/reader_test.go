package export

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestMapRowToMessageRealSchemaUsesRealSenderAndPreservesBinaryFields(t *testing.T) {
	ti := tableInfo{
		name:           "Msg_real",
		colID:          "local_id",
		colServer:      "server_id",
		colTime:        "create_time",
		colType:        "local_type",
		colContent:     "message_content",
		colRealSender:  "real_sender_id",
		colCompress:    "compress_content",
		colPacked:      "packed_info_data",
		colContentType: "wcdb_ct_message_content",
		colSourceType:  "wcdb_ct_source",
	}
	ctx := &parseContext{
		accountID: "wxid_me",
		contacts:  map[string]string{"wxid_friend": "朋友"},
		convNames: map[string]string{},
		convKinds: map[string]string{},
	}
	cols := []string{
		"local_id", "server_id", "create_time", "local_type",
		"message_content", "real_sender_id", "compress_content",
		"packed_info_data", "wcdb_ct_message_content", "wcdb_ct_source",
	}
	raw := []any{
		int64(7), int64(7007), int64(1715000000), int64(1),
		[]byte{0xff, 0x00, 0xfe}, "wxid_friend", int64(0),
		[]byte{0x01, 0x02, 0x03}, int64(1), int64(0),
	}
	m := mapRowToMessage(ti, cols, raw, ctx)
	if m.SenderID != "wxid_friend" || m.SenderName != "朋友" {
		t.Fatalf("real sender mapping lost: %+v", m)
	}
	if m.Direction != DirectionIn {
		t.Fatalf("direction = %q, want in", m.Direction)
	}
	if m.Text != "" || m.ParseStatus != ParsePartial {
		t.Fatalf("binary content should be partial and not text: %+v", m)
	}
	wantContent := base64.StdEncoding.EncodeToString([]byte{0xff, 0x00, 0xfe})
	if got := m.StructuredContent["raw_content_base64"]; got != wantContent {
		t.Fatalf("raw content = %v, want %s", got, wantContent)
	}
	wantPacked := base64.StdEncoding.EncodeToString([]byte{0x01, 0x02, 0x03})
	if got := m.StructuredContent["packed_info_data_base64"]; got != wantPacked {
		t.Fatalf("packed info = %v, want %s", got, wantPacked)
	}
	if got := m.StructuredContent["WCDB_CT_message_content"]; got != int64(1) {
		t.Fatalf("content type = %v", got)
	}
}

func TestMapRowToMessageCompressedContentIsNotPresentedAsPlaintext(t *testing.T) {
	ti := tableInfo{
		name: "Msg_compressed", colID: "local_id", colTime: "create_time",
		colType: "local_type", colContent: "message_content",
		colRealSender: "real_sender_id", colContentType: "wcdb_ct_message_content",
	}
	ctx := &parseContext{
		accountID: "wxid_me",
		contacts:  map[string]string{},
		convNames: map[string]string{},
		convKinds: map[string]string{},
	}
	m := mapRowToMessage(ti,
		[]string{"local_id", "create_time", "local_type", "message_content", "real_sender_id", "wcdb_ct_message_content"},
		[]any{int64(1), int64(1715000000), int64(1), "looks-like-text", "wxid_me", int64(4)},
		ctx,
	)
	if m.Direction != DirectionOut || m.Text != "" || m.ParseStatus != ParsePartial {
		t.Fatalf("compressed content was presented as decoded text: %+v", m)
	}
	if got, _ := m.StructuredContent["raw_content_base64"].(string); got == "" {
		t.Fatal("compressed raw bytes were not retained")
	}
	if !strings.Contains(strings.Join(m.Warnings, ";"), "compressed") {
		t.Fatalf("missing compressed warning: %v", m.Warnings)
	}
}

func TestNormalizeTypeUnpacksNewerPackedLocalType(t *testing.T) {
	if got := normalizeType(0x500000031); got != "app" {
		t.Fatalf("normalizeType(packed 49) = %q, want app", got)
	}
	if got := normalizeType(0x500000001); got != "text" {
		t.Fatalf("normalizeType(packed 1) = %q, want text", got)
	}
}

func TestNumericRealSenderDoesNotInventDirection(t *testing.T) {
	ti := tableInfo{
		name: "Msg_numeric_sender", colID: "local_id", colTime: "create_time",
		colType: "local_type", colContent: "message_content",
		colRealSender: "real_sender_id",
	}
	ctx := &parseContext{
		accountID: "wxid_me",
		contacts:  map[string]string{},
		convNames: map[string]string{},
		convKinds: map[string]string{},
	}
	m := mapRowToMessage(ti,
		[]string{"local_id", "create_time", "local_type", "message_content", "real_sender_id"},
		[]any{int64(1), int64(1715000000), int64(1), "hello", "79"},
		ctx,
	)
	if m.Direction != DirectionUnknown {
		t.Fatalf("numeric internal sender invented direction %q", m.Direction)
	}
	if m.SenderID != "79" || m.ParseStatus != ParsePartial {
		t.Fatalf("numeric sender was not retained honestly: %+v", m)
	}
}
