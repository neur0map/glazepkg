package ui

import (
	"strings"
	"testing"
)

func TestClassifyMatch(t *testing.T) {
	cases := []struct {
		name, desc, query string
		want              matchTier
	}{
		{"git", "", "git", tierPrefix},
		{"git-lfs", "", "git", tierPrefix},
		{"libgit2", "", "git", tierContains},
		{"openssl", "supports git", "git", tierDesc},
		{"curl", "http tool", "git", tierNone},
		{"GIT", "", "git", tierPrefix},
		{"git", "", "GIT", tierPrefix},
		{"", "", "git", tierNone},
	}
	for _, c := range cases {
		got := classifyMatch(strings.ToLower(c.name), strings.ToLower(c.desc), strings.ToLower(c.query))
		if got != c.want {
			t.Errorf("classifyMatch(%q,%q,%q) = %v, want %v", c.name, c.desc, c.query, got, c.want)
		}
	}
}
