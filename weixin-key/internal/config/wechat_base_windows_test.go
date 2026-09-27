//go:build windows

package config

import "testing"

func TestFixedDriveWeChatBases(t *testing.T) {
	drives := []string{`C:\`, `E:\`}
	fakeDirs := func(root string) []string {
		if root == `E:` {
			return []string{"软件", "xwechat_files", "temp"}
		}
		return nil
	}
	got := FixedDriveWeChatBases(drives, fakeDirs)
	want := map[string]bool{
		`C:\xwechat_files`:      true,
		`E:\xwechat_files`:      true,
		`E:\软件\xwechat_files`:   true,
		`E:\temp\xwechat_files`: true,
	}
	for _, b := range got {
		if !want[b] {
			t.Fatalf("unexpected base %q", b)
		}
		delete(want, b)
	}
	if len(want) != 0 {
		t.Fatalf("missing bases: %v", want)
	}
}
