package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Migrating a legacy schema-2 config to schema 4 must keep the original keys
// map intact, add typed key_entries, and preserve the pre-migration bytes in
// a one-time backup file.
func TestUpdateMigratesLegacySchemaWithBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	legacy := `{"schema_version":2,"wxid":"wxid_test","db_root":"D:\\data","keys":{"aabb":"ccdd"}}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WECHAT_CLI_CONFIG", path)

	if err := Update(func(c *Config) error {
		c.Wxid = "wxid_test" // no-op write; migration happens on persist
		return nil
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", got.SchemaVersion, CurrentSchemaVersion)
	}
	if got.Keys["aabb"] != "ccdd" {
		t.Fatalf("legacy keys map lost: %#v", got.Keys)
	}
	entry, ok := got.KeyEntries["aabb"]
	if !ok || entry.EncKey != "ccdd" || entry.Kind != "enc_key" {
		t.Fatalf("key_entries missing typed entry: %#v", got.KeyEntries)
	}

	backup, err := os.ReadFile(path + ".schema4-backup.protected")
	if err != nil {
		t.Fatalf("migration backup missing: %v", err)
	}
	var protected protectedValue
	if err := json.Unmarshal(backup, &protected); err != nil {
		t.Fatal(err)
	}
	restored, err := unprotectConfigBytes(&protected)
	if err != nil || string(restored) != legacy {
		t.Fatal("protected backup did not preserve exact legacy bytes")
	}
	clear(restored)

	// A second update must NOT overwrite the first backup.
	if err := Update(func(c *Config) error { return nil }); err != nil {
		t.Fatalf("second Update: %v", err)
	}
	backup2, err := os.ReadFile(path + ".schema4-backup.protected")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup2) != string(backup) {
		t.Fatal("backup was overwritten by a later migration")
	}
}

// SetVerifiedKey writes both views and bumps the schema.
func TestSetVerifiedKeyWritesBothViews(t *testing.T) {
	var c Config
	c.Wxid = "wxid_x"
	c.KDF = "pbkdf2-sha512-256000"
	c.SetVerifiedKey("aabb", "ccdd", "test-route", 123)
	if c.Keys["aabb"] != "ccdd" {
		t.Fatal("legacy keys map not updated")
	}
	e := c.KeyEntries["aabb"]
	if e.EncKey != "ccdd" || e.Source != "test-route" || e.WxID != "wxid_x" || e.KDF != "pbkdf2-sha512-256000" || e.VerifiedAt != 123 {
		t.Fatalf("entry = %#v", e)
	}
	if c.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("schema = %d, want %d", c.SchemaVersion, CurrentSchemaVersion)
	}
}
