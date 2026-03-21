package manager

import (
	"encoding/json"
	"testing"
)

// ── JSON unmarshaling ─────────────────────────────────────────────────────────

func TestWindowsUpdatesScanJSON(t *testing.T) {
	w := &WindowsUpdates{}

	updates := []winUpdate{
		{
			Title:      "2024-01 Cumulative Update for Windows 11",
			KBArticle:  "5034203",
			Size:       512 * 1024 * 1024,
			Severity:   "Important",
			Categories: "Security Updates",
		},
		{
			Title:      "Definition Update for Windows Defender",
			KBArticle:  "N/A",
			Size:       8 * 1024 * 1024,
			Severity:   "Unspecified",
			Categories: "Definition Updates",
		},
	}
	data, _ := json.Marshal(updates)

	// Simulate what Scan() does after receiving PS output
	var parsed []winUpdate
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	pkgs := w.buildPackages(parsed)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}

	// First update: has KB number → name should include [KB...]
	if pkgs[0].Name != "2024-01 Cumulative Update for Windows 11 [KB5034203]" {
		t.Errorf("pkg[0].Name = %q", pkgs[0].Name)
	}
	if pkgs[0].Version != "5034203" {
		t.Errorf("pkg[0].Version = %q, want KB article", pkgs[0].Version)
	}
	if pkgs[0].Description != "Important — Security Updates" {
		t.Errorf("pkg[0].Description = %q", pkgs[0].Description)
	}
	if pkgs[0].SizeBytes != 512*1024*1024 {
		t.Errorf("pkg[0].SizeBytes = %d", pkgs[0].SizeBytes)
	}

	// Second update: no KB → name unchanged, description without severity prefix
	if pkgs[1].Name != "Definition Update for Windows Defender" {
		t.Errorf("pkg[1].Name = %q", pkgs[1].Name)
	}
	if pkgs[1].Description != "Definition Updates" {
		t.Errorf("pkg[1].Description = %q, want plain category", pkgs[1].Description)
	}
}

func TestWindowsUpdatesScanEmptyJSON(t *testing.T) {
	w := &WindowsUpdates{}
	var parsed []winUpdate
	if err := json.Unmarshal([]byte("[]"), &parsed); err != nil {
		t.Fatalf("unmarshal empty array: %v", err)
	}
	pkgs := w.buildPackages(parsed)
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages for empty update list, got %d", len(pkgs))
	}
}

func TestWindowsUpdatesScanLimit(t *testing.T) {
	w := &WindowsUpdates{}
	// Build 60 updates; should be capped at 50
	updates := make([]winUpdate, 60)
	for i := range updates {
		updates[i] = winUpdate{
			Title:     "Update",
			KBArticle: "N/A",
			Severity:  "Unspecified",
		}
	}
	pkgs := w.buildPackages(updates)
	if len(pkgs) != 50 {
		t.Errorf("expected 50 packages (cap), got %d", len(pkgs))
	}
}

func TestWindowsUpdatesSeverityUnspecifiedNoPrefixInDesc(t *testing.T) {
	w := &WindowsUpdates{}
	updates := []winUpdate{{
		Title:      "Driver Update",
		KBArticle:  "N/A",
		Severity:   "Unspecified",
		Categories: "Drivers",
	}}
	pkgs := w.buildPackages(updates)
	// "Unspecified" severity → description should be just the category
	if pkgs[0].Description != "Drivers" {
		t.Errorf("Description = %q, want %q", pkgs[0].Description, "Drivers")
	}
}

func TestWindowsUpdatesSizeFormatted(t *testing.T) {
	w := &WindowsUpdates{}
	updates := []winUpdate{{
		Title:     "Big Update",
		KBArticle: "1234567",
		Size:      2 * 1024 * 1024 * 1024, // 2 GiB
	}}
	pkgs := w.buildPackages(updates)
	if pkgs[0].Size != "2.0 GiB" {
		t.Errorf("Size = %q, want %q", pkgs[0].Size, "2.0 GiB")
	}
}
