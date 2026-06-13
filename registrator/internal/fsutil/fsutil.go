// Package fsutil provides filesystem helpers shared across registrator internals.
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// AtomicWrite writes data to path atomically: it writes a temp file in the same
// directory, then renames it over path. A concurrent reader of path therefore
// always observes either the previous complete file or the new one, never a
// partially written file. The temp file is cleaned up if anything fails before
// the rename. path and the temp file must live on the same filesystem for the
// rename to be atomic (guaranteed here since both are in the same directory).
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("fsutil: create temp in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("fsutil: write %s: %w", tmpName, err)
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("fsutil: chmod %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("fsutil: close %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("fsutil: rename %s -> %s: %w", tmpName, path, err)
	}
	return nil
}
