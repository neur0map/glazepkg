package manager

import (
	"os"
	"path/filepath"
	"testing"
)

// A machine can carry the dotnet host and runtime with no SDK, and there every
// `dotnet tool` command fails. Claiming the manager on those boxes made every
// `gpk upgrade` exit non-zero.
func TestDotnetSDKPresent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DOTNET_ROOT", root)
	t.Setenv("PATH", root) // no dotnet binary to fall back to
	t.Setenv("HOME", root)

	if dotnetSDKPresent() {
		t.Error("no sdk dir at all should not count as present")
	}

	sdk := filepath.Join(root, "sdk")
	if err := os.Mkdir(sdk, 0o755); err != nil {
		t.Fatal(err)
	}
	if dotnetSDKPresent() {
		t.Error("an empty sdk dir is left behind by uninstalls and must not count")
	}

	if err := os.Mkdir(filepath.Join(sdk, "9.0.317"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !dotnetSDKPresent() {
		t.Error("a populated sdk dir should count as present")
	}
}

func TestDotnetToolAvailableNeedsBothBinaryAndSDK(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	t.Setenv("HOME", dir)
	t.Setenv("DOTNET_ROOT", "")

	d := &DotnetTool{}
	if d.Available() {
		t.Error("no dotnet on PATH should not be available")
	}

	if err := os.WriteFile(filepath.Join(dir, "dotnet"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if d.Available() {
		t.Error("dotnet without an SDK should not be available")
	}

	if err := os.MkdirAll(filepath.Join(dir, "sdk", "9.0.317"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !d.Available() {
		t.Error("dotnet with an SDK beside it should be available")
	}
}

// aqua with no config fails its own commands with "configuration file isn't
// found", so gpk must not claim it.
func TestAquaConfigPresent(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "deep", "nested", "dir")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(work)
	t.Setenv("AQUA_GLOBAL_CONFIG", "")

	if aquaConfigPresent() {
		t.Error("no config anywhere should report absent")
	}

	// A config in a parent directory counts: aqua's finder walks up.
	parent := filepath.Join(root, "deep")
	cfg := filepath.Join(parent, "aqua.yaml")
	if err := os.WriteFile(cfg, []byte("packages: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !aquaConfigPresent() {
		t.Error("aqua.yaml in a parent dir should report present")
	}
	if err := os.Remove(cfg); err != nil {
		t.Fatal(err)
	}

	// The dotted name is accepted too.
	dotted := filepath.Join(work, ".aqua.yml")
	if err := os.WriteFile(dotted, []byte("packages: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !aquaConfigPresent() {
		t.Error(".aqua.yml should report present")
	}
	if err := os.Remove(dotted); err != nil {
		t.Fatal(err)
	}

	// A global config outside the tree counts, which is how gpk drives bulk
	// upgrades for a global-tools user.
	global := filepath.Join(root, "global.yaml")
	if err := os.WriteFile(global, []byte("packages: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AQUA_GLOBAL_CONFIG", global)
	if !aquaConfigPresent() {
		t.Error("AQUA_GLOBAL_CONFIG naming a real file should report present")
	}

	t.Setenv("AQUA_GLOBAL_CONFIG", filepath.Join(root, "does-not-exist.yaml"))
	if aquaConfigPresent() {
		t.Error("AQUA_GLOBAL_CONFIG naming a missing file should report absent")
	}
}
