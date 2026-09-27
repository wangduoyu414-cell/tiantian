//go:build windows

package wxkey

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// WeChatProcess describes one running Weixin.exe/WeChat.exe process.
type WeChatProcess struct {
	PID     uint32
	Name    string
	ExePath string
}

// FindWeChatProcesses lists running WeChat processes with their executable
// paths. Never touches process memory.
func FindWeChatProcesses() ([]WeChatProcess, error) {
	procs, err := windowsAllWeChatProcesses()
	if err != nil {
		return nil, err
	}
	out := make([]WeChatProcess, 0, len(procs))
	for _, p := range procs {
		path, err := queryProcessExecutable(p.pid)
		if err != nil {
			return nil, err
		}
		out = append(out, WeChatProcess{
			PID:     p.pid,
			Name:    p.exe,
			ExePath: path,
		})
	}
	return out, nil
}

// Selection overrides (PID/name) are useful for passive diagnostic routes, but
// cannot prove that OTHER instances are absent. Control/install discovery uses
// the actual complete set of known WeChat executable names, never that filter.
func windowsAllWeChatProcesses() ([]windowsProcess, error) {
	return allWeChatProcesses(windowsEnumerateProcesses)
}

func allWeChatProcesses(enumerate func() ([]windowsProcess, error)) ([]windowsProcess, error) {
	all, err := enumerate()
	if err != nil {
		return nil, err
	}
	var out []windowsProcess
	for _, p := range all {
		name := strings.ToLower(p.exe)
		if name == "weixin.exe" || name == "wechat.exe" {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].pid < out[j].pid })
	return out, nil
}

// RequestCloseWeChat is retained for source compatibility. A bare PID is not
// evidence of account ownership; only the account-scoped controller may close.
func RequestCloseWeChat(pid uint32) error {
	return fmt.Errorf("refusing close by bare PID %d; account-scoped approval and process identity are required", pid)
}

// WaitWeChatExit polls until no WeChat process remains or the timeout fires.
func WaitWeChatExit(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		procs, err := windowsAllWeChatProcesses()
		if err != nil {
			return err
		}
		if len(procs) == 0 {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("wechat processes still running after %s (user may have cancelled the close)", timeout)
}

// LaunchWeChat is retained for source compatibility. All start paths are
// quarantined until account-scoped capture readiness has passed review.
func LaunchWeChat(exePath string) (int, error) {
	return 0, windowsActiveCapturePreflight("legacy-launch")
}
