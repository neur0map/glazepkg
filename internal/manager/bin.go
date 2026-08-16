package manager

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// Bin manages pre-compiled binaries installed via bin
// (https://github.com/marcosnils/bin). bin has no registry: every install is a
// GitHub/GitLab/Codeberg/docker/http/go URL, and state lives in a JSON config
// file. Scan reads that config directly (the stable contract) rather than the
// human-oriented `bin ls` table.
type Bin struct{}

func (b *Bin) Name() model.Source { return model.SourceBin }

func (b *Bin) Available() bool { return commandExists("bin") }

func (b *Bin) Scan() ([]model.Package, error) {
	data, err := os.ReadFile(binConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return ParseBinConfig(data)
}

type binEntry struct {
	Path     string `json:"path"`
	Version  string `json:"version"`
	URL      string `json:"url"`
	Provider string `json:"provider"`
}

type binConfig struct {
	DefaultPath string              `json:"default_path"`
	Bins        map[string]binEntry `json:"bins"`
}

// ParseBinConfig turns bin's config.json into one package per managed binary.
// The package name is the binary's filename, which is what `bin update <name>`
// and `bin remove <name>` accept when bin's download dir is on PATH.
func ParseBinConfig(data []byte) ([]model.Package, error) {
	var cfg binConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	pkgs := make([]model.Package, 0, len(cfg.Bins))
	for key, entry := range cfg.Bins {
		path := entry.Path
		if path == "" {
			path = key
		}
		pkgs = append(pkgs, model.Package{
			Name:        filepath.Base(path),
			Version:     entry.Version,
			Source:      model.SourceBin,
			Repository:  entry.URL,
			Location:    path,
			InstalledAt: time.Now(),
		})
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].Name < pkgs[j].Name })
	return pkgs, nil
}

// CheckUpdates runs `bin update --dry-run`, which reports outdated binaries on
// stderr as "<path> <old> -> <new> (<url>)" and exits 3 when any are found.
func (b *Bin) CheckUpdates(pkgs []model.Package) map[string]string {
	// Output lands on stderr and the non-empty case exits non-zero, so combine
	// the streams and ignore the exit error.
	out, _ := exec.Command("bin", "update", "--dry-run").CombinedOutput()
	latest := ParseBinOutdated(out)
	updates := make(map[string]string, len(latest))
	for _, p := range pkgs {
		if v, ok := latest[p.Name]; ok {
			updates[p.Name] = v
		}
	}
	return updates
}

var binAnsiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

// ParseBinOutdated extracts binary-name -> latest-version pairs from
// `bin update --dry-run` output. Each update line reads
// "<path> <oldversion> -> <newversion> (<url>)", prefixed by an ANSI-coloured
// bullet.
func ParseBinOutdated(data []byte) map[string]string {
	updates := make(map[string]string)
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(binAnsiRe.ReplaceAllString(raw, ""))
		arrow := strings.Index(line, " -> ")
		if arrow < 0 {
			continue
		}
		leftFields := strings.Fields(line[:arrow])
		rightFields := strings.Fields(line[arrow+len(" -> "):])
		if len(leftFields) < 2 || len(rightFields) < 1 {
			continue
		}
		path := leftFields[len(leftFields)-2]
		updates[filepath.Base(path)] = rightFields[0]
	}
	return updates
}

func (b *Bin) InstallCmd(name string) *exec.Cmd {
	return exec.Command("bin", "install", name)
}

func (b *Bin) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("bin", "update", name)
}

func (b *Bin) UpgradeCmdYes(name string) *exec.Cmd {
	return exec.Command("bin", "update", "-y", name)
}

func (b *Bin) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("bin", "remove", name)
}

// binConfigPath mirrors bin's own config resolution
// (pkg/config/config.go:getConfigPath) so gpk reads the same file bin writes.
func binConfigPath() string {
	if c := os.Getenv("BIN_CONFIG"); c != "" {
		return c
	}
	home, homeErr := os.UserHomeDir()
	if homeErr == nil {
		if p := filepath.Join(home, ".bin", "config.json"); fileExists(p) {
			return p
		}
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		if fileExists(x) {
			return filepath.Join(x, "bin", "config.json")
		}
	}
	if homeErr == nil {
		if fileExists(filepath.Join(home, ".config")) {
			return filepath.Join(home, ".config", "bin", "config.json")
		}
		return filepath.Join(home, ".bin", "config.json")
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
