//go:build windows

package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const semanticRowBudget int64 = 1_000_000

type messageSemanticReport struct {
	RowsScanned             int64            `json:"rows_scanned"`
	RowsWithContent         int64            `json:"rows_with_content"`
	ContentNull             int64            `json:"content_null"`
	ContentText             int64            `json:"content_text"`
	ContentBlob             int64            `json:"content_blob"`
	ContentOther            int64            `json:"content_other"`
	RealSenderEmpty         int64            `json:"real_sender_empty"`
	RealSenderNonEmpty      int64            `json:"real_sender_nonempty"`
	PackedInfoNonEmpty      int64            `json:"packed_info_nonempty"`
	TimeSeconds             int64            `json:"time_seconds"`
	TimeMilliseconds        int64            `json:"time_milliseconds"`
	TimeUnknown             int64            `json:"time_unknown"`
	ContentTypeMarkers      map[string]int64 `json:"content_type_markers,omitempty"`
	SourceTypeMarkers       map[string]int64 `json:"source_type_markers,omitempty"`
	CompressionFlags        map[string]int64 `json:"compression_flags,omitempty"`
	LocalTypeMarkers        map[string]int64 `json:"local_type_markers,omitempty"`
	RealSenderNumeric       map[string]int64 `json:"real_sender_numeric,omitempty"`
	ShapeTablesWithoutProbe int              `json:"shape_tables_without_probe,omitempty"`
}

func inspectMessageSemantics(ctx context.Context, db *sql.DB, tables []string) (messageSemanticReport, error) {
	out := messageSemanticReport{
		ContentTypeMarkers: make(map[string]int64),
		SourceTypeMarkers:  make(map[string]int64),
		CompressionFlags:   make(map[string]int64),
		LocalTypeMarkers:   make(map[string]int64),
		RealSenderNumeric:  make(map[string]int64),
	}
	for _, table := range tables {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		columns, err := messageColumnsForSemantics(ctx, db, table)
		if err != nil {
			return out, err
		}
		content, ok := columns["message_content"]
		if !ok {
			out.ShapeTablesWithoutProbe++
			continue
		}
		selected := []string{quoteSemanticIdent(content)}
		names := []string{"content"}
		for _, spec := range []struct {
			key string
			out string
		}{
			{"real_sender_id", "sender"},
			{"packed_info_data", "packed"},
			{"wcdb_ct_message_content", "content_type"},
			{"wcdb_ct_source", "source_type"},
			{"compress_content", "compress"},
			{"create_time", "create_time"},
			{"local_type", "local_type"},
		} {
			if c, found := columns[spec.key]; found {
				selected = append(selected, quoteSemanticIdent(c))
				names = append(names, spec.out)
			}
		}
		query := "SELECT " + strings.Join(selected, ",") + " FROM " + quoteSemanticIdent(table)
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			return out, fmt.Errorf("semantic query unavailable: %w", err)
		}
		for rows.Next() {
			if out.RowsScanned >= semanticRowBudget {
				rows.Close()
				return out, errors.New("message semantic row budget exceeded")
			}
			values := make([]any, len(names))
			ptrs := make([]any, len(names))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				rows.Close()
				return out, errors.New("message semantic row unreadable")
			}
			out.RowsScanned++
			if values[0] == nil {
				out.ContentNull++
			} else {
				out.RowsWithContent++
				switch values[0].(type) {
				case string:
					out.ContentText++
				case []byte:
					out.ContentBlob++
				default:
					out.ContentOther++
				}
			}
			for i := 1; i < len(names); i++ {
				switch names[i] {
				case "sender":
					if semanticStringEmpty(values[i]) {
						out.RealSenderEmpty++
					} else {
						out.RealSenderNonEmpty++
						if n, ok := semanticInt(values[i]); ok {
							incrementMarker(out.RealSenderNumeric, fmt.Sprintf("%d", n))
						}
					}
				case "packed":
					if !semanticStringEmpty(values[i]) {
						out.PackedInfoNonEmpty++
					}
				case "content_type":
					incrementMarker(out.ContentTypeMarkers, values[i])
				case "source_type":
					incrementMarker(out.SourceTypeMarkers, values[i])
				case "compress":
					incrementMarker(out.CompressionFlags, values[i])
				case "create_time":
					switch semanticTimeUnit(values[i]) {
					case "s":
						out.TimeSeconds++
					case "ms":
						out.TimeMilliseconds++
					default:
						out.TimeUnknown++
					}
				case "local_type":
					incrementMarker(out.LocalTypeMarkers, values[i])
				}
			}
		}
		if err := errors.Join(rows.Err(), rows.Close()); err != nil {
			return out, errors.New("message semantic query failed")
		}
	}
	return out, nil
}

func messageColumnsForSemantics(ctx context.Context, db *sql.DB, table string) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, "SELECT name FROM pragma_table_xinfo(?) ORDER BY cid", table)
	if err != nil {
		return nil, errors.New("message semantic column metadata unavailable")
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, errors.New("message semantic column metadata unreadable")
		}
		out[strings.ToLower(name)] = name
	}
	if err := rows.Err(); err != nil {
		return nil, errors.New("message semantic column metadata failed")
	}
	return out, nil
}

func quoteSemanticIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func semanticStringEmpty(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(t) == ""
	case []byte:
		return len(t) == 0 || strings.TrimSpace(string(t)) == ""
	default:
		return false
	}
}

func incrementMarker(dst map[string]int64, v any) {
	key := "unknown"
	switch t := v.(type) {
	case nil:
		key = "null"
	case int64:
		key = fmt.Sprintf("%d", t)
	case int:
		key = fmt.Sprintf("%d", t)
	case float64:
		key = fmt.Sprintf("%g", t)
	case []byte:
		key = "blob"
	case string:
		key = "text"
	default:
		key = "other"
	}
	if len(dst) >= 64 {
		if _, ok := dst[key]; !ok {
			key = "other"
		}
	}
	dst[key]++
}

func semanticTimeUnit(v any) string {
	n, ok := semanticInt(v)
	if !ok {
		return "unknown"
	}
	if n <= 0 {
		return "unknown"
	}
	if n > 1e12 {
		return "ms"
	}
	return "s"
}

func semanticInt(v any) (int64, bool) {
	var n int64
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case []byte:
		_, err := fmt.Sscanf(string(t), "%d", &n)
		return n, err == nil
	case string:
		_, err := fmt.Sscanf(t, "%d", &n)
		return n, err == nil
	default:
		return 0, false
	}
}
