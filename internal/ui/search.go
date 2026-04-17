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

// fuzzyFilter returns packages matching the query using fuzzy matching.
// If query is empty, returns all packages.
func fuzzyFilter(pkgs []model.Package, query string) []model.Package {
	if query == "" {
		return pkgs
	}

	source := pkgSearchSource{pkgs: pkgs}
	matches := fuzzy.FindFrom(query, source)

	result := make([]model.Package, 0, len(matches))
	for _, m := range matches {
		result = append(result, pkgs[m.Index])
	}
	return result
}
