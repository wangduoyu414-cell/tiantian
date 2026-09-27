package wxkey

import (
	"errors"
	"sync"
)

// ErrCaptureCleanup records a cleanup failure even if a later retry completed.
// Consumers must not hide it behind a concurrent cancellation request.
var ErrCaptureCleanup = errors.New("capture cleanup encountered a failure")

// CaptureRecovery belongs to one setup job. A failed debugger cleanup is not
// cancellation completion: the capture call and its OS-thread owner remain
// alive until restoration and detach are confirmed. Retry requests never
// bypass safety checks and are bounded/coalesced, not a new capture job.
// Its zero value is ready for use; it must not be copied after first use.
type CaptureRecovery struct {
	mu      sync.Mutex
	waiting bool
	message string
	retry   chan struct{}
	notify  func(bool, string)
}

func NewCaptureRecovery(notify func(waiting bool, message string)) *CaptureRecovery {
	return &CaptureRecovery{retry: make(chan struct{}, 1), notify: notify}
}

func (r *CaptureRecovery) State() (waiting bool, message string) {
	if r == nil {
		return false, ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.waiting, r.message
}

func (r *CaptureRecovery) Retry() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.waiting {
		return false
	}
	select {
	case r.retry <- struct{}{}:
		return true
	default:
		return false
	}
}

func (r *CaptureRecovery) setWaiting(waiting bool, message string) {
	r.mu.Lock()
	r.initLocked()
	changed := r.waiting != waiting || r.message != message
	r.waiting, r.message = waiting, message
	if !waiting {
		select {
		case <-r.retry:
		default:
		}
	}
	notify := r.notify
	r.mu.Unlock()
	if changed && notify != nil {
		// A display/observer failure must never unwind the locked debugger
		// owner. The state remains queryable even if its notification panics.
		defer func() { _ = recover() }()
		notify(waiting, message)
	}
}

func (r *CaptureRecovery) initLocked() {
	if r.retry == nil {
		r.retry = make(chan struct{}, 1)
	}
}

func (r *CaptureRecovery) retryRequests() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.initLocked()
	return r.retry
}
