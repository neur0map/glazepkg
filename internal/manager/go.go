package manager

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/neur0map/glazepkg/internal/model"
)

type Go struct{}

func (g *Go) Name() model.Source { return model.SourceGo }

func (g *Go) Available() bool {
	dir := goBinDir()
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

func (g *Go) Scan() ([]model.Package, error) {
	dir := goBinDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:        e.Name(),
			Version:     "",
			Source:      model.SourceGo,
			InstalledAt: info.ModTime(),
		})
	}
	return pkgs, nil
}

func (g *Go) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("rm", filepath.Join(goBinDir(), name))
}

// Describe reads the main module import path embedded in each installed
// binary using `go version -m`. Reliable because every Go binary built since
// Go 1.12 carries BuildInfo, whereas `go doc <name>` rarely resolves for
// installed binaries (their source is not necessarily cached).
func (g *Go) Describe(pkgs []model.Package) map[string]string {
	binDir := goBinDir()
	descs := make(map[string]string, len(pkgs))
	for _, pkg := range pkgs {
		if desc := goBinaryModulePath(filepath.Join(binDir, pkg.Name)); desc != "" {
			descs[pkg.Name] = desc
		}
	}
	return descs
}

// goBinaryModulePath returns the main module import path for a Go binary at
// path, or "" if the binary is not a Go binary or `go` is not installed.
// Parses the "path" line from `go version -m`:
//
//	<path>: go1.21.0
//		path	github.com/foo/bar/cmd/baz
//		mod	github.com/foo/bar	v1.2.3	h1:...
func goBinaryModulePath(path string) string {
	out, err := exec.Command("go", "version", "-m", path).Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "path" {
			return fields[1]
		}
	}
	return ""
}

func (g *Go) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("go", "install", name+"@latest")
}

func (g *Go) InstallCmd(name string) *exec.Cmd {
	return exec.Command("go", "install", name+"@latest")
}

func goBinDir() string {
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return filepath.Join(gopath, "bin")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "go", "bin")
}
