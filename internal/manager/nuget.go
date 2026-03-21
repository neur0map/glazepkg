package manager

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// Nuget surfaces packages from the local NuGet global package cache (~/.nuget/packages).
// This is intentionally cross-platform: the cache exists wherever the .NET SDK is installed,
// including macOS and Linux. Do not add a runtime.GOOS == "windows" guard here.
type Nuget struct{}

func (n *Nuget) Name() model.Source { return model.SourceNuget }

// Available returns true when the NuGet global package cache directory exists.
func (n *Nuget) Available() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".nuget", "packages"))
	return err == nil
}

// Scan enumerates packages in ~/.nuget/packages.
// Each subdirectory is a package name; its subdirectories are installed versions.
// The latest (semver-highest) version is reported for each package.
func (n *Nuget) Scan() ([]model.Package, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	packagesDir := filepath.Join(home, ".nuget", "packages")
	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		packagePath := filepath.Join(packagesDir, name)

		versions, err := os.ReadDir(packagePath)
		if err != nil {
			continue
		}

		var versionNames []string
		for _, v := range versions {
			if v.IsDir() {
				versionNames = append(versionNames, v.Name())
			}
		}
		if len(versionNames) == 0 {
			continue
		}

		// Find the semver-highest version — lexicographic sort breaks for "10.x" vs "9.x", hence nugetSemverGT.
		latest := versionNames[0]
		for _, v := range versionNames[1:] {
			if nugetSemverGT(v, latest) {
				latest = v
			}
		}

		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     latest,
			Source:      model.SourceNuget,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

// nugetSemverGT returns true if version a is greater than version b.
// Compares dot-separated numeric parts; handles pre-release suffixes (stripped).
func nugetSemverGT(a, b string) bool {
	return nugetCompare(a, b) > 0
}

func nugetCompare(a, b string) int {
	// Strip prerelease suffix first so "6.0.0-preview.5" doesn't gain a phantom 4th segment.
	aBase, aPrerel, _ := strings.Cut(a, "-")
	bBase, bPrerel, _ := strings.Cut(b, "-")

	aParts := strings.Split(aBase, ".")
	bParts := strings.Split(bBase, ".")
	n := len(aParts)
	if len(bParts) > n {
		n = len(bParts)
	}
	for i := 0; i < n; i++ {
		ai, bi := 0, 0
		if i < len(aParts) {
			ai, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bi, _ = strconv.Atoi(bParts[i])
		}
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
	}
	// Base versions equal: stable release beats prerelease.
	if aPrerel == "" && bPrerel != "" {
		return 1
	}
	if aPrerel != "" && bPrerel == "" {
		return -1
	}
	return 0
}
