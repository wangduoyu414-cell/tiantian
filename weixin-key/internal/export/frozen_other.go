//go:build !windows

package export

import (
	"fmt"
	"os"
)

func openFrozenSource(path string) (*os.File, error) {
	return nil, fmt.Errorf("%w: this platform does not yet have a validated encrypted snapshot adapter", ErrLiveEncryptedSnapshot)
}
