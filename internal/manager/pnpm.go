package manager

import (
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Pnpm struct{}

func (n *Pnpm) Name() model.Source { return model.SourcePnpm }

func (n *Pnpm) Available() bool { return commandExists("pnpm") }

func (n *Pnpm) Scan() ([]model.Package, error) {
	out, err := exec.Command("pnpm", "ls", "-g", "--json", "--depth=0").Output()
	if err != nil {
		return nil, err
	}

	var stores []struct {
		Path         string                 `json:"path"`
		Private      bool                   `json:"private"`
		Dependencies map[string]struct {
			Version string `json:"version"`
			From    string `json:"from,omitempty"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(out, &stores); err != nil {
		return nil, err
	}

	pkgs := make([]model.Package, 0)
	// `pnpm` supports multiple local stores - for now, flatten the list instead of grouping
	for _, store := range stores {
		for name, dep := range store.Dependencies {
			pkgs = append(pkgs, model.Package{
				Name:        name,
				Version:     dep.Version,
				Source:      model.SourcePnpm,
				InstalledAt: time.Now(),
				Location:    store.Path,
			})
		}
	}
	return pkgs, nil
}

func (n *Pnpm) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command("pnpm", "outdated", "-g", "--json").Output()
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

func (n *Pnpm) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		out, err := exec.Command("pnpm", "info", pkg.Name, "description").Output()
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
