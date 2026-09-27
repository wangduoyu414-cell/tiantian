//go:build windows

package wxkey

// Route B — debugger-assisted transient-key capture (WeChat 4.1.10+).
//
// Legacy implementation, currently quarantined by capture_safety_windows.go.
// A string reference alone does NOT prove an instruction boundary, calling
// convention or live key. Do not remove the gate based on static string hits.
//
// This is intentionally opt-in (WECHAT_CLI_KEY_HOOK=1): attaching a debugger
// suspends WeChat's threads while stopped and writes debug-register state, so
// it is more invasive than passive scanning. It never modifies WeChat's code
// or on-disk files.

import (
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

const (
	dbgContinue             = 0x00010002
	dbgExceptionNotHandled  = 0x80010001
	exceptionSingleStep     = 0x80000004 // STATUS_SINGLE_STEP (hardware breakpoint / trap flag)
	exceptionBreakpoint     = 0x80000003 // STATUS_BREAKPOINT (debugger startup/DebugBreak)
	exceptionDebugEventCode = 1
	createThreadDebugEvent  = 2
	createProcessDebugEvent = 3
	loadDLLDebugEvent       = 6
	exitProcessDebugEvent   = 5
	infinite                = 0xFFFFFFFF
	contextAMD64            = 0x00100000
	contextControl          = contextAMD64 | 0x1
	contextDebugRegisters   = contextAMD64 | 0x10
	contextFull             = contextAMD64 | 0x1 | 0x2 | 0x8
	threadGetContext        = 0x0008
	threadSetContext        = 0x0010
	threadAllAccess         = 0x1F03FF
	processVMWrite          = 0x0020
	processVMOperation      = 0x0008
)

var (
	procDebugActiveProcess        = kernel32.NewProc("DebugActiveProcess")
	procDebugActiveProcessStop    = kernel32.NewProc("DebugActiveProcessStop")
	procWaitForDebugEvent         = kernel32.NewProc("WaitForDebugEvent")
	procContinueDebugEvent        = kernel32.NewProc("ContinueDebugEvent")
	procOpenThread                = kernel32.NewProc("OpenThread")
	procGetThreadContext          = kernel32.NewProc("GetThreadContext")
	procSetThreadContext          = kernel32.NewProc("SetThreadContext")
	procDebugSetProcessKillOnExit = kernel32.NewProc("DebugSetProcessKillOnExit")
)

// windowsHookScan attaches to the target process as a debugger, sets a
// hardware breakpoint on the code that references the `x'%s'` format string in
// Weixin.dll / WeChatWin.dll, and on each hit performs a fast targeted scan
// for the transient 64-hex raw key. Candidates are verified against the DBs.
func windowsHookScan(pid uint32, dbs []windowsSourceDB, salts map[string]bool, scan *setupScan, deadline time.Time) (err error) {
	if err := windowsActiveCapturePreflight("hook"); err != nil {
		return err
	}
	// Locate the WeChat core module range.
	modStart, modEnd := windowsFindWeChatCoreRange(pid)
	if modStart == 0 || modEnd == 0 {
		return fmt.Errorf("hook: WeChat core module (Weixin.dll/WeChatWin.dll) not found in pid %d", pid)
	}

	rh, _, openErr := procOpenProcess.Call(processVMRead|processQueryInformation|0x00100000, 0, uintptr(pid))
	if rh == 0 {
		return processAccessError(pid, processVMRead|processQueryInformation|0x00100000, openErr)
	}
	defer func() { err = errors.Join(err, (nativeDebugAPI{}).close(rh)) }()

	// Find the `x'%s'` format string, then a code site that references it.
	strAddr := windowsFindBytesInRange(rh, modStart, modEnd, []byte("x'%s'\x00"))
	if strAddr == 0 {
		// Fall back to matching without trailing NUL.
		strAddr = windowsFindBytesInRange(rh, modStart, modEnd, []byte("x'%s'"))
	}
	if strAddr == 0 {
		return fmt.Errorf("hook: x'%%s' format string not found in module range")
	}

	breakAddr := windowsFindCodeRefTo(rh, modStart, modEnd, strAddr)
	if breakAddr == 0 {
		return fmt.Errorf("hook: no code reference to x'%%s' string found")
	}

	// Debugger events must stay on the creating/attaching OS thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ready, err := prepareDebuggerOwner(scan.recovery)
	if err != nil {
		return err
	}
	if err := ready.check(); err != nil {
		return err
	}
	if r, _, err := procDebugActiveProcess.Call(uintptr(pid)); r == 0 {
		return fmt.Errorf("hook: DebugActiveProcess failed: %v", err)
	}
	session := newDebugSession(nativeDebugAPI{process: rh}, pid, breakAddr)
	if scan.recovery != nil {
		session.recovery = scan.recovery
	}
	defer func() {
		if !session.cleaned {
			err = errors.Join(err, session.finishCleanup())
		}
	}()
	scan.addDiag("RouteB: debugger attached to pid %d; breakpoint=0x%x", pid, breakAddr)
	return windowsCaptureDebugLoop(session, rh, dbs, salts, scan, deadline, nil)
}

// Both attach and launch use the same event and cleanup ownership model.
// The global safety gate remains mandatory: no approved module/material or
// startup-breakpoint profile exists. Survival preparation alone is not
// permission to activate capture.
func windowsCaptureDebugLoop(session *debugSession, rh uintptr, dbs []windowsSourceDB, salts map[string]bool,
	scan *setupScan, deadline time.Time, resolve func() (uintptr, error)) error {
	seen := map[string]bool{}
	enqueue, cancel, join := startCandidateVerifier(dbs, salts, scan, deadline)
	return runDebugSession(session, cancel, join, func() error {
		return session.pump(scan.ctx, deadline, func() bool { return scan.lenFound() == len(salts) }, resolve,
			func(tid uint32) {
				for _, key := range windowsCaptureKeyOnBreak(rh, tid, session.breakAddr, scan) {
					if !seen[key] && len(seen) < 1024 {
						seen[key] = true
						enqueue([]string{key})
					}
				}
			})
	})
}

// windowsCaptureKeyOnBreak is the quarantined legacy sampler. A format-string
// reference does not establish that cryptographic material is live. This
// register/stack hypothesis is NOT a supported capture ABI.
func windowsCaptureKeyOnBreak(rh uintptr, tid uint32, breakAddr uintptr, scan *setupScan) []string {
	var out []string
	seen := map[string]bool{}

	// Historical hypothesis: RAX may point at a 32-byte candidate and RSI
	// at a {data,length} container. Static field-access evidence alone does
	// not prove its material kind or identity in a live loaded module.
	th, _, _ := procOpenThread.Call(threadGetContext, 0, uintptr(tid))
	if th != 0 {
		ctx := newAlignedContext(contextFull | contextDebugRegisters)
		if getThreadContext(th, ctx) {
			rax := uintptr(ctxField(ctx, ctxOffRAX))
			rsi := uintptr(ctxField(ctx, ctxOffRSI))
			windowsAppendKeyAt(rh, rax, seen, &out)

			if rsi != 0 {
				var object [16]byte
				if windowsReadExact(rh, rsi, object[:]) {
					data := uintptr(binary.LittleEndian.Uint64(object[0:8]))
					length := binary.LittleEndian.Uint64(object[8:16])
					if length == 32 {
						windowsAppendKeyAt(rh, data, seen, &out)
					}
				}
			}
		}
		procCloseHandle.Call(th)
	}
	if len(out) > 0 {
		scan.addDiag("RouteB: captured %d direct register candidate(s)", len(out))
		return out
	}

	// Read the thread's RSP to locate its stack, then scan a window around it.
	rsp := windowsThreadRSP(tid)

	if rsp != 0 {
		// Scan a generous window of the stack in both directions.
		const stackWindow = 0x20000 // 128 KB
		start := rsp
		if start > stackWindow {
			start = rsp - stackWindow
		} else {
			start = 0
		}
		windowsScanWindowForKeys(rh, start, rsp+stackWindow, seen, &out)
	}

	return out
}

func windowsAppendKeyAt(rh, addr uintptr, seen map[string]bool, out *[]string) {
	if addr == 0 {
		return
	}
	key := make([]byte, 32)
	if !windowsReadExact(rh, addr, key) || !plausibleKeyBytes(key) {
		return
	}
	hexKey := fmt.Sprintf("%x", key)
	if !seen[hexKey] {
		seen[hexKey] = true
		*out = append(*out, hexKey)
	}
}

func windowsReadExact(rh, addr uintptr, dst []byte) bool {
	if len(dst) == 0 {
		return true
	}
	var got uintptr
	r, _, _ := procReadProcessMemory.Call(
		rh,
		addr,
		uintptr(unsafe.Pointer(&dst[0])),
		uintptr(len(dst)),
		uintptr(unsafe.Pointer(&got)),
	)
	return r != 0 && got == uintptr(len(dst))
}

// windowsScanWindowForKeys scans [start,end) for x'<64hex>' literals.
func windowsScanWindowForKeys(rh, start, end uintptr, seen map[string]bool, out *[]string) {
	const chunk = 1 << 20
	for addr := start; addr < end; {
		n := uintptr(chunk)
		if remain := end - addr; remain < n {
			n = remain
		}
		buf := make([]byte, n)
		var got uintptr
		r, _, _ := procReadProcessMemory.Call(rh, addr, uintptr(unsafe.Pointer(&buf[0])), n, uintptr(unsafe.Pointer(&got)))
		if r != 0 && got > 0 {
			scanRawKeyLiterals64(buf[:got], seen, out)
		}
		addr += n
	}
}

// windowsScanHeapForTransientKeys walks committed writable regions and looks
// for x'<64hex>' literals. Bounded by deadline-free quick pass; used only at
// the breakpoint instant.
func windowsScanHeapForTransientKeys(rh uintptr, seen map[string]bool, out *[]string) {
	const chunk = 4 << 20
	const maxUserAddress = uintptr(0x00007fffffffffff)
	scanned := 0
	const maxRegions = 4096 // safety cap
	for addr := uintptr(0); addr < maxUserAddress; {
		var m windowsMemoryBasicInformation
		r, _, _ := procVirtualQueryEx.Call(rh, addr, uintptr(unsafe.Pointer(&m)), unsafe.Sizeof(m))
		if r == 0 {
			addr += 0x10000
			continue
		}
		next := m.BaseAddress + m.RegionSize
		if next <= addr {
			return
		}
		if windowsReadableRegion(m) {
			var overlap []byte
			for off := uintptr(0); off < m.RegionSize; {
				n := uintptr(chunk)
				if remain := m.RegionSize - off; remain < n {
					n = remain
				}
				buf := make([]byte, n)
				var got uintptr
				rr, _, _ := procReadProcessMemory.Call(rh, m.BaseAddress+off, uintptr(unsafe.Pointer(&buf[0])), n, uintptr(unsafe.Pointer(&got)))
				if rr != 0 && got > 0 {
					data := append(append([]byte{}, overlap...), buf[:got]...)
					scanRawKeyLiterals64(data, seen, out)
					if len(data) > 128 {
						overlap = append(overlap[:0], data[len(data)-128:]...)
					} else {
						overlap = append(overlap[:0], data...)
					}
				}
				off += n
			}
			scanned++
			if scanned > maxRegions {
				return
			}
		}
		addr = next
	}
}

// ---- Debug-register / context helpers ----

// windowsThreadRSP returns the RSP register of the given thread, or 0.
func windowsThreadRSP(tid uint32) uintptr {
	th, _, _ := procOpenThread.Call(threadGetContext, 0, uintptr(tid))
	if th == 0 {
		return 0
	}
	defer procCloseHandle.Call(th)
	ctx := newAlignedContext(contextControl)
	if !getThreadContext(th, ctx) {
		return 0
	}
	return uintptr(ctxField(ctx, ctxOffRSP))
}

// ---- CONTEXT struct handling ----
//
// The x64 CONTEXT is large and alignment-sensitive (16-byte aligned). We keep a
// byte buffer and access the fields we need by offset. Offsets below are for
// the Windows x64 CONTEXT structure.

const (
	ctxBufSize   = 1232 // sizeof(CONTEXT) x64, rounded up
	ctxOffP1     = 0
	ctxOffFlags  = 0x30 // ContextFlags
	ctxOffEFlags = 0x44 // EFlags (the trap flag lives here)
	ctxOffDR0    = 0x48
	ctxOffDR7    = 0x70 // Dr7 control register; 0x68 is Dr6 (status) and does NOT arm anything
	ctxOffRAX    = 0x78
	ctxOffRSP    = 0x98
	ctxOffRSI    = 0xA8
)

// eflagsTrapFlag is the TF bit in EFlags: single-step one instruction.
const eflagsTrapFlag = 0x100

// alignedContext holds a CONTEXT byte buffer with a 16-byte-aligned view. We
// index the aligned sub-slice with binary.LittleEndian so we never keep a
// uintptr across a call boundary (which would be unsafe if the GC moved the
// backing array). The slice header keeps the memory pinned for the Syscall.
type alignedContext struct {
	raw []byte
	off int // offset within raw where the 16-byte-aligned CONTEXT begins
}

func newAlignedContext(flags uint32) *alignedContext {
	raw := make([]byte, ctxBufSize+16)
	base := uintptr(unsafe.Pointer(&raw[0]))
	aligned := (base + 15) &^ 15
	off := int(aligned - base)
	c := &alignedContext{raw: raw, off: off}
	setCtxField32(c, ctxOffFlags, flags)
	return c
}

// view returns the aligned CONTEXT sub-slice.
func (c *alignedContext) view() []byte { return c.raw[c.off:] }

// ptr returns a pointer to the first byte of the aligned CONTEXT. It must only
// be used as an argument to a syscall in the same statement.
func (c *alignedContext) basePtr() unsafe.Pointer {
	return unsafe.Pointer(&c.raw[c.off])
}

func ctxField(c *alignedContext, off uintptr) uint64 {
	return binary.LittleEndian.Uint64(c.view()[off : off+8])
}

func setCtxField(c *alignedContext, off uintptr, v uint64) {
	binary.LittleEndian.PutUint64(c.view()[off:off+8], v)
}

func setCtxField32(c *alignedContext, off uintptr, v uint32) {
	binary.LittleEndian.PutUint32(c.view()[off:off+4], v)
}

func ctxField32(c *alignedContext, off uintptr) uint32 {
	return binary.LittleEndian.Uint32(c.view()[off : off+4])
}

func getThreadContext(th uintptr, c *alignedContext) bool {
	r, _, _ := procGetThreadContext.Call(th, uintptr(c.basePtr()))
	return r != 0
}

func setThreadContext(th uintptr, c *alignedContext) bool {
	r, _, _ := procSetThreadContext.Call(th, uintptr(c.basePtr()))
	return r != 0
}

// ---- module + memory search helpers ----

// windowsFindWeChatCoreRange returns the memory range of the WeChat core module,
// trying Weixin.dll (4.1.x) first then WeChatWin.dll (older 4.x).
func windowsFindWeChatCoreRange(pid uint32) (start, end uintptr) {
	for _, name := range []string{"weixin.dll", "wechatwin.dll"} {
		if s, e := windowsFindModuleRange(pid, name); s != 0 && e != 0 {
			return s, e
		}
	}
	return 0, 0
}

// windowsFindModuleRange returns [base, base+size) for the named module.
func windowsFindModuleRange(pid uint32, wantLower string) (start, end uintptr) {
	snap, _, _ := procCreateToolhelp32.Call(th32csSnapModule, uintptr(pid))
	if snap == uintptr(syscall.InvalidHandle) || snap == 0 {
		return 0, 0
	}
	defer procCloseHandle.Call(snap)
	var entry windowsModuleEntry32W
	entry.Size = uint32(unsafe.Sizeof(entry))
	r, _, _ := procModule32FirstW.Call(snap, uintptr(unsafe.Pointer(&entry)))
	if r == 0 {
		return 0, 0
	}
	for {
		name := toLowerASCII(syscall.UTF16ToString(entry.ModuleName[:]))
		if name == wantLower {
			base := entry.ModBaseAddr
			return base, base + uintptr(entry.ModBaseSize)
		}
		entry.Size = uint32(unsafe.Sizeof(entry))
		r, _, _ = procModule32NextW.Call(snap, uintptr(unsafe.Pointer(&entry)))
		if r == 0 {
			break
		}
	}
	return 0, 0
}

// windowsFindBytesInRange scans [start,end) for the first occurrence of pat and
// returns its absolute address, or 0.
func windowsFindBytesInRange(rh, start, end uintptr, pat []byte) uintptr {
	const chunk = 4 << 20
	overlapKeep := len(pat) - 1
	if overlapKeep < 0 {
		overlapKeep = 0
	}
	for addr := start; addr < end; {
		var m windowsMemoryBasicInformation
		r, _, _ := procVirtualQueryEx.Call(rh, addr, uintptr(unsafe.Pointer(&m)), unsafe.Sizeof(m))
		if r == 0 {
			addr += 0x10000
			continue
		}
		next := m.BaseAddress + m.RegionSize
		if next <= addr {
			return 0
		}
		if windowsReadableRegion(m) {
			regionStart := m.BaseAddress
			regionEnd := next
			if regionStart < start {
				regionStart = start
			}
			if regionEnd > end {
				regionEnd = end
			}
			var prev []byte
			var prevAddr uintptr
			for off := regionStart; off < regionEnd; {
				n := uintptr(chunk)
				if remain := regionEnd - off; remain < n {
					n = remain
				}
				buf := make([]byte, n)
				var got uintptr
				rr, _, _ := procReadProcessMemory.Call(rh, off, uintptr(unsafe.Pointer(&buf[0])), n, uintptr(unsafe.Pointer(&got)))
				if rr != 0 && got > 0 {
					data := append(append([]byte{}, prev...), buf[:got]...)
					if idx := indexBytes(data, pat); idx >= 0 {
						if len(prev) > 0 {
							return prevAddr + uintptr(idx)
						}
						return off + uintptr(idx)
					}
					if len(buf[:got]) >= overlapKeep {
						prev = append(prev[:0], buf[got-uintptr(overlapKeep):got]...)
						prevAddr = off + got - uintptr(overlapKeep)
					}
				}
				off += n
			}
		}
		addr = next
	}
	return 0
}

// windowsFindCodeRefTo scans the executable region of [start,end) for an
// x64 RIP-relative LEA/MOV that references target, returning the instruction
// address. It matches the common `lea reg, [rip+disp32]` (48/4C 8D ...) and
// `mov reg, [rip+disp32]` (48/4C 8B ...) encodings.
func windowsFindCodeRefTo(rh, start, end, target uintptr) uintptr {
	const chunk = 4 << 20
	for addr := start; addr < end; {
		var m windowsMemoryBasicInformation
		r, _, _ := procVirtualQueryEx.Call(rh, addr, uintptr(unsafe.Pointer(&m)), unsafe.Sizeof(m))
		if r == 0 {
			addr += 0x10000
			continue
		}
		next := m.BaseAddress + m.RegionSize
		if next <= addr {
			return 0
		}
		// Only executable regions.
		if m.State == memCommit && m.Protect&pageGuard == 0 && windowsExecutable(m.Protect) {
			for off := uintptr(0); off < m.RegionSize; {
				n := uintptr(chunk)
				if remain := m.RegionSize - off; remain < n {
					n = remain
				}
				buf := make([]byte, n)
				var got uintptr
				rr, _, _ := procReadProcessMemory.Call(rh, m.BaseAddress+off, uintptr(unsafe.Pointer(&buf[0])), n, uintptr(unsafe.Pointer(&got)))
				if rr != 0 && got > 0 {
					data := buf[:got]
					base := m.BaseAddress + off
					// Look for RIP-relative refs: opcode(1-3) + disp32.
					// We scan for 0x8D (lea) / 0x8B (mov) with REX prefix and
					// verify the computed target.
					for i := 0; i+7 < len(data); i++ {
						b0 := data[i]
						if b0 != 0x48 && b0 != 0x4C && b0 != 0x49 && b0 != 0x4D {
							continue
						}
						op := data[i+1]
						if op != 0x8D && op != 0x8B {
							continue
						}
						modrm := data[i+2]
						// RIP-relative addressing: mod=00, rm=101
						if modrm&0xC7 != 0x05 {
							continue
						}
						disp := int32(binary.LittleEndian.Uint32(data[i+3 : i+7]))
						instrLen := uintptr(7)
						rip := base + uintptr(i) + instrLen
						computed := rip + uintptr(int64(disp))
						if computed == target {
							return base + uintptr(i)
						}
					}
				}
				off += n
			}
		}
		addr = next
	}
	return 0
}

func windowsExecutable(protect uint32) bool {
	// PAGE_EXECUTE(0x10), _READ(0x20), _READWRITE(0x40), _WRITECOPY(0x80)
	switch protect & 0xFF {
	case 0x10, 0x20, 0x40, 0x80:
		return true
	}
	return false
}

// ---- small byte utilities ----

func indexBytes(haystack, needle []byte) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func toLowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
