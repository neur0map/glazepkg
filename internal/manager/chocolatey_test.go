package manager

import (
	"testing"
)

// ── parseListOutput ───────────────────────────────────────────────────────────

func TestChocolateyParseListOutputV1(t *testing.T) {
	c := &Chocolatey{}
	input := "Chocolatey v1.4.0\n" +
		"chocolatey 1.4.0\n" +
		"git 2.44.0\n" +
		"vscode 1.85.2\n" +
		"3 packages installed.\n"

	pkgs := c.parseListOutput(input)
	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}
	tests := []struct{ name, version string }{
		{"chocolatey", "1.4.0"},
		{"git", "2.44.0"},
		{"vscode", "1.85.2"},
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

func TestChocolateyParseListOutputV2(t *testing.T) {
	c := &Chocolatey{}
	// v2 format is identical but no --local-only flag needed
	input := "Chocolatey v2.3.0\n" +
		"chocolatey 2.3.0\n" +
		"chocolatey-compatibility.extension 1.0.0\n" +
		"git 2.44.0\n" +
		"3 packages installed.\n"

	pkgs := c.parseListOutput(input)
	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d: %v", len(pkgs), pkgs)
	}
	if pkgs[1].Name != "chocolatey-compatibility.extension" {
		t.Errorf("pkg[1].Name = %q", pkgs[1].Name)
	}
}

func TestChocolateyParseListOutputSkipsHeaderFooter(t *testing.T) {
	c := &Chocolatey{}
	input := "Chocolatey v1.4.0\n" +
		"WARNING: Some warning here\n" +
		"Validation Warnings:\n" +
		"  - Something\n" +
		"git 2.44.0\n" +
		"1 packages installed.\n"

	pkgs := c.parseListOutput(input)
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d: %v", len(pkgs), pkgs)
	}
	if pkgs[0].Name != "git" {
		t.Errorf("Name = %q, want %q", pkgs[0].Name, "git")
	}
}

func TestChocolateyParseListOutputNonDigitVersionSkipped(t *testing.T) {
	c := &Chocolatey{}
	// Lines where the "version" field doesn't start with a digit should be skipped
	input := "Chocolatey v1.4.0\n" +
		"git 2.44.0\n" +
		"sometext notaversion\n" +
		"1 packages installed.\n"

	pkgs := c.parseListOutput(input)
	if len(pkgs) != 1 || pkgs[0].Name != "git" {
		t.Errorf("expected only 'git', got %v", pkgs)
	}
}

func TestChocolateyParseListOutputEmpty(t *testing.T) {
	c := &Chocolatey{}
	input := "Chocolatey v2.3.0\n0 packages installed.\n"
	pkgs := c.parseListOutput(input)
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

// ── parseOutdatedOutput ───────────────────────────────────────────────────────

func TestChocolateyParseOutdatedOutput(t *testing.T) {
	c := &Chocolatey{}
	input := "Chocolatey v1.4.0\n" +
		"Outdated Packages\n" +
		"Output is package name | current version | available version | pinned?\n" +
		"git|2.44.0|2.45.0|false\n" +
		"vscode|1.85.2|1.86.0|false\n" +
		"2 packages are outdated.\n"

	updates := c.parseOutdatedOutput(input)
	if len(updates) != 2 {
		t.Fatalf("expected 2 updates, got %d: %v", len(updates), updates)
	}
	if updates["git"] != "2.45.0" {
		t.Errorf("git update = %q, want %q", updates["git"], "2.45.0")
	}
	if updates["vscode"] != "1.86.0" {
		t.Errorf("vscode update = %q, want %q", updates["vscode"], "1.86.0")
	}
}

func TestChocolateyParseOutdatedOutputPinned(t *testing.T) {
	c := &Chocolatey{}
	// Pinned packages still appear in choco outdated but shouldn't be an error to parse
	input := "Chocolatey v2.3.0\n" +
		"git|2.44.0|2.45.0|false\n" +
		"pinned-pkg|1.0.0|2.0.0|true\n"

	updates := c.parseOutdatedOutput(input)
	if updates["git"] != "2.45.0" {
		t.Errorf("git = %q, want 2.45.0", updates["git"])
	}
	if updates["pinned-pkg"] != "2.0.0" {
		t.Errorf("pinned-pkg = %q, want 2.0.0", updates["pinned-pkg"])
	}
}

func TestChocolateyParseOutdatedOutputNone(t *testing.T) {
	c := &Chocolatey{}
	input := "Chocolatey v2.3.0\n0 packages are outdated.\n"
	updates := c.parseOutdatedOutput(input)
	if len(updates) != 0 {
		t.Errorf("expected 0 updates, got %v", updates)
	}
}

// ── isV2 version detection ────────────────────────────────────────────────────

func TestChocolateyIsV2Detection(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"2.3.0", true},
		{"2.0.0", true},
		{"9.1.2", true},
		{"10.0.0", true}, // double-digit major must not be misclassified
		{"1.4.0", false},
		{"1.0.0", false},
		{"0.9.0", false},
	}
	for _, tt := range tests {
		got := chocoIsV2OrLater(tt.version)
		if got != tt.want {
			t.Errorf("version %q: isV2 = %v, want %v", tt.version, got, tt.want)
		}
	}
}
