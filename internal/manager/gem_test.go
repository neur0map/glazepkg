package manager

import "testing"

func TestIsSystemGemPath(t *testing.T) {
	cases := []struct {
		goos, path string
		want       bool
	}{
		{"darwin", "/usr/bin/gem", true},
		{"darwin", "/opt/homebrew/opt/ruby/bin/gem", false},
		{"darwin", "/Users/x/.rbenv/shims/gem", false},
		{"linux", "/usr/bin/gem", false},
		{"windows", "C:\\Ruby\\bin\\gem", false},
	}
	for _, c := range cases {
		if got := isSystemGemPath(c.goos, c.path); got != c.want {
			t.Errorf("isSystemGemPath(%q, %q) = %v, want %v", c.goos, c.path, got, c.want)
		}
	}
}
