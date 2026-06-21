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

// nodeBundled ships with the Node.js runtime. These show up in `npm ls -g`
// but self-upgrading them via npm is discouraged and, when node came from a
// system manager like Homebrew, harmful, so gpk hides them (issue #49).
var nodeBundled = map[string]bool{"npm": true, "corepack": true}

func (n *Npm) Scan() ([]model.Package, error) {
	out, err := exec.Command("npm", "list", "-g", "--json", "--depth=0").Output()
	// npm exits 1 on peer dep issues but still prints JSON; only empty is fatal.
	if err != nil && out == nil {
		return nil, err
	}
	return parseNpmList(out)
}

func parseNpmList(data []byte) ([]model.Package, error) {
	var result struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	pkgs := make([]model.Package, 0, len(result.Dependencies))
	for name, dep := range result.Dependencies {
		if nodeBundled[name] {
			continue
		}
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

func (n *Npm) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("npm", "install", "-g", name+"@latest")
}

func (n *Npm) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("npm", "uninstall", "-g", name)
}

func (n *Npm) Search(query string) ([]model.Package, error) {
	// Run: npm search query --json
	out, err := exec.Command("npm", "search", query, "--json").Output()
	if err != nil {
		return nil, err
	}
	var results []struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(out, &results); err != nil {
		return nil, err
	}
	var pkgs []model.Package
	for _, r := range results {
		pkgs = append(pkgs, model.Package{
			Name:        r.Name,
			Version:     r.Version,
			Source:      model.SourceNpm,
			Description: r.Description,
		})
	}
	return pkgs, nil
}

func (n *Npm) InstallCmd(name string) *exec.Cmd {
	return exec.Command("npm", "install", "-g", name)
}
