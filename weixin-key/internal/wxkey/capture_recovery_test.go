package wxkey

import "testing"

func TestCaptureRecoveryZeroValueAndBoundedRetry(t *testing.T) {
	var r CaptureRecovery
	if r.Retry() {
		t.Fatal("idle controller accepted retry")
	}
	r.setWaiting(true, "fixture")
	if !r.Retry() || r.Retry() {
		t.Fatal("retry requests were not bounded/coalesced")
	}
	<-r.retryRequests()
	if !r.Retry() {
		t.Fatal("consumed retry was not released")
	}
	r.setWaiting(false, "")
	select {
	case <-r.retryRequests():
		t.Fatal("stale retry leaked into the next waiting period")
	default:
	}
}

func TestCaptureRecoveryObserverPanicCannotUnwindOwner(t *testing.T) {
	r := NewCaptureRecovery(func(bool, string) { panic("synthetic display failure") })
	r.setWaiting(true, "fixture")
	if waiting, msg := r.State(); !waiting || msg != "fixture" || !r.Retry() {
		t.Fatal("observer failure destroyed query/retry state")
	}
	r.setWaiting(false, "")
}
