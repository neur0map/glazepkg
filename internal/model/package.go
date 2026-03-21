package model

import "time"

type Source string

const (
	SourcePacman  Source = "pacman"
	SourceAUR     Source = "aur"
	SourcePip     Source = "pip"
	SourcePipx    Source = "pipx"
	SourceCargo   Source = "cargo"
	SourceGo      Source = "go"
	SourceNpm     Source = "npm"
	SourcePnpm    Source = "pnpm"
	SourceBun     Source = "bun"
	SourceFlatpak Source = "flatpak"
	SourceBrew    Source = "brew"
	SourceApt     Source = "apt"
	SourceDnf     Source = "dnf"
	SourceSnap Source = "snap"
	SourceMacPorts Source = "macports"
	SourcePkgsrc   Source = "pkgsrc"
	SourceOpam     Source = "opam"
	SourceGem      Source = "gem"
	SourcePkg      Source = "pkg"
	SourceComposer Source = "composer"
	SourceMas      Source = "mas"
	SourceApk      Source = "apk"
	SourceNix      Source = "nix"
	SourceConda    Source = "conda"
	SourceLuarocks Source = "luarocks"
	SourceXbps     Source = "xbps"
	SourcePortage  Source = "portage"
	SourceGuix     Source = "guix"
)

type Package struct {
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	Source        Source    `json:"source"`
	Description   string    `json:"description,omitempty"`
	Size          string    `json:"size,omitempty"`
	Repository    string    `json:"repository,omitempty"`
	DependsOn     []string  `json:"depends_on,omitempty"`
	RequiredBy    []string  `json:"required_by,omitempty"`
	InstalledAt   time.Time `json:"installed_at"`
	LatestVersion string    `json:"-"` // not persisted, populated at runtime
	Location      string    `json:"-"`
	SizeBytes     int64     `json:"size_bytes,omitempty"`
}

// Key returns a unique identifier for this package across all managers.
func (p Package) Key() string {
	return string(p.Source) + ":" + p.Name
}

type Snapshot struct {
	Timestamp time.Time          `json:"timestamp"`
	Packages  map[string]Package `json:"packages"` // keyed by Key()
}

type DiffEntry struct {
	Old Package
	New Package
}

type Diff struct {
	Added    []Package
	Removed  []Package
	Upgraded []DiffEntry
}

// ComputeDiff returns the difference between an old snapshot and a new one.
func ComputeDiff(old, new *Snapshot) Diff {
	var d Diff

	for key, pkg := range new.Packages {
		oldPkg, exists := old.Packages[key]
		if !exists {
			d.Added = append(d.Added, pkg)
		} else if oldPkg.Version != pkg.Version {
			d.Upgraded = append(d.Upgraded, DiffEntry{Old: oldPkg, New: pkg})
		}
	}

	for key, pkg := range old.Packages {
		if _, exists := new.Packages[key]; !exists {
			d.Removed = append(d.Removed, pkg)
		}
	}

	return d
}
