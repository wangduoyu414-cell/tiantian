//go:build !windows

package main

import "gioui.org/widget/material"

// loadCJKFonts is a no-op outside Windows (system fonts are used as-is).
func loadCJKFonts(th *material.Theme) {}
