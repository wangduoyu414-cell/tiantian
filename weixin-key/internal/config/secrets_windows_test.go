//go:build windows

package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestConfigSecretMaterialIsNotPlaintextOnDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("WECHAT_CLI_CONFIG", path)
	material := "synthetic-sensitive-value-not-a-real-key"
	c := &Config{Wxid: "synthetic", DBRoot: `D:\synthetic`, Keys: map[string]string{"salt": material}, Passphrase: material, KDF: "test", ImageKey: material, Key: material}
	if err := Save(c); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b, []byte(material)) {
		t.Fatal("secret material was written in plaintext")
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Keys["salt"] != material || got.Passphrase != material || got.ImageKey != material || got.Key != material {
		t.Fatal("protected round trip lost material")
	}
}

func TestConfigProtectionRejectsTamperingAndMixedFormats(t *testing.T) {
	raw, err := encodeConfig(&Config{Wxid: "synthetic", DBRoot: `D:\synthetic`, Passphrase: "synthetic-material"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		edit func(*diskConfig)
	}{
		{"account", func(d *diskConfig) { d.Wxid = "other" }},
		{"root", func(d *diskConfig) { d.DBRoot = `D:\other` }},
		{"ciphertext", func(d *diskConfig) { d.Protected.Ciphertext[len(d.Protected.Ciphertext)/2] ^= 1 }},
		{"scheme", func(d *diskConfig) { d.Protected.Scheme = "unknown" }},
		{"mixed", func(d *diskConfig) { d.Key = "plaintext-injection" }},
		{"future-schema", func(d *diskConfig) { d.SchemaVersion++ }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var d diskConfig
			if err := json.Unmarshal(raw, &d); err != nil {
				t.Fatal(err)
			}
			tc.edit(&d)
			changed, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decodeConfig(changed); err == nil {
				t.Fatal("modified protected configuration accepted")
			}
		})
	}
	if _, err := decodeConfig([]byte(`{"schema_version":5,"key":"plaintext"}`)); !errors.Is(err, ErrSecretStorage) {
		t.Fatalf("v5 plaintext: %v", err)
	}
}

func TestConfigAndMigrationBackupHaveCurrentUserOnlyDACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("WECHAT_CLI_CONFIG", path)
	if err := os.WriteFile(path, []byte(`{"schema_version":4,"keys":{"salt":"synthetic"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Update(func(*Config) error { return nil }); err != nil {
		t.Fatal(err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{path, path + ".schema4-backup.protected"} {
		f, err := os.Open(name)
		if err != nil {
			t.Fatal(err)
		}
		sd, err := windows.GetSecurityInfo(windows.Handle(f.Fd()), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		control, _, err := sd.Control()
		if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
			t.Fatalf("DACL inherits unexpected entries: %v", err)
		}
		// Inspect actual ACEs. Windows may add the harmless auto-inherited
		// control flag even on a protected DACL, so textual SDDL equality is
		// not an access check.
		acl, _, err := sd.DACL()
		if err != nil || acl == nil || acl.AceCount != 1 {
			t.Fatal("expected exactly one access-control entry")
		}
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(acl, 0, &ace); err != nil {
			t.Fatal(err)
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Header.AceFlags != 0 ||
			ace.Mask != windows.ACCESS_MASK(0x001F01FF) || !sid.Equals(user.User.Sid) { // FILE_ALL_ACCESS
			t.Fatalf("DACL not restricted to current user (file %s)", filepath.Base(name))
		}
	}
}

func TestConfigMigrationBackupFailurePreservesLegacy(t *testing.T) {
	for _, existing := range []string{"unreadable-backup", "directory", "protected-but-not-config"} {
		t.Run(existing, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			t.Setenv("WECHAT_CLI_CONFIG", path)
			raw := []byte(`{"schema_version":4,"keys":{"salt":"synthetic-original"}}`)
			if err := os.WriteFile(path, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			backup := path + ".schema4-backup.protected"
			if existing == "directory" {
				if err := os.Mkdir(backup, 0o700); err != nil {
					t.Fatal(err)
				}
			} else {
				b := []byte("invalid protected backup")
				if existing == "protected-but-not-config" {
					protected, err := protectConfigBytes([]byte("synthetic but not a configuration"))
					if err != nil {
						t.Fatal(err)
					}
					b, err = json.Marshal(protected)
					if err != nil {
						t.Fatal(err)
					}
				}
				if err := os.WriteFile(backup, b, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := Update(func(c *Config) error { c.Key = "replacement"; return nil }); err == nil {
				t.Fatal("migration succeeded without a usable recovery backup")
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(raw, after) {
				t.Fatal("failed migration changed the original file")
			}
		})
	}
}

func TestConfigAllSecretFieldsAndInputPreservation(t *testing.T) {
	xor := 173
	c := &Config{Wxid: "synthetic", ImageXORKey: &xor, Keys: map[string]string{"salt": "synthetic-key"}}
	raw, err := encodeConfig(c)
	if err != nil {
		t.Fatal(err)
	}
	if c.SchemaVersion != 0 || c.KeyEntries != nil || c.Keys["salt"] != "synthetic-key" {
		t.Fatal("serialization mutated caller state")
	}
	for _, field := range []string{`"keys":`, `"key_entries":`, `"image_xor_key":`} {
		if bytes.Contains(raw, []byte(field)) {
			t.Fatal("secret field in public JSON")
		}
	}
	got, err := decodeConfig(raw)
	if err != nil || got.ImageXORKey == nil || *got.ImageXORKey != xor || got.KeyEntries["salt"].EncKey != "synthetic-key" {
		t.Fatal("round-trip lost derived secret fields")
	}
	if _, err := readConfigBytes(strings.NewReader(strings.Repeat(" ", maxConfigBytes+1))); err == nil {
		t.Fatal("oversized input accepted")
	}
}

func TestConfigMigrationRejectsBackupLargerThanReaderLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("WECHAT_CLI_CONFIG", path)
	// Legacy unknown fields must survive in the recovery bytes. Base64 expands
	// an otherwise readable 13 MiB old file past the 16 MiB backup reader cap.
	raw := []byte(`{"schema_version":4,"keys":{"salt":"synthetic"},"padding":"` + strings.Repeat("x", 13<<20) + `"}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Update(func(*Config) error { return nil }); err == nil {
		t.Fatal("migration created a backup that cannot be read back")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(after, raw) {
		t.Fatal("oversized backup failure changed the original config")
	}
	if _, err := os.Stat(path + ".schema4-backup.protected"); !os.IsNotExist(err) {
		t.Fatal("oversized unusable backup was published")
	}
}

func TestConfigMigrationRejectsExistingBackupWithBroadDACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("WECHAT_CLI_CONFIG", path)
	raw := []byte(`{"schema_version":4,"keys":{"salt":"synthetic-original"}}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	sealed, err := protectConfigBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(sealed)
	if err != nil {
		t.Fatal(err)
	}
	backup := path + ".schema4-backup.protected"
	if err := os.WriteFile(backup, b, 0o600); err != nil {
		t.Fatal(err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + user.User.Sid.String() + ")(A;;GR;;;WD)")
	if err != nil {
		t.Fatal(err)
	}
	acl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	// Only this disposable synthetic file, never its directory or user files.
	if err := windows.SetNamedSecurityInfo(backup, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		t.Fatal(err)
	}
	if err := Update(func(*Config) error { return nil }); err == nil {
		t.Error("migration accepted a broadly readable backup")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(after, raw) {
		t.Fatal("bad backup permissions changed the legacy config")
	}
}
func TestConfigMigrationDoesNotCreatePlaintextSecretBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	t.Setenv("WECHAT_CLI_CONFIG", path)
	material := "synthetic-old-material"
	raw := []byte(`{"schema_version":4,"wxid":"synthetic","keys":{"salt":"` + material + `"}}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Update(func(*Config) error { return nil }); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(b, []byte(material)) {
			t.Fatalf("plaintext secret remains in newly written migration artifact %s", e.Name())
		}
	}
}
func TestUnreadableProtectedConfigIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("WECHAT_CLI_CONFIG", path)
	if err := os.WriteFile(path, []byte(`{"schema_version":5,"wxid":"synthetic","protected_secrets":{"scheme":"dpapi-user-v1","ciphertext":"AQID"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("unreadable protected config was treated as missing material")
	}
}
