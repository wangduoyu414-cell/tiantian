//go:build !windows

package config

func coordinateConfigWrites(wait bool, fn func() error) error { return fn() }
