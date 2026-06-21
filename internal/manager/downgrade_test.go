package manager

import "testing"

func TestPacmanCacheVersion(t *testing.T) {
	cases := []struct {
		base, name, want string
	}{
		{"a52dec-0.8.0-3-x86_64.pkg.tar.zst", "a52dec", "0.8.0-3"},
		{"python-requests-2.31.0-1-any.pkg.tar.zst", "python-requests", "2.31.0-1"},
		{"foo-1:2.3.4-1-x86_64.pkg.tar.zst", "foo", "1:2.3.4-1"},
		// A longer-named sibling sharing the prefix must not be misread.
		{"python-requests-doc-1.0-1-any.pkg.tar.zst", "python-requests", ""},
		{"unrelated-1.0-1-x86_64.pkg.tar.zst", "foo", ""},
	}
	for _, c := range cases {
		if got := pacmanCacheVersion(c.base, c.name); got != c.want {
			t.Errorf("pacmanCacheVersion(%q,%q) = %q, want %q", c.base, c.name, got, c.want)
		}
	}
}
