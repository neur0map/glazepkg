package manager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSkippedAndEnabled(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if got := Skipped(); len(got) != 0 {
		t.Errorf("no config should skip nothing, got %v", got)
	}
	if len(Enabled()) != len(All()) {
		t.Error("no skip list should enable every manager")
	}

	if err := os.MkdirAll(filepath.Join(dir, "glazepkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "[managers]\nskip = [\"pnpm\", \" npm \", \"\"]\n"
	if err := os.WriteFile(filepath.Join(dir, "glazepkg", "config.toml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	skip := Skipped()
	if !skip["pnpm"] || !skip["npm"] {
		t.Errorf("skip set = %v, want pnpm and npm (whitespace trimmed)", skip)
	}
	if skip[""] {
		t.Error("an empty entry must not skip an unnamed manager")
	}

	enabled := Enabled()
	if len(enabled) != len(All())-2 {
		t.Errorf("enabled %d managers, want %d", len(enabled), len(All())-2)
	}
	for _, m := range enabled {
		if m.Name() == "pnpm" || m.Name() == "npm" {
			t.Errorf("%s is skipped but still enabled", m.Name())
		}
	}
}
