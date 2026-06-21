package version

import (
	"sort"
	"testing"
)

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.10.0", "1.9.0", 1},       // numeric, not lexical
		{"1.14.1-1", "1.14.1-2", -1}, // pacman pkgrel
		{"1.14.1-2", "1.14.1-1", 1},  // pacman pkgrel
		{"1:1.0", "2.0", 1},          // epoch beats upstream
		{"2.0", "1:1.0", -1},         // epoch beats upstream
		{"1.0.0~rc1", "1.0.0", -1},   // tilde is pre-release
		{"1.0.0", "1.0.0~rc1", 1},    // tilde is pre-release
		{"1.2.3.post1", "1.2.3", 1},  // pip post-release
		{"v1.2.0", "1.2.0", 0},       // leading v ignored
		{"1.0", "1.0.0", -1},         // missing component sorts lower
		{"1.14.1-2ubuntu1", "1.14.1-1", 1},
		{"", "1.0", -1},
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestSortNewestFirst(t *testing.T) {
	v := []string{"1.2.16-1", "1.2.10-1", "1.3.0-1", "1.2.16-2"}
	sort.Slice(v, func(i, j int) bool { return Compare(v[i], v[j]) > 0 })
	want := []string{"1.3.0-1", "1.2.16-2", "1.2.16-1", "1.2.10-1"}
	for i := range want {
		if v[i] != want[i] {
			t.Fatalf("sorted = %v, want %v", v, want)
		}
	}
}
