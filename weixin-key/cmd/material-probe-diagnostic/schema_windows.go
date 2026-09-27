//go:build windows

package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

type messageColumnShape struct {
	Name string `json:"name"`
	Type string `json:"declared_type"`
}

// No source table names, row values, defaults, usernames or message bodies.
type messageSchemaShape struct {
	SHA256  string               `json:"sha256"`
	Tables  int                  `json:"tables"`
	Columns []messageColumnShape `json:"columns"`
}

func inspectMessageSchemaShapes(ctx context.Context, db *sql.DB, tables []string) ([]messageSchemaShape, error) {
	if len(tables) > 4096 {
		return nil, errors.New("message schema table budget exceeded")
	}
	byHash := map[string]*messageSchemaShape{}
	for _, name := range tables {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Include generated/hidden columns in identity and total budget.
		// Bind the table name as data; omit defaults, expressions and rows.
		rows, err := db.QueryContext(ctx, "SELECT name,type FROM pragma_table_xinfo(?) ORDER BY cid LIMIT 129", name)
		if err != nil {
			return nil, errors.New("message column metadata unavailable")
		}
		var columns []messageColumnShape
		for rows.Next() {
			var col messageColumnShape
			if err := rows.Scan(&col.Name, &col.Type); err != nil {
				rows.Close()
				return nil, errors.New("message column metadata unreadable")
			}
			if len(col.Name) > 256 || len(col.Type) > 128 || len(columns) == 128 {
				rows.Close()
				return nil, errors.New("message column metadata budget exceeded")
			}
			col.Type = strings.ToUpper(col.Type)
			columns = append(columns, col)
		}
		err = errors.Join(rows.Err(), rows.Close())
		if err != nil || len(columns) == 0 {
			return nil, errors.New("message column metadata incomplete")
		}
		raw, err := json.Marshal(columns)
		if err != nil {
			return nil, errors.New("message column metadata encoding failed")
		}
		digest := sha256.Sum256(raw)
		hash := hex.EncodeToString(digest[:])
		if found := byHash[hash]; found != nil {
			found.Tables++
		} else {
			if len(byHash) == 16 {
				return nil, errors.New("message schema shape budget exceeded")
			}
			byHash[hash] = &messageSchemaShape{SHA256: hash, Tables: 1, Columns: columns}
		}
	}
	out := make([]messageSchemaShape, 0, len(byHash))
	for _, shape := range byHash {
		out = append(out, *shape)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SHA256 < out[j].SHA256 })
	return out, nil
}
