package manager

import (
	"os"
	"path/filepath"

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

func goBinDir() string {
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return filepath.Join(gopath, "bin")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "go", "bin")
}
