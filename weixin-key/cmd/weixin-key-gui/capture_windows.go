//go:build windows

package main

import (
	"context"
	"fmt"
	"weixin-key/internal/wxkey"
)

// Called only after the job's restart approval gate. All input is job-local.
func captureFlow(ctx context.Context, opts wxkey.SetupOptions) error {
	res, diag, err := wxkey.RunSetupContext(ctx, opts)
	if err != nil {
		if diag != "" {
			return fmt.Errorf("%w；本次诊断：%s", err, diag)
		}
		return err
	}
	if opts.Progress != nil {
		opts.Progress(fmt.Sprintf("所选账号材料已验证（%d 个库）", len(res.Results)))
	}
	return nil
}
