package manager

import (
	"bufio"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type BrewCask struct{}

func (b *BrewCask) Name() model.Source { return model.SourceBrewCask }

func (b *BrewCask) Available() bool { return commandExists("brew") }

func (b *BrewCask) Scan() ([]model.Package, error) {
	info, err := fetchBrewInfo()
	if err != nil {
		return nil, err
	}
	return brewCaskPackages(info, brewCaskroom(), time.Now()), nil
}

func brewCaskroom() string {
	out, err := exec.Command("brew", "--caskroom").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func brewCaskPackages(info *brewInfo, caskroom string, now time.Time) []model.Package {
	if info == nil {
		return nil
	}

	var pkgs []model.Package
	for _, c := range info.Casks {
		token := strings.TrimSpace(c.Token)
		if token == "" {
			continue
		}

		if c.Installed == nil {
			continue
		}

		version := brewCaskInstalledVersion(c)
		if version == "" {
			continue
		}

		installedAt := now
		if c.InstalledTime > 0 {
			installedAt = time.Unix(c.InstalledTime, 0)
		}

		location := ""
		if caskroom != "" {
			location = filepath.Join(caskroom, token, version)
		}

		pkgs = append(pkgs, model.Package{
			Name:        token,
			Version:     version,
			Description: c.Desc,
			Source:      model.SourceBrewCask,
			InstalledAt: installedAt,
			Location:    location,
		})
	}
	return pkgs
}

func (b *BrewCask) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command("brew", "outdated", "--json=v2").Output()
	if err != nil || len(out) == 0 {
		return nil
	}
	return parseBrewCaskUpdates(out)
}

func parseBrewCaskUpdates(out []byte) map[string]string {
	var outdated struct {
		Casks []struct {
			Name              string   `json:"name"`
			CurrentVersion    string   `json:"current_version"`
			InstalledVersions []string `json:"installed_versions"`
		} `json:"casks"`
	}
	if err := json.Unmarshal(out, &outdated); err != nil {
		return nil
	}

	updates := make(map[string]string, len(outdated.Casks))
	for _, c := range outdated.Casks {
		if c.Name == "" || c.CurrentVersion == "" {
			continue
		}
		updates[c.Name] = c.CurrentVersion
	}
	return updates
}

func (b *BrewCask) Describe(pkgs []model.Package) map[string]string {
	info, err := fetchBrewInfo()
	if err != nil {
		return nil
	}

	descs := make(map[string]string, len(info.Casks))
	for _, c := range info.Casks {
		if brewCaskInstalledVersion(c) == "" || c.Desc == "" {
			continue
		}
		descs[c.Token] = c.Desc
	}
	return descs
}

func brewCaskInstalledVersion(c brewCask) string {
	if c.Installed == nil {
		return ""
	}
	version := strings.TrimSpace(*c.Installed)
	if version != "" {
		return version
	}
	return strings.TrimSpace(c.Version)
}

func (b *BrewCask) Search(query string) ([]model.Package, error) {
	out, err := exec.Command("brew", "search", "--casks", query).Output()
	if err != nil {
		return nil, err
	}

	return parseBrewCaskSearch(string(out)), nil
}

func parseBrewCaskSearch(out string) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name == "" || strings.HasPrefix(name, "==>") {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:   name,
			Source: model.SourceBrewCask,
		})
	}
	return pkgs
}

func (b *BrewCask) InstallCmd(name string) *exec.Cmd {
	return exec.Command("brew", "install", "--cask", name)
}

func (b *BrewCask) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("brew", "upgrade", "--cask", name)
}

func (b *BrewCask) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("brew", "uninstall", "--cask", name)
}
