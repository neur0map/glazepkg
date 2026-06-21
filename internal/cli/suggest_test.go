package cli

import "testing"

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"install", "install", 0},
		{"install", "instal", 1},
		{"", "abc", 3},
		{"abc", "", 3},
		{"ffmpeg", "ffmpgg", 1},
		{"kitten", "sitting", 3},
	}
	for _, c := range cases {
		if got := levenshtein(c.a, c.b); got != c.want {
			t.Errorf("levenshtein(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestClosest(t *testing.T) {
	subs := []string{"install", "remove", "upgrade", "list", "search"}
	got, d := closest("instal", subs)
	if got != "install" || d != 1 {
		t.Errorf("closest(instal) = %q (d=%d), want install (1)", got, d)
	}
}

func TestSuggestNames(t *testing.T) {
	pool := []string{"ffmpeg", "ffmpeg-git", "vlc", "mpv"}
	got := suggestNames("ffmpgg", pool, 3, 5)
	if len(got) == 0 || got[0] != "ffmpeg" {
		t.Errorf("suggestNames(ffmpgg) = %v, want ffmpeg first", got)
	}
	// A far-off query yields nothing within the distance budget.
	if got := suggestNames("zzzzzz", pool, 2, 5); len(got) != 0 {
		t.Errorf("suggestNames(zzzzzz) = %v, want empty", got)
	}
}
