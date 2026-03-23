package manager

import "github.com/neur0map/glazepkg/internal/model"

// Searcher is implemented by managers that can search for available packages.
type Searcher interface {
	Search(query string) ([]model.Package, error)
}
