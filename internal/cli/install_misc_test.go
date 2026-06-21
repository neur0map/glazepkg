package cli

import "testing"

func TestSplitVersionPin(t *testing.T) {
	cases := []struct {
		raw, name, ver string
	}{
		{"ffmpeg", "ffmpeg", ""},
		{"black@24.1.0", "black", "24.1.0"},
		{"@scope/pkg@1.2.3", "@scope/pkg", "1.2.3"},
		{"@scope/pkg", "@scope/pkg", ""},
	}
	for _, c := range cases {
		name, ver := splitVersionPin(c.raw)
		if name != c.name || ver != c.ver {
			t.Errorf("splitVersionPin(%q) = (%q,%q), want (%q,%q)", c.raw, name, ver, c.name, c.ver)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("truncate short = %q", got)
	}
	if got := truncate("hello world", 5); got != "hell…" {
		t.Errorf("truncate = %q, want hell…", got)
	}
}
