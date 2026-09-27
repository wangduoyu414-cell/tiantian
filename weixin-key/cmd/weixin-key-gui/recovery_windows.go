//go:build windows

package main

import (
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"weixin-key/internal/guicore"
)

// This fallback is only for a failed Gio window, never for normal close.
// No automatic write retries: each MessageBox acknowledgement requests one.
func waitForRecoveryWithoutWindow(job *guicore.Job, done <-chan struct{}) {
	messageBox := windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW")
	title, _ := windows.UTF16PtrFromString("微信导出 — 安全恢复")
	for {
		select {
		case <-done:
			return
		case <-time.After(250 * time.Millisecond):
		}
		stage, events, _, _, _ := job.State()
		if stage != guicore.StageRecovering {
			continue
		}
		text := "显示窗口发生错误，但采集恢复仍未完成，工具不会直接退出。\n请勿强制结束工具。\n点击“确定”只请求重试恢复，不会重新采集。"
		if len(events) > 0 {
			text += "\n\n" + events[len(events)-1].Message
		}
		body, _ := windows.UTF16PtrFromString(text)
		// MB_OK | MB_ICONWARNING. No misleading cancel/force-exit option.
		result, _, _ := messageBox.Call(0, uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(title)), 0x30)
		if result == 1 { // IDOK
			job.RetryRecovery()
		} else {
			// No repeated popups after a native dialog failure. State remains owned;
			// target exit can still complete cleanup. No force-exit path.
			<-done
			return
		}
	}
}
