//go:build windows

package wxkey

import (
	"errors"
	"fmt"
)

// ErrActiveCaptureUnavailable is a safety stop, NOT "no key found". Never
// recover from it by trying another active route or by reporting success.
var ErrActiveCaptureUnavailable = errors.New("主动采集未通过安全验收；未关闭、启动或附加微信")

// windowsActiveCapturePreflight quarantines the legacy active implementations.
// The independent review found un-restored DR/TF state, swallowed debugger
// errors, missing account binding, and unproven breakpoint semantics. The
// installed 4.1.13.65 module has a static candidate, not an approved profile.
//
// This is deliberately NOT a supported capture adapter or a completion claim.
// There is no environment/CLI override. Re-enabling needs implementation fixes,
// module-bound instruction/ABI evidence, synthetic failure tests and a new
// independent review. Consent to restart alone cannot waive these conditions.
func windowsActiveCapturePreflight(route string) error {
	return fmt.Errorf("%w（%s）：调试状态恢复及版本绑定待修复/复审；有效缓存与离线导入仍可使用", ErrActiveCaptureUnavailable, route)
}
