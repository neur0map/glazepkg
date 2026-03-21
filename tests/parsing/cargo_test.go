package parsing

import (
	"bufio"
	"strings"
	"testing"
)

func TestCargoListParsing(t *testing.T) {
	output := `bat v0.24.0:
    bat
cargo-update v14.0.3:
    cargo-install-update
    cargo-install-update-config
ripgrep v14.1.0 (/home/user/src/ripgrep):
    rg
`
	type pkg struct{ name, version string }
	var pkgs []pkg
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, " ") || line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		version := strings.TrimPrefix(parts[1], "v")
		version = strings.TrimSuffix(version, ":")
		pkgs = append(pkgs, pkg{name, version})
	}

	if len(pkgs) != 3 {
		t.Fatalf("expected 3 crates, got %d", len(pkgs))
	}
	if pkgs[0].name != "bat" || pkgs[0].version != "0.24.0" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
	if pkgs[1].name != "cargo-update" || pkgs[1].version != "14.0.3" {
		t.Errorf("pkg 1: %+v", pkgs[1])
	}
	if pkgs[2].name != "ripgrep" || pkgs[2].version != "14.1.0" {
		t.Errorf("pkg 2: %+v", pkgs[2])
	}
}
