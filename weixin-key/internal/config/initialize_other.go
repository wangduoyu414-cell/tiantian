//go:build !windows

package config

import (
	"context"
	"errors"
)

func InitializeMetadataAtPath(ctx context.Context, path, accountRoot, account string) (MetadataInitProof, error) {
	if err := ctx.Err(); err != nil {
		return MetadataInitProof{}, err
	}
	return MetadataInitProof{}, errors.New("new-machine protected-cache initialization is currently Windows-only")
}
