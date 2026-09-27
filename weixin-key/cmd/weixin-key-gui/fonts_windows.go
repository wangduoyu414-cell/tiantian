//go:build windows

package main

import (
	"os"

	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/text"
	"gioui.org/widget/material"
)

// loadCJKFonts adds Microsoft YaHei (and fallbacks) so Chinese UI text
// renders instead of tofu boxes. Best effort: gofont stays as fallback.
func loadCJKFonts(th *material.Theme) {
	collection := gofont.Collection()
	for _, path := range []string{
		`C:\Windows\Fonts\msyh.ttc`,
		`C:\Windows\Fonts\simhei.ttf`,
		`C:\Windows\Fonts\simsun.ttc`,
	} {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		faces, err := opentype.ParseCollection(b)
		if err != nil || len(faces) == 0 {
			continue
		}
		collection = append(collection, faces...)
		break
	}
	th.Shaper = text.NewShaper(text.WithCollection(collection))
}
