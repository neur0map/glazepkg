package cli

import (
	"reflect"
	"testing"
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
		{"double-dash separator", []string{"--json", "--", "-weird-name"}, []string{"--json", "-weird-name"}},
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
