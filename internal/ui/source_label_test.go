package ui

import (
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestSourceLabelUsesCompactCaskName(t *testing.T) {
	if got := sourceLabel(model.SourceBrewCask); got != "cask" {
		t.Fatalf("sourceLabel(brew-cask) = %q, want %q", got, "cask")
	}
	if got := sourceLabel(model.SourceBrew); got != "brew" {
		t.Fatalf("sourceLabel(brew) = %q, want %q", got, "brew")
	}
}

func TestRenderFixedBadgeUsesCompactCaskName(t *testing.T) {
	badge := renderFixedBadge(model.SourceBrewCask)
	if strings.Contains(badge, "brew-cask") {
		t.Fatalf("badge leaked internal source name: %q", badge)
	}
	if !strings.Contains(badge, "cask") {
		t.Fatalf("badge did not include compact cask label: %q", badge)
	}
}

func TestFormatSourceUsesCompactCaskName(t *testing.T) {
	got := formatSource(model.Package{Source: model.SourceBrewCask})
	if got != "cask" {
		t.Fatalf("formatSource(brew-cask) = %q, want %q", got, "cask")
	}
}

func TestBatchSourceListUsesCompactCaskName(t *testing.T) {
	var b strings.Builder
	ops := []batchOp{{pkg: model.Package{Name: "firefox", Source: model.SourceBrewCask}}}
	writeSortedSourceLists(&b, ops, 80)
	got := b.String()
	if strings.Contains(got, "brew-cask") {
		t.Fatalf("batch source list leaked internal source name: %q", got)
	}
	if !strings.Contains(got, "cask: firefox") {
		t.Fatalf("batch source list missing compact cask label: %q", got)
	}
}
