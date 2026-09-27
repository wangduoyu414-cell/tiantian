// imgkey-probe locates the account-global WeChat v4 image AES-128 key in a
// running WeChat process by read-only memory scanning.
//
// The key is loaded lazily by WeChat: it is only resident after the user has
// viewed chat images in this session. A miss means "not loaded yet", not
// "does not exist" — ask the user to open a few images and rerun.
//
// Read-only: no writes to the target process, no debug attach, no files
// created. The recovered key is printed to stdout exactly once; treat it as
// a secret (prefer consuming it via WECHAT_CLI_IMGKEY_HEX).
package main

import (
	"bytes"
	"crypto/aes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32           = windows.NewLazySystemDLL("kernel32.dll")
	procVirtualQueryEx = kernel32.NewProc("VirtualQueryEx")
	procReadProcMem    = kernel32.NewProc("ReadProcessMemory")
)

type memBasicInfo struct {
	BaseAddress       uintptr
	AllocationBase    uintptr
	AllocationProtect uint32
	PartitionID       uint16
	RegionSize        uintptr
	State             uint32
	Protect           uint32
	Type              uint32
}

const (
	memCommit  = 0x1000
	memPrivate = 0x20000
	pageRW     = 0x04
	maxRegion  = 256 << 20
)

var (
	jpgSOI  = []byte{0xFF, 0xD8, 0xFF}
	wxgfSig = []byte("wxgf")
)

// jpegPlausible requires the JPEG marker structure a real decoder accepts:
// SOI plus a JFIF/Exif/DQT/DHT marker within the first blocks.
func jpegPlausible(b []byte) bool {
	if !bytes.HasPrefix(b, jpgSOI) {
		return false
	}
	for _, off := range []int{2, 16, 18, 20, 32, 34, 36} {
		if off+1 < len(b) && b[off] == 0xFF {
			switch b[off+1] {
			case 0xE0, 0xE1, 0xDB, 0xC4:
				return true
			}
		}
	}
	return false
}

func usage() {
	fmt.Fprintln(os.Stderr, `imgkey-probe — locate the WeChat v4 image AES key in a running process

  imgkey-probe --pid <n> [--pid <n>...] --dat <any-v4-dat-file> [--pretty]

--dat must be a WeChat v4 .dat image (magic 07 08 56 32); its AES region
serves as the validation anchor. Any thumb (_t.dat) from the account works.
Exit: 0 = key found, 1 = not resident, 2 = usage error.`)
}

func main() {
	var pids []uint32
	var datPath string
	pretty := false
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--pid":
			i++
			if i >= len(args) {
				usage()
				os.Exit(2)
			}
			v, err := strconv.ParseUint(args[i], 10, 32)
			if err != nil || v == 0 {
				usage()
				os.Exit(2)
			}
			pids = append(pids, uint32(v))
		case "--dat":
			i++
			if i >= len(args) {
				usage()
				os.Exit(2)
			}
			datPath = args[i]
		case "--pretty":
			pretty = true
		default:
			usage()
			os.Exit(2)
		}
	}
	if len(pids) == 0 || datPath == "" {
		usage()
		os.Exit(2)
	}

	dat, err := os.ReadFile(datPath)
	if err != nil || len(dat) < 79 || !bytes.HasPrefix(dat, []byte{0x07, 0x08, 0x56, 0x32}) {
		fmt.Fprintln(os.Stderr, "imgkey-probe: --dat is not a readable v4 .dat file")
		os.Exit(2)
	}
	testBlocks := dat[15:79] // 4 AES blocks

	type result struct {
		Found bool   `json:"found"`
		Key   string `json:"key_hex,omitempty"`
		PID   uint32 `json:"pid,omitempty"`
		Note  string `json:"note,omitempty"`
	}
	printRes := func(r result) {
		var b []byte
		if pretty {
			b, _ = json.MarshalIndent(r, "", "  ")
		} else {
			b, _ = json.Marshal(r)
		}
		fmt.Println(string(b))
	}

	for _, pid := range pids {
		key, found := scanProcess(pid, testBlocks)
		if found {
			printRes(result{Found: true, Key: key, PID: pid})
			return
		}
	}
	printRes(result{Found: false, Note: "image key not resident; ask the user to open a few chat images in WeChat, then rerun"})
	os.Exit(1)
}

func scanProcess(pid uint32, testBlocks []byte) (string, bool) {
	h, err := windows.OpenProcess(windows.PROCESS_VM_READ|windows.PROCESS_QUERY_INFORMATION, false, pid)
	if err != nil {
		return "", false
	}
	defer windows.CloseHandle(h)

	var addr uintptr
	for addr < 0x7FFFFFFFFFFF {
		var m memBasicInfo
		r, _, _ := procVirtualQueryEx.Call(uintptr(h), addr, uintptr(unsafe.Pointer(&m)), unsafe.Sizeof(m))
		if r == 0 {
			break
		}
		next := m.BaseAddress + m.RegionSize
		if next <= addr {
			break
		}
		addr = next
		if m.State != memCommit || m.Type != memPrivate || m.Protect != pageRW || m.RegionSize > maxRegion || m.RegionSize < 4096 {
			continue
		}
		buf := make([]byte, m.RegionSize)
		var got uintptr
		r2, _, _ := procReadProcMem.Call(uintptr(h), m.BaseAddress, uintptr(unsafe.Pointer(&buf[0])), m.RegionSize, uintptr(unsafe.Pointer(&got)))
		if r2 == 0 || got < 32 {
			continue
		}
		buf = buf[:got]
		for off := 0; off+16 <= len(buf); off += 4 {
			c, err := aes.NewCipher(buf[off : off+16])
			if err != nil {
				continue
			}
			var first [16]byte
			c.Decrypt(first[:], testBlocks[:16])
			if !bytes.HasPrefix(first[:], jpgSOI) && !bytes.HasPrefix(first[:], wxgfSig) {
				continue
			}
			full := make([]byte, len(testBlocks))
			for i := 0; i+16 <= len(testBlocks); i += 16 {
				c.Decrypt(full[i:i+16], testBlocks[i:i+16])
			}
			if bytes.HasPrefix(full, wxgfSig) || jpegPlausible(full) {
				return hex.EncodeToString(buf[off : off+16]), true
			}
		}
	}
	return "", false
}
