//go:build !windows && !darwin

package main

import "errors"

// pickDirectory is unsupported on this platform; the path field stays manual.
func pickDirectory(initial string) (string, error) {
	return "", errors.New("folder picker not supported on this platform; type the path")
}
