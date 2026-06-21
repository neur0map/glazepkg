package manager

import (
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// SoftwareUpdate surfaces pending macOS system updates via `softwareupdate`.
// Read-only: applying updates needs sudo and may trigger a restart.
const maxSoftwareUpdates = 50

type SoftwareUpdate struct{}

func (s *SoftwareUpdate) Name() model.Source { return model.SourceSoftwareUpdate }

func (s *SoftwareUpdate) Available() bool {
	return runtime.GOOS == "darwin" && commandExists("softwareupdate")
}

func (s *SoftwareUpdate) Scan() ([]model.Package, error) {
	if runtime.GOOS != "darwin" {
		return nil, nil
	}
	// The listing is written to stderr, so combine both streams.
	out, err := exec.Command("softwareupdate", "--list", "--no-scan").CombinedOutput()
	if err != nil {
		return nil, err
	}
	pkgs := parseSoftwareUpdateList(out)
	if len(pkgs) > maxSoftwareUpdates {
		pkgs = pkgs[:maxSoftwareUpdates]
	}
	return pkgs, nil
}

// parseSoftwareUpdateList parses `softwareupdate --list` output. Each update spans
// two lines: a "* Label:" (recommended) or "- Label:" (optional) line followed by
// an indented line of comma-separated "Key: value" fields (Title, Version, ...).
func parseSoftwareUpdateList(data []byte) []model.Package {
	var pkgs []model.Package
	now := time.Now()
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		var label string
		switch {
		case strings.HasPrefix(line, "* Label:"):
			label = strings.TrimSpace(strings.TrimPrefix(line, "* Label:"))
		case strings.HasPrefix(line, "- Label:"):
			label = strings.TrimSpace(strings.TrimPrefix(line, "- Label:"))
		default:
			continue
		}
		if label == "" || i+1 >= len(lines) {
			continue
		}
		fields := parseSoftwareUpdateFields(lines[i+1])
		i++
		pkgs = append(pkgs, model.Package{
			Name:        label,
			Version:     fields["Version"],
			Source:      model.SourceSoftwareUpdate,
			Description: fields["Title"],
			InstalledAt: now,
		})
	}
	return pkgs
}

// parseSoftwareUpdateFields splits a "Key: value, Key: value" line into a map.
func parseSoftwareUpdateFields(line string) map[string]string {
	fields := make(map[string]string)
	for _, part := range strings.Split(line, ",") {
		key, val, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(key)] = strings.TrimSpace(val)
	}
	return fields
}
