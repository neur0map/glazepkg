package cli

import (
	"reflect"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
)

func TestReorderFlagsFirst(t *testing.T) {
	stringFlags := []string{"manager", "m"}
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"empty", []string{}, []string{}},
		{"already in order", []string{"--json", "git"}, []string{"--json", "git"}},
		{"single positional", []string{"git"}, []string{"git"}},
		{"single flag", []string{"--json"}, []string{"--json"}},
		{"flag after positional", []string{"git", "--json"}, []string{"--json", "git"}},
		{"multiple positional + flags", []string{"vim", "git", "--json", "--quiet"}, []string{"--json", "--quiet", "vim", "git"}},
		{"string flag with value", []string{"pkg", "--manager", "pacman"}, []string{"--manager", "pacman", "pkg"}},
		{"short string flag with value", []string{"pkg", "-m", "pacman"}, []string{"-m", "pacman", "pkg"}},
		{"equals form does not consume next", []string{"pkg", "--manager=pacman", "--json"}, []string{"--manager=pacman", "--json", "pkg"}},
		{"double-dash separator", []string{"--json", "--", "-weird-name"}, []string{"--json", "--", "-weird-name"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := reorderFlagsFirst(c.in, stringFlags)
			// Normalize empty slices for comparison
			if len(got) == 0 && len(c.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("reorderFlagsFirst(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestPrepManagerArgs(t *testing.T) {
	mgrs := manager.All()
	cases := []struct {
		name       string
		in         []string
		valueFlags []string
		want       []string
	}{
		{"inline manager", []string{"ffmpeg", "-aur"}, nil, []string{"--manager", "aur", "ffmpeg"}},
		{"cask alias", []string{"--cask", "firefox"}, nil, []string{"--manager", "brew-cask", "firefox"}},
		{"merge explicit and inline", []string{"pkg", "--manager", "brew", "-aur"}, nil, []string{"--manager", "brew,aur", "pkg"}},
		{"value flag kept with value", []string{"ripgrep", "--limit", "5"}, []string{"limit"}, []string{"--limit", "5", "ripgrep"}},
		{"no manager", []string{"git", "--json"}, nil, []string{"--json", "git"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := prepManagerArgs(c.in, mgrs, c.valueFlags...)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("prepManagerArgs(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
