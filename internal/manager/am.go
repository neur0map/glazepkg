package manager

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// AM inventories AppImage programs installed by AM (system-wide) or AppMan
// (rootless). Each program lives in its own directory containing a `remove`
// uninstall script; that file marks a directory as an installed app.
type AM struct{}

func (a *AM) Name() model.Source { return model.SourceAM }

func (a *AM) Available() bool {
	return commandExists("am") || commandExists("appman")
}

func (a *AM) Scan() ([]model.Package, error) {
	var pkgs []model.Package
	seen := make(map[string]bool)
	for _, root := range amRoots() {
		for _, p := range amScanRoot(root) {
			if seen[p.Name] {
				continue
			}
			seen[p.Name] = true
			pkgs = append(pkgs, p)
		}
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Name < pkgs[j].Name })
	return pkgs, nil
}

// amRoots returns the existing apps roots to scan: /opt and /var/opt for AM
// (system-wide) plus the AppMan apps directory from its config file.
func amRoots() []string {
	var roots []string
	add := func(p string) {
		if p == "" {
			return
		}
		fi, err := os.Stat(p)
		if err != nil || !fi.IsDir() {
			return
		}
		for _, r := range roots {
			if r == p {
				return
			}
		}
		roots = append(roots, p)
	}
	if commandExists("am") {
		add("/opt")
		add("/var/opt")
	}
	add(appmanRoot())
	return roots
}

// appmanRoot resolves the AppMan apps directory. Its config file holds the
// path verbatim; relative paths are taken under $HOME.
func appmanRoot() string {
	cfgHome := os.Getenv("XDG_CONFIG_HOME")
	if cfgHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		cfgHome = filepath.Join(home, ".config")
	}
	data, err := os.ReadFile(filepath.Join(cfgHome, "appman", "appman-config"))
	if err != nil {
		return ""
	}
	p := strings.TrimSpace(string(data))
	if p == "" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		p = filepath.Join(home, p)
	}
	return p
}

// amScanRoot returns the apps directly under root: child directories that
// contain a regular `remove` file.
func amScanRoot(root string) []model.Package {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var pkgs []model.Package
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		appdir := filepath.Join(root, e.Name())
		if fi, err := os.Stat(filepath.Join(appdir, "remove")); err != nil || !fi.Mode().IsRegular() {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:        e.Name(),
			Version:     amVersionFromString(amFirstLine(filepath.Join(appdir, "version"))),
			Source:      model.SourceAM,
			Location:    appdir,
			InstalledAt: time.Now(),
		})
	}
	return pkgs
}

func amFirstLine(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}

// amVersionPatterns are tried in order: dotted semver, then 8-digit date, then
// revision hash. The date pattern is reachable only because semver requires a
// dot, so an all-digit token like 20240115 falls through to it.
var amVersionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`v?[0-9]+(\.[0-9]+)+`),
	regexp.MustCompile(`[0-9]{8}`),
	regexp.MustCompile(`[0-9]+-[0-9a-f]{7,}`),
}

func amVersionFromString(s string) string {
	for _, re := range amVersionPatterns {
		if m := re.FindString(s); m != "" {
			return strings.TrimPrefix(m, "v")
		}
	}
	return ""
}

func amBinary() string {
	if commandExists("am") {
		return "am"
	}
	return "appman"
}

func (a *AM) InstallCmd(name string) *exec.Cmd { return exec.Command(amBinary(), "-i", name) }

func (a *AM) UpgradeCmd(name string) *exec.Cmd { return exec.Command(amBinary(), "-u", name) }

func (a *AM) RemoveCmd(name string) *exec.Cmd { return exec.Command(amBinary(), "-R", name) }
