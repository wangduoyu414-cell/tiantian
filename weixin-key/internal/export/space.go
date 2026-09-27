package export

import (
	"errors"
	"fmt"
	"os"
)

// checkExportSpace accounts for the worst normal fallback, where an encrypted
// snapshot and its decrypted copy coexist, plus WAL bytes and a fixed margin.
// It is deliberately a lower-bound guard, not a claim that output growth is
// bounded by this value.
func checkExportSpace(outDir string, dbs []sourceDB) error {
	var source, wal int64
	for _, db := range dbs {
		fi, err := os.Stat(db.path)
		if err != nil {
			return fmt.Errorf("stat source %s: %w", db.rel, err)
		}
		if fi.Size() < 0 || source > (1<<62)-fi.Size() {
			return errors.New("source size overflow while calculating export budget")
		}
		source += fi.Size()
		if wi, err := os.Stat(db.path + "-wal"); err == nil {
			if wi.Size() < 0 || wal > (1<<62)-wi.Size() {
				return errors.New("WAL size overflow while calculating export budget")
			}
			wal += wi.Size()
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat WAL %s: %w", db.rel, err)
		}
	}
	if source > ((1<<62)-wal-exportStagingMargin)/2 {
		return errors.New("export storage budget overflow")
	}
	required := source*2 + wal + exportStagingMargin
	free, err := availableBytes(outDir)
	if err != nil {
		// A platform that cannot report free space must fail closed rather than
		// silently claiming the capacity check passed.
		return err
	}
	if free < required {
		return fmt.Errorf("insufficient free space for export staging: need at least %d bytes, have %d", required, free)
	}
	return nil
}
