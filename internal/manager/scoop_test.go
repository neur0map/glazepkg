package manager

import (
	"testing"
	"time"
)

// ── parseTabular ──────────────────────────────────────────────────────────────

func TestScoopParseTabular(t *testing.T) {
	s := &Scoop{}
	input := "  Name             Version          Source  Updated\n" +
		"  ----             -------          ------  -------\n" +
		"  7zip             24.09            main    2024-01-05 15:09:21\n" +
		"  git              2.44.0           main    2024-01-08 11:22:33\n" +
		"  aria2            1.37.0-1         extras  2024-02-01 09:00:00\n"

	pkgs, err := s.parseTabular(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d: %v", len(pkgs), pkgs)
	}

	tests := []struct {
		name, version, repo string
	}{
		{"7zip", "24.09", "main"},
		{"git", "2.44.0", "main"},
		{"aria2", "1.37.0-1", "extras"},
	}
	for i, tt := range tests {
		if pkgs[i].Name != tt.name {
			t.Errorf("[%d] Name = %q, want %q", i, pkgs[i].Name, tt.name)
		}
		if pkgs[i].Version != tt.version {
			t.Errorf("[%d] Version = %q, want %q", i, pkgs[i].Version, tt.version)
		}
		if pkgs[i].Repository != tt.repo {
			t.Errorf("[%d] Repository = %q, want %q", i, pkgs[i].Repository, tt.repo)
		}
	}
}

func TestScoopParseTabularTimestamp(t *testing.T) {
	s := &Scoop{}
	input := "  Name  Version  Source  Updated\n" +
		"  ----  -------  ------  -------\n" +
		"  git   2.44.0   main    2024-03-15 10:30:00\n"

	pkgs, err := s.parseTabular(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	want, _ := time.ParseInLocation("2006-01-02 15:04:05", "2024-03-15 10:30:00", time.Local)
	if !pkgs[0].InstalledAt.Equal(want) {
		t.Errorf("InstalledAt = %v, want %v", pkgs[0].InstalledAt, want)
	}
}

func TestScoopParseTabularDateOnlyTimestamp(t *testing.T) {
	s := &Scoop{}
	input := "  Name  Version  Source  Updated\n" +
		"  ----  -------  ------  -------\n" +
		"  git   2.44.0   main    2024-03-15\n"

	pkgs, err := s.parseTabular(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := time.ParseInLocation("2006-01-02", "2024-03-15", time.Local)
	if !pkgs[0].InstalledAt.Equal(want) {
		t.Errorf("InstalledAt = %v, want %v", pkgs[0].InstalledAt, want)
	}
}

func TestScoopParseTabularNoSeparator(t *testing.T) {
	s := &Scoop{}
	// No separator → no packages
	input := "  git  2.44.0  main\n  7zip  24.09  main\n"
	pkgs, err := s.parseTabular(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages without separator, got %d", len(pkgs))
	}
}

// ── parseLegacy ───────────────────────────────────────────────────────────────

func TestScoopParseLegacy(t *testing.T) {
	s := &Scoop{}
	input := "Installed apps:\n" +
		"  7zip 24.09 [main]\n" +
		"  git 2.44.0 [main]\n" +
		"  aria2 1.37.0-1 [extras]\n"

	pkgs, err := s.parseLegacy(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d: %v", len(pkgs), pkgs)
	}
	tests := []struct{ name, version string }{
		{"7zip", "24.09"},
		{"git", "2.44.0"},
		{"aria2", "1.37.0-1"},
	}
	for i, tt := range tests {
		if pkgs[i].Name != tt.name {
			t.Errorf("[%d] Name = %q, want %q", i, pkgs[i].Name, tt.name)
		}
		if pkgs[i].Version != tt.version {
			t.Errorf("[%d] Version = %q, want %q", i, pkgs[i].Version, tt.version)
		}
	}
}

func TestScoopParseLegacyNoBracket(t *testing.T) {
	s := &Scoop{}
	// Some older versions omit the [bucket] annotation
	input := "Installed apps:\n  git 2.44.0\n  curl 8.4.0\n"
	pkgs, err := s.parseLegacy(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d: %v", len(pkgs), pkgs)
	}
	if pkgs[0].Name != "git" || pkgs[0].Version != "2.44.0" {
		t.Errorf("pkg[0] = {%s %s}", pkgs[0].Name, pkgs[0].Version)
	}
}

func TestScoopParseLegacySkipsHeadersAndBlankLines(t *testing.T) {
	s := &Scoop{}
	input := "\nInstalled apps:\n\n  git 2.44.0 [main]\n\n"
	pkgs, err := s.parseLegacy(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 || pkgs[0].Name != "git" {
		t.Errorf("expected [{git 2.44.0}], got %v", pkgs)
	}
}

// ── parseStatusOutput ─────────────────────────────────────────────────────────

func TestScoopParseStatusOutput(t *testing.T) {
	s := &Scoop{}
	input := " Name  Installed Version  Latest Version  Missing Dependencies  Info\n" +
		" ----  -----------------  --------------  --------------------  ----\n" +
		" git   2.43.0             2.44.0\n" +
		" 7zip  23.01              24.09\n"

	updates := s.parseStatusOutput(input)
	if len(updates) != 2 {
		t.Fatalf("expected 2 updates, got %d: %v", len(updates), updates)
	}
	if updates["git"] != "2.44.0" {
		t.Errorf("git = %q, want 2.44.0", updates["git"])
	}
	if updates["7zip"] != "24.09" {
		t.Errorf("7zip = %q, want 24.09", updates["7zip"])
	}
}

func TestScoopParseStatusOutputEverythingOk(t *testing.T) {
	s := &Scoop{}
	input := "Everything is ok!\n"
	updates := s.parseStatusOutput(input)
	if len(updates) != 0 {
		t.Errorf("expected 0 updates for 'Everything is ok', got %v", updates)
	}
}

func TestScoopParseStatusOutputScoopUpdateMessage(t *testing.T) {
	s := &Scoop{}
	// Some versions say "Run scoop update to update Scoop"
	input := "Run scoop update to update Scoop itself.\n"
	updates := s.parseStatusOutput(input)
	if len(updates) != 0 {
		t.Errorf("expected 0 updates, got %v", updates)
	}
}
