package manager

import (
	"testing"
)

// ── nugetCompare ──────────────────────────────────────────────────────────────

func TestNugetCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		// Equal versions
		{"1.0.0", "1.0.0", 0},
		{"2.3.4", "2.3.4", 0},
		{"0.0.1", "0.0.1", 0},
		// Greater
		{"2.0.0", "1.0.0", 1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.1", "1.0.0", 1},
		{"10.0.0", "9.0.0", 1},
		{"1.10.0", "1.9.0", 1},
		// Less
		{"1.0.0", "2.0.0", -1},
		{"1.0.0", "1.1.0", -1},
		{"1.0.0", "1.0.1", -1},
		// Different number of parts
		{"1.0", "1.0.0", 0},
		{"1.1", "1.0.0", 1},
		{"1.0", "1.0.1", -1},
		// Pre-release: stable beats prerelease of same base version
		{"2.0.0-preview1", "1.9.9", 1}, // higher major wins regardless of prerelease
		{"1.0.0-beta", "1.0.0", -1},    // prerelease < stable of same version
		{"1.0.0", "1.0.0-beta", 1},     // stable > prerelease of same version
		// Four-part NuGet versions
		{"4.8.1.0", "4.8.0.1", 1},
		{"4.8.0.1", "4.8.1.0", -1},
	}
	for _, tt := range tests {
		got := nugetCompare(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("nugetCompare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestNugetSemverGT(t *testing.T) {
	if !nugetSemverGT("2.0.0", "1.9.9") {
		t.Error("2.0.0 should be > 1.9.9")
	}
	if nugetSemverGT("1.0.0", "1.0.0") {
		t.Error("1.0.0 should not be > 1.0.0")
	}
	if nugetSemverGT("1.0.0", "2.0.0") {
		t.Error("1.0.0 should not be > 2.0.0")
	}
}

func TestNugetComparePicksLatest(t *testing.T) {
	versions := []string{"1.0.0", "3.0.0", "2.0.0", "1.5.0", "10.0.0", "9.9.9"}
	latest := versions[0]
	for _, v := range versions[1:] {
		if nugetSemverGT(v, latest) {
			latest = v
		}
	}
	if latest != "10.0.0" {
		t.Errorf("expected latest = 10.0.0, got %s", latest)
	}
}

func TestNugetComparePicksLatestPreRelease(t *testing.T) {
	// "6.0.0-preview.5" beats "5.0.2" because 6 > 5 (higher major wins)
	versions := []string{"5.0.2", "6.0.0-preview.5", "4.8.1"}
	latest := versions[0]
	for _, v := range versions[1:] {
		if nugetSemverGT(v, latest) {
			latest = v
		}
	}
	if latest != "6.0.0-preview.5" {
		t.Errorf("expected 6.0.0-preview.5, got %s", latest)
	}
}
