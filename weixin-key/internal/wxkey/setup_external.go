//go:build !windows

package wxkey

import (
	"context"
	"errors"
)

func defaultSetupOptions() SetupOptions { return SetupOptions{} }

func runSetup() (*SetupResult, string, error) {
	return runSetupExternal()
}

func runSetupContext(ctx context.Context, opts SetupOptions) (*SetupResult, string, error) {
	if opts.Restart || opts.DBRoot != "" {
		return nil, "", errors.New("explicit-account capture adapter is not implemented on this platform")
	}
	return runSetupExternalContext(ctx)
}
