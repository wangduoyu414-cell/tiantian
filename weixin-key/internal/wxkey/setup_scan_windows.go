//go:build windows

package wxkey

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// setupScan is the per-job result collector for one setup run. Every capture
// route writes verified material here; nothing reaches the config file until
// the single-writer commit at the end of runSetup. It replaces the bare
// shared "found" map so that per-salt provenance and any discovered account
// passphrase survive to persistence.
type setupScan struct {
	mu          sync.Mutex // guards every field; Route D verifies candidates in parallel
	ctx         context.Context
	recovery    *CaptureRecovery
	diagnostics []string

	found            map[string]string // salt hex -> verified per-DB enc_key hex
	foundSource      map[string]string // salt hex -> route/source label
	passphraseHex    string            // verified account passphrase, if any route found one
	passphraseSource string
}

// lenFound returns the number of resolved salts (concurrency-safe).
func (s *setupScan) lenFound() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.found)
}

// hasSalt reports whether salt is already resolved.
func (s *setupScan) hasSalt(salt string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.found[salt]
	return ok
}

func newSetupScan() *setupScan {
	return &setupScan{ctx: context.Background(), found: map[string]string{}, foundSource: map[string]string{}}
}

func (s *setupScan) addDiag(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.diagnostics) < 256 {
		s.diagnostics = append(s.diagnostics, fmt.Sprintf(format, args...))
	}
}

func (s *setupScan) diagString() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Join(s.diagnostics, " | ")
}

func (s *setupScan) stopped(deadline time.Time) bool {
	return s.ctx.Err() != nil || windowsKeyScanDeadlineExceeded(deadline)
}

func (s *setupScan) wait(d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-timer.C:
		return nil
	}
}

// noteVerified records one typed verification outcome.
func (s *setupScan) noteVerified(v KeyCandidateVerification) {
	if !v.Match {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.found[v.SaltHex]; !ok {
		s.found[v.SaltHex] = v.EncKeyHex
		src := v.Source
		if src == "" {
			src = string(v.Kind)
		}
		s.foundSource[v.SaltHex] = src
	}
	if v.Kind == MaterialPassphrase && s.passphraseHex == "" {
		s.passphraseHex = v.PassphraseHex
		s.passphraseSource = v.Source
	}
}

// noteRawKey records material that is already known to be a raw enc_key
// (e.g. an x"<key><salt>" literal, which is bound to its salt by shape).
func (s *setupScan) noteRawKey(saltHex, encKeyHex, source string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.found[saltHex]; ok {
		return
	}
	s.found[saltHex] = encKeyHex
	s.foundSource[saltHex] = source
}

// notePassphrase records a verified account passphrase once.
func (s *setupScan) notePassphrase(passphraseHex, source string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.passphraseHex == "" {
		s.passphraseHex = passphraseHex
		s.passphraseSource = source
	}
}
