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

func TestRankPackages_FuzzyFallbackTriggersWhenAllTiersEmpty(t *testing.T) {
	// Query "gt" has no strict substring hit in these names/descriptions,
	// but fuzzy will match "git" because g-t appears in order.
	pkgs := []model.Package{
		{Name: "abc", Description: "xyz"},
		{Name: "git", Description: "version control"},
	}
	got := rankPackages(pkgs, "gt")
	if len(got) == 0 {
		t.Fatalf("expected fuzzy fallback to return at least one result")
	}
	if got[0].Name != "git" {
		t.Errorf("expected fuzzy fallback to return git first, got %s", got[0].Name)
	}
}

func TestRankPackages_FuzzyFallbackNotMixedWithTiers(t *testing.T) {
	// "git" is a strict prefix hit; "xyz" would be a fuzzy hit for "gt" but
	// must NOT appear because the strict tier is non-empty.
	pkgs := []model.Package{
		{Name: "git"},
		{Name: "xyz"},
	}
	got := rankPackages(pkgs, "git")
	if len(got) != 1 || got[0].Name != "git" {
		t.Fatalf("expected only [git], got %v", got)
	}
}

func TestRankPackages_NoMatchAnywhereReturnsEmpty(t *testing.T) {
	pkgs := []model.Package{
		{Name: "abc"},
		{Name: "def"},
	}
	got := rankPackages(pkgs, "xyz")
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestRankGroupsByName_TierOrdering(t *testing.T) {
	// ssl-tools: tier 1 (prefix), libssl: tier 2 (contains), curl: tier 3 (desc).
	groups := []searchResultGroup{
		{name: "curl", entries: []model.Package{{Name: "curl", Description: "ssl-capable http client"}}},
		{name: "libssl", entries: []model.Package{{Name: "libssl", Description: ""}}},
		{name: "ssl-tools", entries: []model.Package{{Name: "ssl-tools", Description: ""}}},
	}
	got := rankGroupsByName(groups, "ssl")
	want := []string{"ssl-tools", "libssl", "curl"}
	if len(got) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(got))
	}
	for i, name := range want {
		if got[i].name != name {
			t.Errorf("index %d: expected %s, got %s", i, name, got[i].name)
		}
	}
}

func TestRankGroupsByName_NonMatchingPreservedAtBottom(t *testing.T) {
	// ssl-tools: tier 1, libssl: tier 2, curl: tier 3 (desc-only),
	// unrelated: tierNone — must appear at the end.
	groups := []searchResultGroup{
		{name: "curl", entries: []model.Package{{Name: "curl", Description: "ssl-capable"}}},
		{name: "libssl", entries: []model.Package{{Name: "libssl"}}},
		{name: "ssl-tools", entries: []model.Package{{Name: "ssl-tools"}}},
		{name: "unrelated", entries: []model.Package{{Name: "unrelated", Description: "nothing here"}}},
	}
	got := rankGroupsByName(groups, "ssl")
	want := []string{"ssl-tools", "libssl", "curl", "unrelated"}
	if len(got) != 4 {
		t.Fatalf("expected 4 groups (pure reorder), got %d", len(got))
	}
	for i, name := range want {
		if got[i].name != name {
			t.Errorf("index %d: expected %s, got %s", i, name, got[i].name)
		}
	}
}

func TestRankGroupsByName_EmptyQueryReturnsInput(t *testing.T) {
	groups := []searchResultGroup{
		{name: "a", entries: []model.Package{{Name: "a"}}},
		{name: "b", entries: []model.Package{{Name: "b"}}},
	}
	got := rankGroupsByName(groups, "")
	if len(got) != 2 || got[0].name != "a" || got[1].name != "b" {
		t.Fatalf("empty query did not return input order: %v", got)
	}
	// Whitespace-only query must also return input unchanged.
	got = rankGroupsByName(groups, "   ")
	if len(got) != 2 || got[0].name != "a" || got[1].name != "b" {
		t.Fatalf("whitespace query did not return input order: %v", got)
	}
}

func TestRankGroupsByName_LengthInvariant(t *testing.T) {
	groups := []searchResultGroup{
		{name: "alpha", entries: []model.Package{{Name: "alpha"}}},
		{name: "beta", entries: []model.Package{{Name: "beta"}}},
		{name: "gamma", entries: []model.Package{{Name: "gamma"}}},
	}
	for _, q := range []string{"", "x", "a", "ALPHA"} {
		got := rankGroupsByName(groups, q)
		if len(got) != len(groups) {
			t.Errorf("length invariant violated for query %q: got %d, want %d", q, len(got), len(groups))
		}
	}
}
