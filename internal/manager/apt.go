package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Apt struct{}

func (a *Apt) Name() model.Source { return model.SourceApt }

func (a *Apt) Available() bool { return commandExists("dpkg-query") }

func (a *Apt) Scan() ([]model.Package, error) {
	out, err := exec.Command("dpkg-query", "-W", "-f=${Package} ${Version}\n").Output()
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:        fields[0],
			Version:     fields[1],
			Source:      model.SourceApt,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (a *Apt) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, p := range pkgs {
		out, err := exec.Command("apt-cache", "show", p.Name).Output()
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Description:") {
				descs[p.Name] = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
				break
			}
		}
	}
	return descs
}
