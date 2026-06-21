package manager

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// Gvm lists Go toolchains installed via moovweb/gvm (https://github.com/moovweb/gvm).
// gvm is a sourced shell function, not a PATH binary, so detection and scanning
// are filesystem-based against <root>/gos. Scan-only: gvm cannot be exec'd.
type Gvm struct{}

func (g *Gvm) Name() model.Source { return model.SourceGvm }

func (g *Gvm) Available() bool {
	info, err := os.Stat(filepath.Join(gvmRoot(), "gos"))
	return err == nil && info.IsDir()
}

func (g *Gvm) Scan() ([]model.Package, error) {
	root := gvmRoot()
	entries, err := os.ReadDir(filepath.Join(root, "gos"))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return parseGvmVersions(root, names), nil
}

// gvmRoot resolves the gvm install root: $GVM_ROOT, else $HOME/.gvm.
func gvmRoot() string {
	if root := os.Getenv("GVM_ROOT"); root != "" {
		return root
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".gvm")
}

// parseGvmVersions maps gos entry names to packages, skipping the "system"
// pseudo-version and dotfiles. Sorted by Name.
func parseGvmVersions(root string, names []string) []model.Package {
	now := time.Now()
	pkgs := make([]model.Package, 0, len(names))
	for _, name := range names {
		if name == "system" || strings.HasPrefix(name, ".") {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     strings.TrimPrefix(name, "go"),
			Source:      model.SourceGvm,
			Location:    filepath.Join(root, "gos", name),
			InstalledAt: now,
		})
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Name < pkgs[j].Name })
	return pkgs
}
