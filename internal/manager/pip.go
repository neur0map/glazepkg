package manager

import (
	"bufio"
	"encoding/json"
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

	pkgs := make([]model.Package, 0, len(entries))
	for _, e := range entries {
		pkgs = append(pkgs, model.Package{
			Name:        e.Name,
			Version:     e.Version,
			Source:      model.SourcePip,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
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
