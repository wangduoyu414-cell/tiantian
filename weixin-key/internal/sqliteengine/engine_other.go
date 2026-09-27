//go:build !windows || !amd64

package sqliteengine

import (
	"context"
	"errors"
)

const Version = "unavailable-on-this-platform"

var ErrUnavailable = errors.New("validated encrypted snapshot engine unavailable on this platform")

func Backup(ctx context.Context, source, destination, keyHex, saltHex string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrUnavailable
}
