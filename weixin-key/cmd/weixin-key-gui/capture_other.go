//go:build !windows

package main

import (
	"context"
	"errors"
	"weixin-key/internal/wxkey"
)

func captureFlow(ctx context.Context, opts wxkey.SetupOptions) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if opts.Progress != nil {
		opts.Progress("本平台暂不支持自动采集；请提供合法授权材料")
	}
	return errors.New("capture not implemented on this platform")
}
