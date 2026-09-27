package pathguard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOutputSourceAndNonexistentSuffix(t *testing.T) {
	source := t.TempDir()
	for _, out := range []string{source, filepath.Join(source, "new", "deeper")} {
		if _, err := ResolveOutput(out, source); err == nil {
			t.Fatal("output inside source accepted")
		}
	}
	out := filepath.Join(t.TempDir(), "not", "created")
	got, err := ResolveOutput(out, source)
	if err != nil || got != out {
		t.Fatalf("separate output: %s %v", got, err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("validation wrote to disk")
	}
}
func TestOutputIntermediateLinkIntoSource(t *testing.T) {
	source, parent := t.TempDir(), t.TempDir()
	link := filepath.Join(parent, "link")
	if err := os.Symlink(source, link); err != nil {
		t.Skipf("OS disallows creating a synthetic directory link: %v", err)
	}
	if _, err := ResolveOutput(filepath.Join(link, "not-yet-created", "export"), source); err == nil {
		t.Fatal("intermediate link into source accepted")
	}
}
