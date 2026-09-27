package export

import (
	"fmt"
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// Markdown rendering (CommonMark-compatible, UTF-8)
// ---------------------------------------------------------------------------

// fenceRun matches runs of 3+ backticks or tildes.
var fenceRun = regexp.MustCompile("(`{3,}|~{3,})")

// escapeBodyFence returns the body ready for inclusion in the document. When
// the body itself contains a code-fence run, the whole body is wrapped in a
// fence strictly longer than the longest run inside it, so embedded fences
// can never break the document structure.
func escapeBodyFence(body string) string {
	longest := 0
	for _, m := range fenceRun.FindAllString(body, -1) {
		if len(m) > longest {
			longest = len(m)
		}
	}
	if longest == 0 {
		return body
	}
	fence := strings.Repeat("~", longest+1)
	return fence + "\n" + body + "\n" + fence
}

// renderMessageMarkdown renders one message:
//
//	#### 2026-09-11 20:00:00 +08:00 · 张三 · 收到 · 文本
//
//	正文……
//
//	<!-- meta: id=svr:123 db=message/msg_0.db table=Msg_x rowid=42 type=1 status=parsed -->
func renderMessageMarkdown(m *Message) string {
	var b strings.Builder
	ts := m.Timestamp
	if ts == "" {
		ts = "（时间未知）"
	}
	sender := m.SenderName
	if sender == "" {
		sender = m.SenderID
	}
	if sender == "" {
		sender = "（发送者未知）"
	}
	dir := directionLabel(m.Direction)
	typ := typeLabel(m.Type, m.SourceType)

	fmt.Fprintf(&b, "#### %s · %s · %s · %s\n\n", ts, sender, dir, typ)

	body := m.Text
	if body == "" && m.StructuredContent != nil {
		if raw, ok := m.StructuredContent["raw_content"].(string); ok && raw != "" {
			body = raw
		}
	}
	if body != "" {
		b.WriteString(escapeBodyFence(body))
		b.WriteString("\n")
	} else if _, compressed := m.StructuredContent["raw_content_base64"]; compressed {
		b.WriteString("（正文为压缩或二进制内容，未进行未经验证的解码；原始字节见同 ID 的 JSONL 记录）\n")
	} else if m.Type != "text" {
		fmt.Fprintf(&b, "（%s消息：内容未提取，详见同 ID 的 JSONL 记录）\n", typeLabel(m.Type, m.SourceType))
	}
	for _, a := range m.Attachments {
		if a.Ref != "" {
			fmt.Fprintf(&b, "\n- 附件：%s（%s）：`%s`\n", a.Kind, a.Status, a.Ref)
		} else {
			fmt.Fprintf(&b, "\n- 附件：%s（%s）\n", a.Kind, a.Status)
		}
	}

	fmt.Fprintf(&b, "\n<!-- meta: id=%s source=%s table=%s rowid=%d type=%d status=%s",
		m.MessageID, m.Provenance.DB, m.Provenance.Table, m.Provenance.RowID, m.SourceType, m.ParseStatus)
	for _, w := range m.Warnings {
		fmt.Fprintf(&b, " warn=%q", w)
	}
	b.WriteString(" -->\n\n")
	return b.String()
}

func directionLabel(d string) string {
	switch d {
	case DirectionOut:
		return "发出"
	case DirectionIn:
		return "收到"
	case DirectionSystem:
		return "系统"
	default:
		return "方向未知"
	}
}

func typeLabel(t string, src int64) string {
	switch t {
	case "text":
		return "文本"
	case "image":
		return "图片"
	case "voice":
		return "语音"
	case "video":
		return "视频"
	case "emoji":
		return "表情"
	case "app":
		return "文件/链接"
	case "system":
		return "系统"
	default:
		return fmt.Sprintf("未知类型(%d)", src)
	}
}

// ---------------------------------------------------------------------------
// Filesystem-safe names
// ---------------------------------------------------------------------------

// windowsReserved are device names Windows refuses as file names.
var windowsReserved = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true, "com5": true,
	"com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true, "lpt5": true,
	"lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

// sanitizeFileComponent maps an arbitrary ID to a stable, portable file
// component: [A-Za-z0-9._-] kept, everything else becomes '_'; Windows
// reserved names and over-long/empty results are normalized. The mapping is
// deterministic so reruns produce identical names.
func sanitizeFileComponent(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	out = strings.Trim(out, ".")
	if out == "" {
		out = "unnamed"
	}
	if windowsReserved[strings.ToLower(out)] {
		out = "_" + out
	}
	if len(out) > 80 {
		// Keep the tail identifiable: prefix + hash of the full string.
		out = out[:64] + "-" + shortHash(s)
	}
	return out
}

func shortHash(s string) string {
	// FNV-1a 32-bit, hex. Stable across runs; used only for collision-safe
	// file naming, not for message identity.
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return fmt.Sprintf("%08x", h)
}
