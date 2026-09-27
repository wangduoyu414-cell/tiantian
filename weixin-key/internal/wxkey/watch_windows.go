//go:build windows

package wxkey

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"time"
	"unsafe"
)

// Login/DB-open readiness watch: after a WeChat relaunch, key material only
// exists in memory once the account's databases are actually OPEN (their
// SQLCipher cipher contexts hold the DB salt). Process existence, windows, or
// old DB files are NOT evidence of that. This watcher polls all WeChat
// processes (re-enumerated each cycle, so freshly spawned children are
// covered) for any target DB salt appearing in writable memory.
//
// The watch has its own budget (default 10 min, WECHAT_CLI_LOGIN_TIMEOUT)
// independent of the scan budget: QR-code login time must not eat compute
// time.

// WindowsWaitForDBOpen polls until any of saltsHex appears in a WeChat
// process's writable memory. Returns that process's PID. Errors on timeout
// with an explicit "not logged in" message.
func WindowsWaitForDBOpen(saltsHex []string, timeout, interval time.Duration, progress func(string)) (uint32, error) {
	var salts [][]byte
	for _, s := range saltsHex {
		b, err := hex.DecodeString(s)
		if err != nil || len(b) != 16 {
			continue
		}
		salts = append(salts, b)
	}
	if len(salts) == 0 {
		return 0, fmt.Errorf("no usable salts to watch")
	}
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	if interval <= 0 {
		interval = 3 * time.Second
	}
	deadline := time.Now().Add(timeout)
	cycles := 0
	for {
		cycles++
		procs, err := windowsTargetProcesses()
		if err == nil {
			for _, p := range procs {
				if windowsProcessContainsAny(p.pid, salts, 8*time.Second) {
					return p.pid, nil
				}
			}
		}
		if time.Now().After(deadline) {
			return 0, fmt.Errorf("等待超时：%s 内未检测到任何目标数据库在微信中打开（请确认已完成登录）", timeout)
		}
		if progress != nil {
			progress(fmt.Sprintf("等待登录与数据库打开…（已等待 %ds，覆盖 %d 个进程）", cycles*int(interval/time.Second), len(procs)))
		}
		time.Sleep(interval)
	}
}

// windowsProcessContainsAny reports whether any needle appears in the
// process's committed writable memory. Bounded by a per-process time budget
// so one huge process cannot starve the watch cycle.
func windowsProcessContainsAny(pid uint32, needles [][]byte, budget time.Duration) bool {
	h, _, _ := procOpenProcess.Call(processVMRead|processQueryInformation, 0, uintptr(pid))
	if h == 0 {
		return false
	}
	defer procCloseHandle.Call(h)

	deadline := time.Now().Add(budget)
	const chunkSize = 4 << 20
	const maxUserAddress = uintptr(0x00007fffffffffff)
	for addr := uintptr(0); addr < maxUserAddress; {
		if time.Now().After(deadline) {
			return false
		}
		var m windowsMemoryBasicInformation
		r, _, _ := procVirtualQueryEx.Call(h, addr, uintptr(unsafe.Pointer(&m)), unsafe.Sizeof(m))
		if r == 0 {
			addr += 0x10000
			continue
		}
		next := m.BaseAddress + m.RegionSize
		if next <= addr {
			break
		}
		if windowsWritableHeapRegion(m) {
			for off := uintptr(0); off < m.RegionSize; {
				n := uintptr(chunkSize)
				if remain := m.RegionSize - off; remain < n {
					n = remain
				}
				buf := make([]byte, n)
				var got uintptr
				rr, _, _ := procReadProcessMemory.Call(h, m.BaseAddress+off, uintptr(unsafe.Pointer(&buf[0])), n, uintptr(unsafe.Pointer(&got)))
				if rr != 0 && got > 0 {
					for _, needle := range needles {
						if bytes.Contains(buf[:got], needle) {
							return true
						}
					}
				}
				off += n
			}
		}
		addr = next
	}
	return false
}
