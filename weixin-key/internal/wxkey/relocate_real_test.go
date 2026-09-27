package wxkey

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRelocateAgainstRealWeixinDLL runs relocation against a live Weixin.dll if
// one is installed, and asserts the MMV1 / x'%s' / Config.Cipher anchors each
// resolve to a plausible function entry. Skips when WeChat is not installed.
func TestRelocateAgainstRealWeixinDLL(t *testing.T) {
	base := `C:\Program Files\Tencent\Weixin`
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Skip("WeChat not installed at default path")
	}
	var dll string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		cand := filepath.Join(base, e.Name(), "Weixin.dll")
		if _, err := os.Stat(cand); err == nil {
			dll = cand // keep latest matching version dir
		}
	}
	if dll == "" {
		t.Skip("no versioned Weixin.dll found")
	}
	t.Logf("analyzing %s", dll)

	rep, err := RelocateHookPoints(dll, nil)
	if err != nil {
		t.Fatalf("RelocateHookPoints: %v", err)
	}
	t.Logf("image: base=0x%x sizeofimage=0x%x anchors=%v", rep.ImageBase, rep.SizeOfImage, rep.AnchorHits)

	// Every string anchor must appear; each must yield at least one function entry.
	for _, name := range []string{"MMV1", "x'%s'", "Config.Cipher"} {
		if rep.AnchorHits[name] == 0 {
			t.Errorf("anchor %q not found in image", name)
			continue
		}
		found := false
		for _, c := range rep.Candidates {
			if c.Anchor == name && c.FuncEntryRVA != 0 {
				found = true
				t.Logf("  %s: stringRVA=0x%x leaRVA=0x%x funcEntry=0x%x", name, c.StringRVA, c.LeaRVA, c.FuncEntryRVA)
			}
		}
		if !found {
			t.Errorf("anchor %q found but produced no function entry", name)
		}
	}
}
