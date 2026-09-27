//go:build !windows

package main

import "weixin-key/internal/guicore"

func waitForRecoveryWithoutWindow(_ *guicore.Job, done <-chan struct{}) {
	// Windows debugger ownership is not used by these platform adapters.
	<-done
}
