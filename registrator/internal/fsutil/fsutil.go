// Package fsutil provides filesystem helpers shared across registrator internals.
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AtomicWrite writes data to path atomically (see AtomicWriteFunc with a nil
// validate callback).
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	return AtomicWriteFunc(path, data, perm, nil)
}

// AtomicWriteFunc writes data to a temp file in the same directory as path,
// optionally runs validate against that temp file, and only then renames it over
// path. A concurrent reader of path therefore always observes either the previous
// complete file or the new one, never a partial write — and never a file that
// failed validation. If validate returns an error, path is left untouched and the
// temp file removed. The temp file keeps path's extension so tools that select a
// parser by extension (e.g. KrakenD: .json) behave identically on it. path and the
// temp file share a directory, so the rename is atomic.
func AtomicWriteFunc(path string, data []byte, perm os.FileMode, validate func(tempPath string) error) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)

	tmp, err := os.CreateTemp(dir, "."+stem+".tmp-*"+ext)
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

	if validate != nil {
		if err := validate(tmpName); err != nil {
			return fmt.Errorf("fsutil: validation failed for %s: %w", path, err)
		}
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("fsutil: rename %s -> %s: %w", tmpName, path, err)
	}
	return nil
}
