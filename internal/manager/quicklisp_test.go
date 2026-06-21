package manager

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestParseQuicklispReleases(t *testing.T) {
	names := []string{"alexandria.txt", "cl-ppcre.txt", ".gitkeep", "notes.md"}
	pkgs := parseQuicklispReleases(names, "2024-04-01")

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	want := []string{"alexandria", "cl-ppcre"}
	for i, exp := range want {
		if pkgs[i].Name != exp {
			t.Errorf("pkg %d Name = %q, want %q", i, pkgs[i].Name, exp)
		}
		if pkgs[i].Version != "2024-04-01" {
			t.Errorf("pkg %d Version = %q, want 2024-04-01", i, pkgs[i].Version)
		}
		if pkgs[i].Source != model.SourceQuicklisp {
			t.Errorf("pkg %d Source = %q, want %q", i, pkgs[i].Source, model.SourceQuicklisp)
		}
		if pkgs[i].Repository != "quicklisp" {
			t.Errorf("pkg %d Repository = %q, want quicklisp", i, pkgs[i].Repository)
		}
	}
}

func TestParseDistVersion(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{"present", "name: quicklisp\nversion: 2024-04-01\nsystem-index-url: x\n", "2024-04-01"},
		{"absent", "name: quicklisp\nsystem-index-url: x\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseDistVersion([]byte(tt.data)); got != tt.want {
				t.Errorf("parseDistVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
