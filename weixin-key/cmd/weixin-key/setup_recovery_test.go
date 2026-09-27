package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"weixin-key/internal/wxkey"
)

func TestSetupInterruptWaitsForCleanupAndPreservesFailure(t *testing.T) {
	signals := make(chan os.Signal, 2)
	started, cancelled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	defer close(release)
	done := make(chan error, 1)
	var stderr bytes.Buffer
	go func() {
		_, _, err := setupWithSignals(signals, &stderr, func(ctx context.Context, r *wxkey.CaptureRecovery) (*wxkey.SetupResult, string, error) {
			if r == nil {
				return nil, "", errors.New("missing recovery controller")
			}
			close(started)
			<-ctx.Done()
			close(cancelled)
			<-release
			return nil, "", errors.Join(ctx.Err(), wxkey.ErrCaptureCleanup)
		})
		done <- err
	}()
	<-started
	signals <- os.Interrupt
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("interrupt did not request cancellation")
	}
	select {
	case <-done:
		t.Fatal("setup returned before cleanup finished")
	default:
	}
	release <- struct{}{}
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || !errors.Is(err, wxkey.ErrCaptureCleanup) {
			t.Fatalf("cleanup failure lost: %v", err)
		}
		if !strings.Contains(stderr.String(), "安全清理") {
			t.Fatal("cancellation was not explained on stderr")
		}
	case <-time.After(time.Second):
		t.Fatal("setup did not return after cleanup")
	}
}
