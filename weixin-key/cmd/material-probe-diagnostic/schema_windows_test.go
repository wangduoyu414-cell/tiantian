//go:build windows

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestMessageSchemaShapeRedactionAndBudgets(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	names := []string{"Msg_PRIVATE_TABLE_ID", "Msg_SECOND_PRIVATE_ID"}
	for _, name := range names {
		if _, err := db.Exec("CREATE TABLE " + name + "(local_id INTEGER, message_content TEXT DEFAULT 'PRIVATE_DEFAULT')"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec("INSERT INTO " + name + " VALUES(1,'PRIVATE_CHAT_BODY')"); err != nil {
			t.Fatal(err)
		}
	}
	got, err := inspectMessageSchemaShapes(context.Background(), db, names)
	if err != nil || len(got) != 1 || got[0].Tables != 2 || len(got[0].Columns) != 2 {
		t.Fatal("schema grouping failed", err)
	}
	raw, _ := json.Marshal(got)
	if strings.Contains(string(raw), "PRIVATE") || strings.Contains(string(raw), "Msg_") {
		t.Fatal("table identity/default/body leaked")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := inspectMessageSchemaShapes(ctx, db, names); err == nil {
		t.Fatal("schema cancellation ignored")
	}
	var many []string
	for i := 0; i < 129; i++ {
		many = append(many, fmt.Sprintf("c%d TEXT", i))
	}
	if _, err := db.Exec("CREATE TABLE Msg_wide(" + strings.Join(many, ",") + ")"); err != nil {
		t.Fatal(err)
	}
	if _, err := inspectMessageSchemaShapes(context.Background(), db, []string{"Msg_wide"}); err == nil {
		t.Fatal("column budget ignored")
	}
	var shapes []string
	for i := 0; i < 17; i++ {
		name := fmt.Sprintf("Msg_shape%d", i)
		if _, err := db.Exec(fmt.Sprintf("CREATE TABLE %s(c%d TEXT)", name, i)); err != nil {
			t.Fatal(err)
		}
		shapes = append(shapes, name)
	}
	if _, err := inspectMessageSchemaShapes(context.Background(), db, shapes); err == nil {
		t.Fatal("shape budget ignored")
	}
}

func TestMessageSchemaIncludesGeneratedColumnsInShapeAndBudget(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, stmt := range []string{
		"CREATE TABLE Msg_plain(id INTEGER)",
		"CREATE TABLE Msg_generated(id INTEGER, extra INTEGER GENERATED ALWAYS AS (id+1) VIRTUAL)",
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	got, err := inspectMessageSchemaShapes(context.Background(), db, []string{"Msg_plain", "Msg_generated"})
	if err != nil || len(got) != 2 {
		t.Error("generated column omitted from shape", err)
	} else if len(got[0].Columns)+len(got[1].Columns) != 3 {
		t.Error("generated column missing")
	}
	cols := []string{"id INTEGER"}
	for i := 0; i < 128; i++ {
		cols = append(cols, fmt.Sprintf("g%d INTEGER GENERATED ALWAYS AS (id+%d) VIRTUAL", i, i))
	}
	if _, err := db.Exec("CREATE TABLE Msg_generated_wide(" + strings.Join(cols, ",") + ")"); err != nil {
		t.Fatal(err)
	}
	if _, err := inspectMessageSchemaShapes(context.Background(), db, []string{"Msg_generated_wide"}); err == nil {
		t.Fatal("generated columns bypassed total column budget")
	}
}
