package manager

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// Aqua manages CLI tools declared in aqua configuration files
// (https://aquaproj.github.io/). aqua is config-file driven: the set of
// installed tools is whatever is listed in the nearest aqua.yaml plus the
// global configuration files named by $AQUA_GLOBAL_CONFIG.
type Aqua struct{}

func (a *Aqua) Name() model.Source { return model.SourceAqua }

// Available requires a configuration aqua can actually act on. With aqua
// installed but no config, its own commands exit with "configuration file isn't
// found", which would fail every `gpk upgrade` for anyone who installed aqua
// for project-local use and ran gpk from elsewhere.
func (a *Aqua) Available() bool {
	return commandExists("aqua") && aquaConfigPresent()
}

// aquaConfigNames are the file names aqua's own config finder accepts.
var aquaConfigNames = [...]string{"aqua.yaml", "aqua.yml", ".aqua.yaml", ".aqua.yml"}

// aquaConfigPresent reports whether AQUA_GLOBAL_CONFIG names an existing file,
// or a config sits in the working directory or any parent, which is the same
// pair of places the aqua commands gpk builds will look.
func aquaConfigPresent() bool {
	for _, p := range filepath.SplitList(os.Getenv("AQUA_GLOBAL_CONFIG")) {
		if p == "" {
			continue
		}
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return true
		}
	}
	dir, err := os.Getwd()
	if err != nil {
		return false
	}
	for {
		for _, name := range aquaConfigNames {
			if info, err := os.Stat(filepath.Join(dir, name)); err == nil && !info.IsDir() {
				return true
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

// Scan lists the tools aqua manages via `aqua list --installed --all`, which
// enumerates the packages declared in the nearest aqua.yaml and every global
// configuration file. aqua's "installed" set is the config, not the download
// cache, so this is the honest source of top-level tools.
func (a *Aqua) Scan() ([]model.Package, error) {
	out, err := exec.Command("aqua", "list", "--installed", "--all").Output()
	if err != nil {
		return nil, err
	}
	return ParseAquaList(out), nil
}

// ParseAquaList parses `aqua list --installed --all` output. Each line is
// "<package name>\t<version>\t<registry name>" (see aqua's list_installed.go).
func ParseAquaList(data []byte) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if fields[0] == "" {
			continue
		}
		pkg := model.Package{
			Name:        fields[0],
			Source:      model.SourceAqua,
			InstalledAt: time.Now(),
		}
		if len(fields) > 1 {
			pkg.Version = fields[1]
		}
		if len(fields) > 2 {
			pkg.Repository = fields[2]
		}
		pkgs = append(pkgs, pkg)
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Name < pkgs[j].Name })
	return pkgs
}

// Search filters the packages available across the configured registries.
// `aqua list` prints every registry package as "<registry name>,<package
// name>"; aqua has no server-side query, so gpk filters the list locally.
func (a *Aqua) Search(query string) ([]model.Package, error) {
	out, err := exec.Command("aqua", "list").Output()
	if err != nil || len(out) == 0 {
		return nil, nil
	}
	return ParseAquaRegistryList(out, query), nil
}

// ParseAquaRegistryList parses `aqua list` output ("<registry>,<package>") and
// keeps the entries whose package name contains query (case-insensitive).
func ParseAquaRegistryList(data []byte, query string) []model.Package {
	q := strings.ToLower(query)
	var pkgs []model.Package
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		registry, name, ok := strings.Cut(line, ",")
		if !ok || name == "" {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(name), q) {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:       name,
			Source:     model.SourceAqua,
			Repository: registry,
		})
	}
	return pkgs
}

// InstallCmd adds a package to the first global configuration file with
// `aqua generate -g -i`, pinning the resolved latest version. aqua installs the
// binary lazily on first use (or with a subsequent `aqua install`); adding it
// to the global config is how aqua users register a global tool. Requires
// $AQUA_GLOBAL_CONFIG to point at an existing file.
func (a *Aqua) InstallCmd(name string) *exec.Cmd {
	return exec.Command("aqua", "generate", "-g", "-i", name)
}

// aquaGlobalConfig returns the first path in $AQUA_GLOBAL_CONFIG, matching the
// file aqua's `generate -g` writes to (GlobalConfigFilePaths[0]).
func aquaGlobalConfig() string {
	for _, p := range filepath.SplitList(os.Getenv("AQUA_GLOBAL_CONFIG")) {
		if p != "" {
			return p
		}
	}
	return ""
}
