package manager

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestParseSoftwareUpdateList(t *testing.T) {
	output := []byte(`Software Update Tool

Finding available software
Software Update found the following new or updated software:
* Label: macOS Sequoia 15.2-24C101
	Title: macOS Sequoia 15.2, Version: 15.2, Size: 6451234KiB, Recommended: YES, Action: restart,
* Label: Safari18.2MontereyAuto-18.2
	Title: Safari, Version: 18.2, Size: 123456KiB, Recommended: YES,
- Label: SomeOptionalUpdate-1.0
	Title: Some Optional Thing, Version: 1.0, Size: 2048KiB, Recommended: NO,
`)
	pkgs := parseSoftwareUpdateList(output)
	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}
	expected := []struct{ name, version, desc string }{
		{"macOS Sequoia 15.2-24C101", "15.2", "macOS Sequoia 15.2"},
		{"Safari18.2MontereyAuto-18.2", "18.2", "Safari"},
		{"SomeOptionalUpdate-1.0", "1.0", "Some Optional Thing"},
	}
	for i, exp := range expected {
		if pkgs[i].Name != exp.name || pkgs[i].Version != exp.version || pkgs[i].Description != exp.desc {
			t.Errorf("pkg %d: got {%q %q %q}, want {%q %q %q}", i,
				pkgs[i].Name, pkgs[i].Version, pkgs[i].Description, exp.name, exp.version, exp.desc)
		}
		if pkgs[i].Source != model.SourceSoftwareUpdate {
			t.Errorf("pkg %d: Source = %q, want %q", i, pkgs[i].Source, model.SourceSoftwareUpdate)
		}
	}
}

func TestParseSoftwareUpdateListEmpty(t *testing.T) {
	pkgs := parseSoftwareUpdateList([]byte("Software Update Tool\n\nNo new software available.\n"))
	if len(pkgs) != 0 {
		t.Fatalf("expected 0 packages, got %d", len(pkgs))
	}
}
