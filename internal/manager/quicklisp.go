package manager

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// Quicklisp inventories Common Lisp libraries installed via Quicklisp. There is
// no per-package CLI (installs happen in the REPL), so this manager is read-only
// and scans the installed-release marker files on disk.
type Quicklisp struct{}

func (q *Quicklisp) Name() model.Source { return model.SourceQuicklisp }

func (q *Quicklisp) Available() bool {
	info, err := os.Stat(quicklispReleasesDir())
	return err == nil && info.IsDir()
}

func (q *Quicklisp) Scan() ([]model.Package, error) {
	entries, err := os.ReadDir(quicklispReleasesDir())
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	data, _ := os.ReadFile(filepath.Join(qlHome(), "dists", "quicklisp", "distinfo.txt"))
	return parseQuicklispReleases(names, parseDistVersion(data)), nil
}

func qlHome() string {
	if home := os.Getenv("QUICKLISP_HOME"); home != "" {
		return home
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "quicklisp")
}

func quicklispReleasesDir() string {
	return filepath.Join(qlHome(), "dists", "quicklisp", "installed", "releases")
}

// parseQuicklispReleases turns release marker filenames into packages. Each
// installed release is a "<name>.txt" file; the version is the shared dist
// version. Hidden entries and non-".txt" files are skipped.
func parseQuicklispReleases(names []string, version string) []model.Package {
	var pkgs []model.Package
	now := time.Now()
	for _, name := range names {
		if strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".txt") {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:        strings.TrimSuffix(name, ".txt"),
			Version:     version,
			Source:      model.SourceQuicklisp,
			Repository:  "quicklisp",
			InstalledAt: now,
		})
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Name < pkgs[j].Name })
	return pkgs
}

// parseDistVersion reads the "version:" field from a Quicklisp distinfo.txt.
func parseDistVersion(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "version:"); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}
