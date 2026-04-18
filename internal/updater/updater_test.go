package updater

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWrapReplaceErr_NonPermissionErrorPassesThrough(t *testing.T) {
	got := wrapReplaceErr(errors.New("disk full"), "/tmp/gpk")
	if got == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(got.Error(), "cannot replace binary") {
		t.Errorf("expected 'cannot replace binary' wrapper, got %q", got)
	}
	if !strings.Contains(got.Error(), "disk full") {
		t.Errorf("expected original cause in chain, got %q", got)
	}
}

func TestWrapReplaceErr_PermissionIsDetectedOnEveryPlatform(t *testing.T) {
	// os.PathError{ Err: fs.ErrPermission } is the shape os.OpenFile
	// returns on every OS; this is the exact failure mode that used to
	// slip past the old string match on Windows.
	pathErr := &fs.PathError{Op: "open", Path: "/fake", Err: fs.ErrPermission}
	got := wrapReplaceErr(pathErr, `C:\Program Files\gpk\gpk.exe`)
	if got == nil {
		t.Fatal("expected a wrapped permission error")
	}
	msg := got.Error()
	if runtime.GOOS == "windows" {
		if !strings.Contains(msg, "installer") {
			t.Errorf("windows branch should point at the installer, got %q", msg)
		}
		if strings.Contains(msg, "sudo") {
			t.Errorf("windows branch should not suggest sudo, got %q", msg)
		}
	} else {
		if !strings.Contains(msg, "sudo gpk update") {
			t.Errorf("unix branch should suggest sudo, got %q", msg)
		}
	}
}

func TestMoveFile_RenamesWithinSameDirectory(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dest := filepath.Join(dir, "dest")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := moveFile(src, dest); err != nil {
		t.Fatalf("moveFile: %v", err)
	}

	if _, err := os.Stat(src); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("src should be gone after move, got err=%v", err)
	}
	body, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(body) != "hello" {
		t.Errorf("dest content = %q, want %q", body, "hello")
	}
}

func TestMoveFile_OverwritesExistingDest(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dest := filepath.Join(dir, "dest")
	if err := os.WriteFile(src, []byte("new"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := os.WriteFile(dest, []byte("old"), 0o644); err != nil {
		t.Fatalf("write dest: %v", err)
	}

	if err := moveFile(src, dest); err != nil {
		t.Fatalf("moveFile: %v", err)
	}

	body, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(body) != "new" {
		t.Errorf("dest should contain new content, got %q", body)
	}
}

func TestStageDownload_WritesBodyToTempFile(t *testing.T) {
	payload := "fake binary contents"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, payload)
	}))
	defer srv.Close()

	path, err := stageDownload(srv.URL)
	if err != nil {
		t.Fatalf("stageDownload: %v", err)
	}
	t.Cleanup(func() { os.Remove(path) })

	if !strings.HasPrefix(filepath.Base(path), "gpk-update-") {
		t.Errorf("temp file name should carry the gpk-update- prefix, got %q", filepath.Base(path))
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read temp file: %v", err)
	}
	if string(body) != payload {
		t.Errorf("staged body = %q, want %q", body, payload)
	}
}

func TestStageDownload_ReportsNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	if _, err := stageDownload(srv.URL); err == nil {
		t.Fatal("expected error on 404 response")
	} else if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention the status code, got %q", err)
	}
}

func TestBinaryName_MatchesCurrentPlatform(t *testing.T) {
	got := binaryName()
	wantPrefix := "gpk-" + runtime.GOOS + "-" + runtime.GOARCH
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("binaryName() = %q, want prefix %q", got, wantPrefix)
	}
	if runtime.GOOS == "windows" && !strings.HasSuffix(got, ".exe") {
		t.Errorf("windows binary name should end in .exe, got %q", got)
	}
	if runtime.GOOS != "windows" && strings.HasSuffix(got, ".exe") {
		t.Errorf("non-windows binary name should not end in .exe, got %q", got)
	}
}
