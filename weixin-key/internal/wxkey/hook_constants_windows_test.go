//go:build windows

package wxkey

import "testing"

// The debugger route depends on exact Windows status codes and x64 CONTEXT
// offsets. STATUS_SINGLE_STEP is 0x80000004; 0xC0000004 is
// STATUS_INFO_LENGTH_MISMATCH (an error code, never a debug exception).
// Regression: the constant previously held 0xC0000004, which routed hardware
// breakpoint hits to DBG_EXCEPTION_NOT_HANDLED and would crash the target.
func TestDebugExceptionConstants(t *testing.T) {
	if exceptionSingleStep != 0x80000004 {
		t.Fatalf("exceptionSingleStep = 0x%08x, want STATUS_SINGLE_STEP 0x80000004", exceptionSingleStep)
	}
	if exceptionBreakpoint != 0x80000003 {
		t.Fatalf("exceptionBreakpoint = 0x%08x, want STATUS_BREAKPOINT 0x80000003", exceptionBreakpoint)
	}
	if dbgContinue != 0x00010002 {
		t.Fatalf("dbgContinue = 0x%08x, want DBG_CONTINUE 0x00010002", dbgContinue)
	}
	if dbgExceptionNotHandled != 0x80010001 {
		t.Fatalf("dbgExceptionNotHandled = 0x%08x, want 0x80010001", dbgExceptionNotHandled)
	}
}

// x64 CONTEXT field offsets are fixed by the Windows ABI. Dr7 (the debug
// control register that actually enables the DR0 hardware breakpoint) lives
// at 0x70; 0x68 is Dr6 (the status register). Regression: ctxOffDR7 was 0x68,
// so the DR0 breakpoint was never actually enabled.
func TestContextOffsetsMatchAMD64ABI(t *testing.T) {
	if ctxOffDR0 != 0x48 {
		t.Fatalf("ctxOffDR0 = 0x%x, want 0x48", ctxOffDR0)
	}
	if ctxOffDR7 != 0x70 {
		t.Fatalf("ctxOffDR7 = 0x%x, want 0x70 (0x68 is Dr6, the status register)", ctxOffDR7)
	}
	if ctxOffRAX != 0x78 || ctxOffRSP != 0x98 || ctxOffRSI != 0xA8 {
		t.Fatalf("GPR offsets wrong: RAX=0x%x RSP=0x%x RSI=0x%x", ctxOffRAX, ctxOffRSP, ctxOffRSI)
	}
	if ctxOffFlags != 0x30 {
		t.Fatalf("ctxOffFlags = 0x%x, want 0x30 (ContextFlags)", ctxOffFlags)
	}
}
