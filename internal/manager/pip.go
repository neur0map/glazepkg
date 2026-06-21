package manager

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Pip struct{}

func (p *Pip) Name() model.Source { return model.SourcePip }

func (p *Pip) Available() bool { return commandExists("pip") }

func (p *Pip) Scan() ([]model.Package, error) {
	// --not-required filters out packages that are dependencies of other packages,
	// showing only top-level user-intended installs.
	out, err := exec.Command("pip", "list", "--not-required", "--format=json").Output()
	if err != nil {
		return nil, err
	}

	var entries []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, err
	}

	// Classify user vs global installs. Skipped inside a virtualenv, where every
	// package is env-scoped and a user/global label would be misleading (#12).
	var userSet map[string]bool
	if os.Getenv("VIRTUAL_ENV") == "" {
		if uout, uerr := exec.Command("pip", "list", "--user", "--format=json").Output(); uerr == nil {
			userSet = parsePipNameSet(uout)
		}
	}

	pkgs := make([]model.Package, 0, len(entries))
	for _, e := range entries {
		pkgs = append(pkgs, model.Package{
			Name:        e.Name,
			Version:     e.Version,
			Source:      model.SourcePip,
			Scope:       pipScope(e.Name, userSet),
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

// parsePipNameSet parses `pip list --format=json` into a set of normalized names.
func parsePipNameSet(data []byte) map[string]bool {
	var entries []struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(data, &entries) != nil {
		return nil
	}
	set := make(map[string]bool, len(entries))
	for _, e := range entries {
		set[normalizePipName(e.Name)] = true
	}
	return set
}

// pipScope returns "user" when name is in the user-site set, "global" otherwise.
// Returns "" when scope was not determined (no set, e.g. inside a venv).
func pipScope(name string, userSet map[string]bool) string {
	if userSet == nil {
		return ""
	}
	if userSet[normalizePipName(name)] {
		return "user"
	}
	return "global"
}

func normalizePipName(s string) string {
	return strings.ReplaceAll(strings.ToLower(s), "_", "-")
}

func (p *Pip) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command("pip", "list", "--outdated", "--format=json").Output()
	if err != nil || len(out) == 0 {
		return nil
	}

	var entries []struct {
		Name          string `json:"name"`
		LatestVersion string `json:"latest_version"`
	}
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil
	}

	updates := make(map[string]string)
	for _, e := range entries {
		updates[e.Name] = e.LatestVersion
	}
	return updates
}

func (p *Pip) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		out, err := exec.Command("pip", "show", pkg.Name).Output()
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Requires:") {
				req := strings.TrimSpace(strings.TrimPrefix(line, "Requires:"))
				if req == "" {
					deps[pkg.Name] = nil
				} else {
					var pkgDeps []string
					for _, d := range strings.Split(req, ", ") {
						d = strings.TrimSpace(d)
						if d != "" {
							pkgDeps = append(pkgDeps, d)
						}
					}
					deps[pkg.Name] = pkgDeps
				}
				break
			}
		}
	}
	return deps
}

func (p *Pip) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		out, err := exec.Command("pip", "show", pkg.Name).Output()
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Summary:") {
				descs[pkg.Name] = strings.TrimSpace(strings.TrimPrefix(line, "Summary:"))
				break
			}
		}
	}
	return descs
}

func (p *Pip) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("pip", "install", "--upgrade", name)
}

func (p *Pip) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("pip", "uninstall", "-y", name)
}

func (p *Pip) Search(query string) ([]model.Package, error) {
	// pip doesn't have a real search. Use pip index versions for exact match.
	out, err := exec.Command("pip", "index", "versions", query).Output()
	if err != nil {
		return nil, nil // no results, not an error
	}
	// Output: "package-name (latest-version)"
	// Then: "Available versions: 1.0, 0.9, ..."
	line := strings.SplitN(string(out), "\n", 2)[0]
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}
	// Parse "name (version)"
	name := query
	version := ""
	if idx := strings.Index(line, "("); idx > 0 {
		name = strings.TrimSpace(line[:idx])
		end := strings.Index(line, ")")
		if end > idx {
			version = line[idx+1 : end]
		}
	}
	return []model.Package{{
		Name:    name,
		Version: version,
		Source:  model.SourcePip,
	}}, nil
}

func (p *Pip) InstallCmd(name string) *exec.Cmd {
	return exec.Command("pip", "install", name)
}
