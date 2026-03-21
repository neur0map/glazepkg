package manager

import (
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Npm struct{}

func (n *Npm) Name() model.Source { return model.SourceNpm }

func (n *Npm) Available() bool { return commandExists("npm") }

func (n *Npm) Scan() ([]model.Package, error) {
	out, err := exec.Command("npm", "list", "-g", "--json", "--depth=0").Output()
	if err != nil {
		// npm returns exit code 1 if there are peer dep issues, but still outputs JSON
		if out == nil {
			return nil, err
		}
	}

	var result struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	pkgs := make([]model.Package, 0, len(result.Dependencies))
	for name, dep := range result.Dependencies {
		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     dep.Version,
			Source:      model.SourceNpm,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (n *Npm) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command("npm", "outdated", "-g", "--json").Output()
	if err != nil && out == nil {
		return nil
	}
	if len(out) == 0 {
		return nil
	}

	var outdated map[string]struct {
		Latest string `json:"latest"`
	}
	if err := json.Unmarshal(out, &outdated); err != nil {
		return nil
	}

	updates := make(map[string]string)
	for name, info := range outdated {
		if info.Latest != "" {
			updates[name] = info.Latest
		}
	}
	return updates
}

func (n *Npm) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		out, err := exec.Command("npm", "info", pkg.Name, "dependencies", "--json").Output()
		if err != nil || len(out) == 0 {
			deps[pkg.Name] = nil
			continue
		}
		var depMap map[string]string
		if err := json.Unmarshal(out, &depMap); err != nil {
			deps[pkg.Name] = nil
			continue
		}
		var pkgDeps []string
		for name := range depMap {
			pkgDeps = append(pkgDeps, name)
		}
		deps[pkg.Name] = pkgDeps
	}
	return deps
}

func (n *Npm) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		out, err := exec.Command("npm", "info", pkg.Name, "description").Output()
		if err != nil {
			continue
		}
		desc := strings.TrimSpace(string(out))
		if desc != "" {
			descs[pkg.Name] = desc
		}
	}
	return descs
}
