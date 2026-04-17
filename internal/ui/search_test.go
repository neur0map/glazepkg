package ui

import (
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/model"
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

func TestRankPackages_EmptyQueryReturnsInput(t *testing.T) {
	pkgs := []model.Package{{Name: "a"}, {Name: "b"}}
	got := rankPackages(pkgs, "")
	if len(got) != 2 || got[0].Name != "a" || got[1].Name != "b" {
		t.Fatalf("empty query did not return input unchanged: %v", got)
	}
}

func TestRankPackages_WhitespaceQueryReturnsInput(t *testing.T) {
	pkgs := []model.Package{{Name: "a"}, {Name: "b"}}
	got := rankPackages(pkgs, "   ")
	if len(got) != 2 || got[0].Name != "a" || got[1].Name != "b" {
		t.Fatalf("whitespace query did not return input unchanged: %v", got)
	}
}

func TestRankPackages_PrefixBeatsContains(t *testing.T) {
	pkgs := []model.Package{
		{Name: "libgit2"},
		{Name: "git"},
	}
	got := rankPackages(pkgs, "git")
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	if got[0].Name != "git" || got[1].Name != "libgit2" {
		t.Errorf("expected [git, libgit2], got [%s, %s]", got[0].Name, got[1].Name)
	}
}

func TestRankPackages_PrefixContainsDescOrdering(t *testing.T) {
	// git: tier 1 (prefix), libgit2: tier 2 (contains), vim: tier 3 (desc).
	// Input order is intentionally scrambled to prove buckets do the sorting,
	// not input order.
	pkgs := []model.Package{
		{Name: "vim", Description: "editor with git integration"},
		{Name: "libgit2", Description: ""},
		{Name: "git", Description: ""},
	}
	got := rankPackages(pkgs, "git")
	want := []string{"git", "libgit2", "vim"}
	if len(got) != 3 {
		t.Fatalf("expected 3 results, got %d: %v", len(got), got)
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Errorf("index %d: expected %s, got %s", i, name, got[i].Name)
		}
	}
}

func TestRankPackages_WithinTierAlphabeticalOrderPreserved(t *testing.T) {
	// Input is already alphabetical; all tier-1 hits should keep that order.
	pkgs := []model.Package{
		{Name: "git"},
		{Name: "git-lfs"},
		{Name: "github"},
	}
	got := rankPackages(pkgs, "git")
	want := []string{"git", "git-lfs", "github"}
	if len(got) != 3 {
		t.Fatalf("expected 3 results, got %d", len(got))
	}
	for i, name := range want {
		if got[i].Name != name {
			t.Errorf("index %d: expected %s, got %s", i, name, got[i].Name)
		}
	}
}

func TestRankPackages_CaseInsensitive(t *testing.T) {
	pkgs := []model.Package{{Name: "Git"}}
	got := rankPackages(pkgs, "git")
	if len(got) != 1 {
		t.Fatalf("case-insensitive match on name failed: got %d", len(got))
	}

	pkgs = []model.Package{{Name: "git"}}
	got = rankPackages(pkgs, "GIT")
	if len(got) != 1 {
		t.Fatalf("case-insensitive query failed: got %d", len(got))
	}
}

func TestRankPackages_NonMatchesDropped(t *testing.T) {
	pkgs := []model.Package{
		{Name: "git"},
		{Name: "curl", Description: "http tool"},
	}
	got := rankPackages(pkgs, "git")
	if len(got) != 1 || got[0].Name != "git" {
		t.Fatalf("expected only [git], got %v", got)
	}
}
