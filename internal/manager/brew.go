package manager

import (
	"encoding/json"
	"os/exec"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Brew struct{}

func (b *Brew) Name() model.Source { return model.SourceBrew }

func (b *Brew) Available() bool { return commandExists("brew") }

// brewInfo is the shared JSON structure from `brew info --json=v2 --installed`.
type brewInfo struct {
	Formulae []brewFormula `json:"formulae"`
}

type brewFormula struct {
	Name         string        `json:"name"`
	Desc         string        `json:"desc"`
	Dependencies []string      `json:"dependencies"`
	Installed    []brewInstall `json:"installed"`
}

type brewInstall struct {
	Version               string `json:"version"`
	InstalledOnRequest    bool   `json:"installed_on_request"`
	InstalledAsDependency bool   `json:"installed_as_dependency"`
}

func fetchBrewInfo() (*brewInfo, error) {
	out, err := exec.Command("brew", "info", "--json=v2", "--installed").Output()
	if err != nil {
		return nil, err
	}
	var info brewInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (b *Brew) Scan() ([]model.Package, error) {
	info, err := fetchBrewInfo()
	if err != nil {
		return nil, err
	}

	// Build a map of formula name → formula for quick lookup
	formulaMap := make(map[string]*brewFormula, len(info.Formulae))
	for i := range info.Formulae {
		formulaMap[info.Formulae[i].Name] = &info.Formulae[i]
	}

	// Build reverse-dependency map: dep name → list of explicit packages that need it
	requiredBy := make(map[string][]string)
	for _, f := range info.Formulae {
		if len(f.Installed) == 0 || !f.Installed[0].InstalledOnRequest {
			continue
		}
		for _, dep := range f.Dependencies {
			requiredBy[dep] = append(requiredBy[dep], f.Name)
		}
	}

	var pkgs []model.Package
	for _, f := range info.Formulae {
		if len(f.Installed) == 0 {
			continue
		}
		inst := f.Installed[0]

		if inst.InstalledOnRequest {
			// Explicit package → goes in "brew" tab
			pkgs = append(pkgs, model.Package{
				Name:        f.Name,
				Version:     inst.Version,
				Description: f.Desc,
				Source:      model.SourceBrew,
				InstalledAt: time.Now(),
			})
		} else {
			// Auto-installed dependency → goes in "brew-deps" tab
			pkg := model.Package{
				Name:        f.Name,
				Version:     inst.Version,
				Description: f.Desc,
				Source:      model.SourceBrewDeps,
				RequiredBy:  requiredBy[f.Name],
				InstalledAt: time.Now(),
			}
			pkgs = append(pkgs, pkg)
		}
	}
	return pkgs, nil
}

func (b *Brew) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command("brew", "outdated", "--json").Output()
	if err != nil || len(out) == 0 {
		return nil
	}

	var outdated struct {
		Formulae []struct {
			Name           string   `json:"name"`
			CurrentVersion string   `json:"current_version"`
			InstalledVersions []string `json:"installed_versions"`
		} `json:"formulae"`
	}
	if err := json.Unmarshal(out, &outdated); err == nil {
		updates := make(map[string]string)
		for _, f := range outdated.Formulae {
			updates[f.Name] = f.CurrentVersion
		}
		return updates
	}
	return nil
}

func (b *Brew) Describe(pkgs []model.Package) map[string]string {
	// Descriptions are already populated during Scan from the same JSON.
	// This is a fallback for the description cache.
	info, err := fetchBrewInfo()
	if err != nil {
		return nil
	}

	descs := make(map[string]string, len(info.Formulae))
	for _, f := range info.Formulae {
		if f.Desc != "" {
			descs[f.Name] = f.Desc
		}
	}
	return descs
}
