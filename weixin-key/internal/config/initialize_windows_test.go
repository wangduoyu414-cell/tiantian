//go:build windows

package config

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func metadataInitFixture(t *testing.T) (string, string) {
	t.Helper()
	source := filepath.Join(t.TempDir(), "wxid_synthetic_0001")
	if err := os.MkdirAll(filepath.Join(source, "db_storage"), 0o700); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(t.TempDir(), "config.json"), source
}

func assertNoInitTemps(t *testing.T, path string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".init-") {
			t.Fatal("initialization left a temporary file")
		}
	}
}

func TestInitializeMetadataExplicitAndProtectedCacheCompatible(t *testing.T) {
	path, source := metadataInitFixture(t)
	poison := filepath.Join(t.TempDir(), "must-not-create.json")
	t.Setenv("WECHAT_CLI_CONFIG", poison)
	t.Setenv("WX_MCP_CONFIG", poison)
	t.Setenv("WECHAT_CLI_DB_ROOT", `C:\unrelated-account`)
	t.Setenv("WECHAT_CLI_PASSPHRASE_HEX", strings.Repeat("ab", 32))
	t.Setenv("WECHAT_CLI_IMAGE_KEY", strings.Repeat("cd", 16))
	t.Setenv("WX_MCP_IMAGE_KEY", strings.Repeat("ef", 16))
	proof, err := InitializeMetadataAtPath(context.Background(), path, source, "wxid_synthetic")
	if err != nil || !proof.Applied || !proof.MetadataVerified || !proof.PrivateFile || len(proof.SHA256) != 64 {
		t.Fatal("metadata initialization failed", proof, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := decodeConfig(raw)
	if err != nil || hasSecrets(cfg) || cfg.DBRoot != source || cfg.Wxid != "wxid_synthetic" {
		t.Fatal("metadata contents invalid or environment material imported", err)
	}
	if _, err := os.Stat(poison); !os.IsNotExist(err) {
		t.Fatal("environment-selected config was touched")
	}
	again, err := InitializeMetadataAtPath(context.Background(), path, source, "different")
	after, readErr := os.ReadFile(path)
	if err == nil || again.Applied || readErr != nil || !bytes.Equal(raw, after) {
		t.Fatal("initialization overwrote an existing file")
	}
	// The existing reviewed cache writer must accept the fresh metadata. This
	// uses only synthetic material; it is not a live key acquisition test.
	update, err := UpdateProtectedAtPath(context.Background(), path, proof.SHA256, addCheckedSyntheticKey)
	if err != nil || !update.Applied || !update.ProtectedReadback || !update.BackupVerified {
		t.Fatal("fresh metadata did not enter existing protected writer", err)
	}
	if _, err := ReadProtectedAtPath(path, update.SHA256); err != nil {
		t.Fatal("fresh-machine protected cache cannot be read", err)
	}
	assertNoInitTemps(t, path)
}

func TestInitializeMetadataRejectsRemoteVolumesWithoutConnecting(t *testing.T) {
	for _, path := range []string{`Z:\cache`, `\\server\share\cache`, `\\?\C:\cache`, `C:\cache:stream`} {
		if localInitializationVolume(path, func(*uint16) uint32 { return 4 }) {
			t.Fatal("remote/device/stream path accepted")
		}
	}
	for _, kind := range []uint32{0, 1, 4, 5, 6} {
		if localInitializationVolume(`Z:\cache`, func(*uint16) uint32 { return kind }) {
			t.Fatal("nonlocal drive type accepted")
		}
	}
	if !localInitializationVolume(`C:\cache`, func(*uint16) uint32 { return 3 }) {
		t.Fatal("fixed local volume rejected")
	}
}

func TestInitializeMetadataRejectsIntermediateDirectoryLink(t *testing.T) {
	for _, child := range []string{"", "real-child"} {
		t.Run(map[string]string{"": "direct-parent", "real-child": "ancestor"}[child], func(t *testing.T) {
			path, source := metadataInitFixture(t)
			realParent := filepath.Join(filepath.Dir(path), child)
			if err := os.MkdirAll(realParent, 0o700); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(t.TempDir(), "alias")
			if err := os.Symlink(filepath.Dir(path), link); err != nil {
				t.Skipf("cannot create synthetic directory link: %v", err)
			}
			proof, err := InitializeMetadataAtPath(context.Background(), filepath.Join(link, child, "config.json"), source, "wxid_synthetic")
			if err == nil || proof.Applied {
				t.Fatal("directory link was followed by writer")
			}
			if entries, err := os.ReadDir(realParent); err != nil || len(entries) != 0 {
				t.Fatal("reparse preflight created files")
			}
		})
	}
}

func TestInitializationPinsDenyAncestorWriteAndDeleteHandles(t *testing.T) {
	base := t.TempDir()
	parent := filepath.Join(base, "ancestor")
	leaf := filepath.Join(parent, "middle", "leaf")
	if err := os.MkdirAll(leaf, 0o700); err != nil {
		t.Fatal(err)
	}
	pinned, err := openInitializationDirectory(leaf)
	if err != nil {
		t.Fatal(err)
	}
	defer pinned.close()
	name, err := windows.UTF16PtrFromString(parent)
	if err != nil {
		t.Fatal(err)
	}
	for _, access := range []uint32{windows.FILE_WRITE_DATA, windows.DELETE} {
		h, err := windows.CreateFile(name, access,
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
			nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
		if err == nil {
			_ = windows.CloseHandle(h)
			t.Errorf("ancestor pin admitted conflicting access %#x", access)
		} else if !errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
			t.Fatalf("expected actual sharing conflict, got %v", err)
		}
	}
}

func TestInitializeMetadataPostCommitFailuresPreserveApplied(t *testing.T) {
	for _, kind := range []string{"missing-readback", "different-readback", "readback-error"} {
		t.Run(kind, func(t *testing.T) {
			path, source := metadataInitFixture(t)
			proof, err := initializeMetadataAtPath(context.Background(), path, source, "wxid_synthetic",
				initializationHooks{beforeReadback: func(root *os.Root, base string) error {
					switch kind {
					case "missing-readback":
						return root.Remove(base)
					case "different-readback":
						f, err := root.OpenFile(base, os.O_WRONLY|os.O_TRUNC, 0o600)
						if err != nil {
							return err
						}
						_, writeErr := f.WriteString("synthetic corruption")
						return errors.Join(writeErr, f.Close())
					default:
						return errors.New("synthetic readback error")
					}
				}})
			if err == nil || !proof.Applied || proof.MetadataVerified {
				t.Fatal("post-commit failure lost state", kind, proof, err)
			}
			assertNoInitTemps(t, path)
		})
	}
}

func TestInitializeMetadataRejectsInvalidSelectionWithoutWriting(t *testing.T) {
	for _, kind := range []string{"relative-config", "relative-root", "empty-account", "account-path",
		"account-control", "missing-parent", "missing-storage", "inside-source", "alternate-stream", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			path, source := metadataInitFixture(t)
			account := "wxid_synthetic"
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			originalPath := path
			switch kind {
			case "relative-config":
				path = "relative-config.json"
			case "relative-root":
				source = "relative-account"
			case "empty-account":
				account = ""
			case "account-path":
				account = "../account"
			case "account-control":
				account = "account\n"
			case "missing-parent":
				path = filepath.Join(filepath.Dir(path), "absent", "config.json")
			case "missing-storage":
				source = t.TempDir()
			case "inside-source":
				path = filepath.Join(source, "config.json")
			case "alternate-stream":
				path += ":stream"
			case "cancelled":
				cancel()
			}
			proof, err := InitializeMetadataAtPath(ctx, path, source, account)
			if err == nil || proof.Applied {
				t.Fatal("invalid initialization accepted")
			}
			entries, err := os.ReadDir(filepath.Dir(originalPath))
			if err != nil || len(entries) != 0 {
				t.Fatal("failed preflight wrote configuration artifacts")
			}
			if filepath.IsAbs(path) {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatal("invalid target was created", err)
				}
			}
		})
	}
}

func TestInitializeMetadataBusyAndCompetingPublication(t *testing.T) {
	t.Run("busy", func(t *testing.T) {
		path, source := metadataInitFixture(t)
		err := withConfigWriteLock(path, func(*os.Root, string) error {
			proof, err := InitializeMetadataAtPath(context.Background(), path, source, "wxid_synthetic")
			if !errors.Is(err, ErrConfigBusy) || proof.Applied {
				t.Fatal("busy writer was not rejected immediately", err)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("competing-file", func(t *testing.T) {
		path, source := metadataInitFixture(t)
		competitor := []byte("existing file must survive")
		proof, err := initializeMetadataAtPath(context.Background(), path, source, "wxid_synthetic",
			initializationHooks{create: func(root *os.Root, base string) (*os.File, error) {
				if err := os.WriteFile(path, competitor, 0o600); err != nil {
					return nil, err
				}
				return root.OpenFile(base, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
			}})
		raw, readErr := os.ReadFile(path)
		if err == nil || proof.Applied || readErr != nil || !bytes.Equal(raw, competitor) {
			t.Fatal("racing target was overwritten")
		}
		assertNoInitTemps(t, path)
	})
}

func TestInitializeMetadataCancellationAfterCommitRetainsProof(t *testing.T) {
	path, source := metadataInitFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	proof, err := initializeMetadataAtPath(ctx, path, source, "wxid_synthetic",
		initializationHooks{beforeReadback: func(root *os.Root, base string) error {
			cancel()
			return nil
		}})
	if !errors.Is(err, context.Canceled) || !proof.Applied || !proof.MetadataVerified || !proof.PrivateFile {
		t.Fatal("post-commit cancellation lost publication/readback evidence", err)
	}
	assertNoInitTemps(t, path)
}

func TestInitializeMetadataPinsParentOutsideSource(t *testing.T) {
	path, source := metadataInitFixture(t)
	moved := filepath.Join(source, "moved-cache")
	proof, err := initializeMetadataAtPath(context.Background(), path, source, "wxid_synthetic",
		initializationHooks{beforeReadback: func(root *os.Root, base string) error {
			// Both absolute directories belong exclusively to this synthetic
			// fixture. No user directory is moved or removed by this test.
			if err := os.Rename(filepath.Dir(path), moved); err == nil {
				t.Error("configuration parent could move into source during publication")
			}
			return nil
		}})
	if err != nil || !proof.MetadataVerified {
		t.Fatal("pinned publication failed", err)
	}
	if _, err := os.Stat(moved); !os.IsNotExist(err) {
		t.Error("source acquired configuration files")
	}
}

func TestInitializeMetadataCancelledAfterCreationLeavesExplicitIncompleteState(t *testing.T) {
	path, source := metadataInitFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	proof, err := initializeMetadataAtPath(ctx, path, source, "wxid_synthetic",
		initializationHooks{create: func(root *os.Root, base string) (*os.File, error) {
			f, err := root.OpenFile(base, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
			cancel()
			return f, err
		}})
	if !errors.Is(err, context.Canceled) || !proof.Applied || !proof.PrivateFile || proof.MetadataVerified {
		t.Fatal("incomplete metadata state was hidden", proof, err)
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil || len(raw) != 0 {
		t.Fatal("expected an explicitly incomplete empty file", readErr)
	}
	if _, err := InitializeMetadataAtPath(context.Background(), path, source, "wxid_synthetic"); err == nil {
		t.Fatal("incomplete bootstrap silently overwritten")
	}
}
