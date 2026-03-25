package manager

import (
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Composer struct{}

func (c *Composer) Name() model.Source { return model.SourceComposer }

func (c *Composer) Available() bool { return commandExists("composer") }

func (c *Composer) Scan() ([]model.Package, error) {
	out, err := exec.Command("composer", "global", "show", "--format=json").Output()
	if err != nil {
		return nil, err
	}

	var result struct {
		Installed []struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description"`
		} `json:"installed"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}

	pkgs := make([]model.Package, 0, len(result.Installed))
	for _, p := range result.Installed {
		pkgs = append(pkgs, model.Package{
			Name:        p.Name,
			Version:     strings.TrimPrefix(p.Version, "v"),
			Description: p.Description,
			Source:      model.SourceComposer,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (c *Composer) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("composer", "global", "remove", name)
}

func (c *Composer) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command("composer", "global", "outdated", "--format=json").Output()
	if err != nil && len(out) == 0 {
		return nil
	}

	var result struct {
		Installed []struct {
			Name   string `json:"name"`
			Latest string `json:"latest"`
		} `json:"installed"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil
	}

	updates := make(map[string]string)
	for _, p := range result.Installed {
		if p.Latest != "" {
			updates[p.Name] = strings.TrimPrefix(p.Latest, "v")
		}
	}
	return updates
}

func (c *Composer) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		out, err := exec.Command("composer", "global", "show", pkg.Name, "--format=json").Output()
		if err != nil {
			continue
		}
		var info struct {
			Requires map[string]string `json:"requires"`
		}
		if err := json.Unmarshal(out, &info); err != nil {
			continue
		}
		var pkgDeps []string
		for name := range info.Requires {
			pkgDeps = append(pkgDeps, name)
		}
		deps[pkg.Name] = pkgDeps
	}
	return deps
}

func (c *Composer) Describe(pkgs []model.Package) map[string]string {
	// Descriptions are already populated during Scan.
	return nil
}
