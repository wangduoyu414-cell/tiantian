//go:build windows

package wxkey

import (
	"context"
	"time"
)

// One bounded verifier works outside the debugger thread. The event pump
// never blocks on KDF or file IO, including while handling re-arm events.
func startCandidateVerifier(dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) (func([]string), func(), func()) {
	ctx, cancel := context.WithCancel(scan.ctx)
	queue := make(chan string, 32)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case key := <-queue:
				if ctx.Err() != nil || scan.stopped(deadline) {
					return
				}
				if err := windowsVerifyCandidateKeys([]string{key}, dbs, salts, scan, deadline); err != nil {
					scan.addDiag("candidate validation stopped: %v", err)
					return
				}
			}
		}
	}()
	return func(keys []string) {
		for _, key := range keys {
			if ctx.Err() != nil {
				return
			}
			select {
			case queue <- key:
			default:
				scan.addDiag("candidate queue full; sample dropped")
				return
			}
		}
	}, cancel, func() { <-done }
}
