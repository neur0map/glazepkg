package manager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestAMVersionFromString(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://github.com/x/y/releases/download/v1.2.3/app.AppImage", "1.2.3"},
		{"https://h/app-20240115-x86_64.AppImage", "20240115"},
		{"plain-no-version", ""},
		{"v2.0", "2.0"},
	}
	for _, c := range cases {
		if got := amVersionFromString(c.in); got != c.want {
			t.Errorf("amVersionFromString(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestAMScanRoot(t *testing.T) {
	root := t.TempDir()

	appA := filepath.Join(root, "appA")
	if err := os.MkdirAll(appA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appA, "remove"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appA, "version"), []byte("https://github.com/x/y/releases/download/v1.2.3/app.AppImage\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	appB := filepath.Join(root, "appB")
	if err := os.MkdirAll(appB, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appB, "remove"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// No remove file: must be ignored.
	if err := os.MkdirAll(filepath.Join(root, "notapp"), 0o755); err != nil {
		t.Fatal(err)
	}

	pkgs := amScanRoot(root)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}

	byName := make(map[string]model.Package, len(pkgs))
	for _, p := range pkgs {
		byName[p.Name] = p
	}

	a, ok := byName["appA"]
	if !ok {
		t.Fatal("appA not found")
	}
	if a.Version != "1.2.3" {
		t.Errorf("appA version = %q, want %q", a.Version, "1.2.3")
	}
	if a.Location != appA {
		t.Errorf("appA location = %q, want %q", a.Location, appA)
	}

	b, ok := byName["appB"]
	if !ok {
		t.Fatal("appB not found")
	}
	if b.Version != "" {
		t.Errorf("appB version = %q, want empty", b.Version)
	}
	if b.Location != appB {
		t.Errorf("appB location = %q, want %q", b.Location, appB)
	}
}
