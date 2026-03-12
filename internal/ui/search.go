package ui

import (
	"github.com/sahilm/fuzzy"

	"github.com/neur0map/glazepkg/internal/model"
)

// pkgSearchSource implements fuzzy.Source for package fuzzy matching.
type pkgSearchSource struct {
	pkgs []model.Package
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
