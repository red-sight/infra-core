package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "krakend.json")

	if err := AtomicWrite(path, []byte("first"), 0644); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if got := readFile(t, path); got != "first" {
		t.Fatalf("content = %q, want %q", got, "first")
	}
	if mode := statMode(t, path); mode != 0644 {
		t.Fatalf("mode = %o, want 0644", mode)
	}

	// Overwriting replaces content atomically and leaves no temp files behind.
	if err := AtomicWrite(path, []byte("second"), 0600); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	if got := readFile(t, path); got != "second" {
		t.Fatalf("content after overwrite = %q, want %q", got, "second")
	}
	if mode := statMode(t, path); mode != 0600 {
		t.Fatalf("mode after overwrite = %o, want 0600", mode)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected only the target file, found %d entries: %v", len(entries), entries)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func statMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Mode().Perm()
}
