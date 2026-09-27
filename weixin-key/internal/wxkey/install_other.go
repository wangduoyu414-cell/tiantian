//go:build !windows

package wxkey

import "errors"

func defaultWeixinDLLPath() (string, error) {
	return "", errors.New("automatic Weixin.dll discovery is Windows-only; supply --dll explicitly")
}
