package manager

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/neur0map/glazepkg/internal/model"
)

type Go struct{}

func (g *Go) Name() model.Source { return model.SourceGo }

func (g *Go) Available() bool {
	dir := goBinDir()
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

func (g *Go) Scan() ([]model.Package, error) {
	dir := goBinDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:        e.Name(),
			Version:     "",
			Source:      model.SourceGo,
			InstalledAt: info.ModTime(),
		})
	}
	return pkgs, nil
}

func (g *Go) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		// Best-effort: run `go doc <name>` and grab the first non-empty comment line
		out, err := exec.Command("go", "doc", pkg.Name).Output()
		if err != nil {
			continue
		}
		// The package doc comment is typically the first paragraph of output.
		// Take the first non-empty line that doesn't start with "package" or "func".
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "package ") || strings.HasPrefix(line, "func ") ||
				strings.HasPrefix(line, "var ") || strings.HasPrefix(line, "type ") ||
				strings.HasPrefix(line, "const ") {
				continue
			}
			descs[pkg.Name] = line
			break
		}
	}
	return descs
}

func goBinDir() string {
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return filepath.Join(gopath, "bin")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "go", "bin")
}
