package ui

import (
	"strings"

	"github.com/sahilm/fuzzy"

	"github.com/neur0map/glazepkg/internal/model"
)

// pkgSearchSource implements fuzzy.Source for package fuzzy matching.
type pkgSearchSource struct {
	pkgs []model.Package
}

// matchTier describes how well a package matches a query.
// Lower-valued tiers rank higher in search results.
type matchTier int

const (
	tierPrefix   matchTier = iota // name has query as a prefix
	tierContains                  // name contains query but not as a prefix
	tierDesc                      // description contains query
	tierNone                      // no match
)

// classifyMatch returns the best tier for (name, desc) against query.
// All three arguments must already be lowercased.
func classifyMatch(loweredName, loweredDesc, loweredQuery string) matchTier {
	if loweredQuery == "" {
		return tierNone
	}
	if strings.HasPrefix(loweredName, loweredQuery) {
		return tierPrefix
	}
	if strings.Contains(loweredName, loweredQuery) {
		return tierContains
	}
	if strings.Contains(loweredDesc, loweredQuery) {
		return tierDesc
	}
	return tierNone
}

func (s pkgSearchSource) String(i int) string {
	p := s.pkgs[i]
	return p.Name + " " + p.Description
}

func (s pkgSearchSource) Len() int {
	return len(s.pkgs)
}

// rankPackages returns pkgs matching query, ordered by:
//
//	tier 1: name has query as a case-insensitive prefix
//	tier 2: name contains query (case-insensitive), not at start
//	tier 3: description contains query (case-insensitive)
//
// Within each tier, input order is preserved. Packages that match no tier
// are dropped unless the fuzzy fallback branch matches them (see below).
// An empty or whitespace-only query returns pkgs unchanged.
func rankPackages(pkgs []model.Package, query string) []model.Package {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return pkgs
	}

	var prefix, contains, desc []model.Package
	for _, p := range pkgs {
		n := strings.ToLower(p.Name)
		d := strings.ToLower(p.Description)
		switch classifyMatch(n, d, q) {
		case tierPrefix:
			prefix = append(prefix, p)
		case tierContains:
			contains = append(contains, p)
		case tierDesc:
			desc = append(desc, p)
		}
	}

	if len(prefix)+len(contains)+len(desc) > 0 {
		out := make([]model.Package, 0, len(prefix)+len(contains)+len(desc))
		out = append(out, prefix...)
		out = append(out, contains...)
		out = append(out, desc...)
		return out
	}

	// Fuzzy fallback: no strict match anywhere, so run the library fuzzy
	// matcher over "Name Description" for typo tolerance. Pass the raw
	// query, not the trimmed/lowered q — fuzzy.FindFrom handles case
	// folding internally.
	source := pkgSearchSource{pkgs: pkgs}
	matches := fuzzy.FindFrom(query, source)
	out := make([]model.Package, 0, len(matches))
	for _, m := range matches {
		out = append(out, pkgs[m.Index])
	}
	return out
}

// rankGroupsByName reorders search-result groups by the same tier rules as
// rankPackages, but never drops groups. Non-matching groups are appended at
// the end in their original order. An empty or whitespace-only query returns
// groups unchanged.
//
// Unlike rankPackages, this function has no fuzzy fallback: install-search
// results come from managers' own relevance ranking (aliases, tags, stemmed
// matches) and must not be silently removed client-side. A fuzzy fallback
// would only reorder the non-matching "other" bucket in a less predictable
// way.
func rankGroupsByName(groups []searchResultGroup, query string) []searchResultGroup {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return groups
	}

	var prefix, contains, desc, other []searchResultGroup
	for _, g := range groups {
		var d string
		if len(g.entries) > 0 {
			d = g.entries[0].Description
		}
		switch classifyMatch(strings.ToLower(g.name), strings.ToLower(d), q) {
		case tierPrefix:
			prefix = append(prefix, g)
		case tierContains:
			contains = append(contains, g)
		case tierDesc:
			desc = append(desc, g)
		default:
			other = append(other, g)
		}
	}

	out := make([]searchResultGroup, 0, len(groups))
	out = append(out, prefix...)
	out = append(out, contains...)
	out = append(out, desc...)
	out = append(out, other...)
	return out
}
