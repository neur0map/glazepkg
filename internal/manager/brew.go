package manager

import (
	"bufio"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

	// Get cellar sizes in one call
	sizes := brewCellarSizes()

	var pkgs []model.Package
	for _, f := range info.Formulae {
		if len(f.Installed) == 0 {
			continue
		}
		inst := f.Installed[0]
		if !inst.InstalledOnRequest {
			continue // skip auto-installed dependencies
		}
		sizeBytes := sizes[f.Name]
		sizeStr := FormatBytes(sizeBytes)

		pkgs = append(pkgs, model.Package{
			Name:        f.Name,
			Version:     inst.Version,
			Description: f.Desc,
			Source:      model.SourceBrew,
			DependsOn:   f.Dependencies,
			RequiredBy:  requiredBy[f.Name],
			InstalledAt: time.Now(),
			Size:        sizeStr,
			SizeBytes:   sizeBytes,
		})
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
			Name              string   `json:"name"`
			CurrentVersion    string   `json:"current_version"`
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

func (b *Brew) ListDependencies(pkgs []model.Package) map[string][]string {
	info, err := fetchBrewInfo()
	if err != nil {
		return nil
	}

	formulaMap := make(map[string]*brewFormula, len(info.Formulae))
	for i := range info.Formulae {
		formulaMap[info.Formulae[i].Name] = &info.Formulae[i]
	}

	deps := make(map[string][]string, len(pkgs))
	for _, p := range pkgs {
		if f, ok := formulaMap[p.Name]; ok {
			deps[p.Name] = f.Dependencies
		}
	}
	return deps
}

// brewCellarSizes runs a single `du -sk` on the cellar directory and returns
// a map of formula name → size in bytes.
func brewCellarSizes() map[string]int64 {
	cellarOut, err := exec.Command("brew", "--cellar").Output()
	if err != nil {
		return nil
	}
	cellar := strings.TrimSpace(string(cellarOut))

	entries, err := filepath.Glob(filepath.Join(cellar, "*"))
	if err != nil || len(entries) == 0 {
		return nil
	}

	args := append([]string{"-sk"}, entries...)
	out, err := exec.Command("du", args...).Output()
	if err != nil {
		return nil
	}

	sizes := make(map[string]int64)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		kb, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}
		name := filepath.Base(fields[1])
		sizes[name] = kb * 1024 // convert KB to bytes
	}
	return sizes
}

func (b *Brew) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("brew", "upgrade", name)
}

func (b *Brew) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("brew", "uninstall", name)
}

func (b *Brew) Search(query string) ([]model.Package, error) {
	// Run: brew search --formulae query, which returns names one per line
	out, err := exec.Command("brew", "search", "--formulae", query).Output()
	if err != nil {
		return nil, err
	}
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name == "" || strings.HasPrefix(name, "==>") {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:   name,
			Source: model.SourceBrew,
		})
	}
	return pkgs, nil
}

func (b *Brew) InstallCmd(name string) *exec.Cmd {
	return exec.Command("brew", "install", name)
}
