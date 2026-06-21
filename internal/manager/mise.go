package manager

import (
	"encoding/json"
	"os/exec"
	"sort"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// Mise manages tools installed via mise (https://mise.jdx.dev/).
type Mise struct{}

func (m *Mise) Name() model.Source { return model.SourceMise }

func (m *Mise) Available() bool { return commandExists("mise") }

func (m *Mise) Scan() ([]model.Package, error) {
	out, err := exec.Command("mise", "ls", "--json").Output()
	if err != nil {
		return nil, err
	}
	return parseMiseList(out)
}

type miseEntry struct {
	Version     string `json:"version"`
	InstallPath string `json:"install_path"`
	Installed   bool   `json:"installed"`
	Active      bool   `json:"active"`
}

// parseMiseList turns "mise ls --json" (tool name -> versions) into one
// package per tool, preferring the active version and falling back to the
// last installed one.
func parseMiseList(data []byte) ([]model.Package, error) {
	var raw map[string][]miseEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	pkgs := make([]model.Package, 0, len(raw))
	for tool, entries := range raw {
		var chosen *miseEntry
		for i := range entries {
			if entries[i].Active {
				chosen = &entries[i]
				break
			}
		}
		if chosen == nil {
			for i := range entries {
				if entries[i].Installed {
					chosen = &entries[i]
				}
			}
		}
		if chosen == nil {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:        tool,
			Version:     chosen.Version,
			Source:      model.SourceMise,
			Location:    chosen.InstallPath,
			InstalledAt: time.Now(),
		})
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Name < pkgs[j].Name })
	return pkgs, nil
}

func (m *Mise) CheckUpdates(_ []model.Package) map[string]string {
	out, err := exec.Command("mise", "outdated", "--json").Output()
	if err != nil {
		return map[string]string{}
	}
	return parseMiseOutdated(out)
}

// parseMiseOutdated turns "mise outdated --json" (tool name -> info) into a
// name -> latest version map.
func parseMiseOutdated(data []byte) map[string]string {
	var raw map[string]struct {
		Latest string `json:"latest"`
	}
	updates := make(map[string]string)
	if err := json.Unmarshal(data, &raw); err != nil {
		return updates
	}
	for name, info := range raw {
		if info.Latest != "" {
			updates[name] = info.Latest
		}
	}
	return updates
}

func (m *Mise) InstallCmd(name string) *exec.Cmd {
	return exec.Command("mise", "use", "-g", name)
}

func (m *Mise) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("mise", "upgrade", name)
}

func (m *Mise) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("mise", "uninstall", name)
}
