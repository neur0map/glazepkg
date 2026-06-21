package snapshot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.json")

	if err := writeFileAtomic(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(p); string(got) != "hello" {
		t.Errorf("first write = %q, want hello", got)
	}

	if err := writeFileAtomic(p, []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(p); string(got) != "world" {
		t.Errorf("overwrite = %q, want world", got)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected exactly 1 file, got %d (leftover temp?)", len(entries))
	}
}
