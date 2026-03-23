package manager

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func (c *Conda) ListDependencies(pkgs []model.Package) map[string][]string {
	// Read dependency info from conda-meta JSON files in the active prefix
	prefix := os.Getenv("CONDA_PREFIX")
	if prefix == "" {
		return nil
	}
	metaDir := filepath.Join(prefix, "conda-meta")
	entries, err := os.ReadDir(metaDir)
	if err != nil {
		return nil
	}

	// Build a map of package name → deps from conda-meta files
	type metaInfo struct {
		Name    string   `json:"name"`
		Depends []string `json:"depends"`
	}

	metaMap := make(map[string][]string)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(metaDir, e.Name()))
		if err != nil {
			continue
		}
		var m metaInfo
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		var depNames []string
		for _, d := range m.Depends {
			// Format: "name version build" — take just the name
			name := strings.Fields(d)[0]
			if name != "" {
				depNames = append(depNames, name)
			}
		}
		metaMap[m.Name] = depNames
	}

	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		if d, ok := metaMap[pkg.Name]; ok {
			deps[pkg.Name] = d
		}
	}
	return deps
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

func (c *Conda) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command(c.condaCmd(), "update", "--yes", name)
}

func (c *Conda) RemoveCmd(name string) *exec.Cmd {
	return exec.Command(c.condaCmd(), "remove", "--yes", name)
}
