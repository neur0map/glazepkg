package manager

import (
	"encoding/json"
	"os/exec"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Conda struct{}

func (c *Conda) Name() model.Source { return model.SourceConda }

func (c *Conda) Available() bool {
	return commandExists("conda") || commandExists("mamba")
}

func (c *Conda) condaCmd() string {
	if commandExists("mamba") {
		return "mamba"
	}
	return "conda"
}

func (c *Conda) Scan() ([]model.Package, error) {
	out, err := exec.Command(c.condaCmd(), "list", "--json").Output()
	if err != nil {
		return nil, err
	}

	var entries []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Channel string `json:"channel"`
	}
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, err
	}

	pkgs := make([]model.Package, 0, len(entries))
	for _, e := range entries {
		pkgs = append(pkgs, model.Package{
			Name:        e.Name,
			Version:     e.Version,
			Source:      model.SourceConda,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (c *Conda) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command(c.condaCmd(), "update", "--all", "--dry-run", "--json").Output()
	if err != nil && len(out) == 0 {
		return nil
	}

	var result struct {
		Actions struct {
			Link []struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"LINK"`
			Unlink []struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"UNLINK"`
		} `json:"actions"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil
	}

	// Build a map of currently installed versions from UNLINK
	installed := make(map[string]string)
	for _, p := range result.Actions.Unlink {
		installed[p.Name] = p.Version
	}

	// LINK entries with a different version than UNLINK are upgrades
	updates := make(map[string]string)
	for _, p := range result.Actions.Link {
		if oldVer, ok := installed[p.Name]; ok && oldVer != p.Version {
			updates[p.Name] = p.Version
		}
	}
	return updates
}
