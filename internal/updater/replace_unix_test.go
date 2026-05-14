//go:build !windows

package updater

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceBinary_SameFilesystemFastPath(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dest := filepath.Join(dir, "dest")
	if err := os.WriteFile(src, []byte("new"), 0o755); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatalf("write dest: %v", err)
	}

	if err := replaceBinary(src, dest); err != nil {
		t.Fatalf("replaceBinary: %v", err)
	}

	body, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(body) != "new" {
		t.Errorf("dest = %q, want %q", body, "new")
	}
	if _, err := os.Stat(src); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("src should be gone after rename, got err=%v", err)
	}
}

func TestReplaceBinary_CrossFilesystemUsesRenameAside(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dest := filepath.Join(dir, "dest")
	if err := os.WriteFile(src, []byte("new"), 0o755); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatalf("write dest: %v", err)
	}

	// Force the initial rename to fail, mimicking EXDEV when src and dest
	// are on different filesystems. The aside path should still succeed
	// because moveFile's internal copy creates a fresh dest after the .old
	// rename has freed the path.
	saved := renameFile
	renameFile = func(s, d string) error {
		if s == src && d == dest {
			return errors.New("simulated cross-filesystem failure")
		}
		return os.Rename(s, d)
	}
	t.Cleanup(func() { renameFile = saved })

	if err := replaceBinary(src, dest); err != nil {
		t.Fatalf("replaceBinary: %v", err)
	}

	body, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(body) != "new" {
		t.Errorf("dest = %q, want %q", body, "new")
	}
	if _, err := os.Stat(dest + ".old"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("dest.old should be removed after success, got err=%v", err)
	}
}

func TestReplaceBinary_CrossFilesystemRollsBackOnCopyFailure(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dest := filepath.Join(dir, "dest")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatalf("write dest: %v", err)
	}
	// src deliberately does not exist; moveFile inside replaceBinary will
	// fail trying to open it for the copy fallback, and the rollback should
	// restore dest from the .old we renamed aside.

	saved := renameFile
	renameFile = func(s, d string) error {
		if s == src && d == dest {
			return errors.New("simulated cross-filesystem failure")
		}
		return os.Rename(s, d)
	}
	t.Cleanup(func() { renameFile = saved })

	err := replaceBinary(src, dest)
	if err == nil {
		t.Fatal("expected an error when src is missing")
	}

	body, readErr := os.ReadFile(dest)
	if readErr != nil {
		t.Fatalf("dest should have been rolled back to its original content, but read failed: %v", readErr)
	}
	if string(body) != "old" {
		t.Errorf("dest should still contain old content after rollback, got %q", body)
	}
}
