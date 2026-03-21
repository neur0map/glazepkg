package parsing

import (
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
)

func TestSplitApkNameVersion(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"bash-5.2.15-r5", "bash", "5.2.15-r5"},
		{"alpine-base-3.18.4-r0", "alpine-base", "3.18.4-r0"},
		{"apk-tools-2.14.0-r2", "apk-tools", "2.14.0-r2"},
		{"musl-1.2.4-r4", "musl", "1.2.4-r4"},
		{"ca-certificates-bundle-20230506-r0", "ca-certificates-bundle", "20230506-r0"},
		{"noversion", "noversion", ""},
	}
	for _, tt := range tests {
		name, ver := manager.SplitApkNameVersion(tt.input)
		if name != tt.wantName || ver != tt.wantVersion {
			t.Errorf("SplitApkNameVersion(%q) = (%q, %q), want (%q, %q)",
				tt.input, name, ver, tt.wantName, tt.wantVersion)
		}
	}
}

func TestApkInfoParsing(t *testing.T) {
	output := `musl-1.2.4-r4 - the musl c library
busybox-1.36.1-r18 - Size optimized toolbox
alpine-base-3.18.4-r0 - Meta package for minimal Alpine base`

	type pkg struct{ name, version, desc string }
	var pkgs []pkg
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		var nameVer, desc string
		if sepIdx := strings.Index(line, " - "); sepIdx >= 0 {
			nameVer = line[:sepIdx]
			desc = line[sepIdx+3:]
		}
		name, version := manager.SplitApkNameVersion(nameVer)
		pkgs = append(pkgs, pkg{name, version, desc})
	}

	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}
	if pkgs[0].name != "musl" || pkgs[0].version != "1.2.4-r4" {
		t.Errorf("pkg 0: got %q %q", pkgs[0].name, pkgs[0].version)
	}
	if pkgs[1].name != "busybox" || pkgs[1].desc != "Size optimized toolbox" {
		t.Errorf("pkg 1: got name=%q desc=%q", pkgs[1].name, pkgs[1].desc)
	}
	if pkgs[2].name != "alpine-base" {
		t.Errorf("pkg 2: got name=%q, want alpine-base", pkgs[2].name)
	}
}

func TestApkUpgradeParsing(t *testing.T) {
	output := `(1/3) Upgrading musl (1.2.4-r4 -> 1.2.5-r0)
(2/3) Upgrading busybox (1.36.1-r18 -> 1.36.1-r28)
(3/3) Upgrading alpine-base (3.18.4-r0 -> 3.19.0-r0)`

	updates := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, "Upgrading") {
			continue
		}
		upgIdx := strings.Index(line, "Upgrading ")
		if upgIdx < 0 {
			continue
		}
		rest := line[upgIdx+len("Upgrading "):]
		parenIdx := strings.Index(rest, "(")
		if parenIdx < 0 {
			continue
		}
		name := strings.TrimSpace(rest[:parenIdx])
		inner := strings.TrimSuffix(strings.TrimSpace(rest[parenIdx+1:]), ")")
		parts := strings.Split(inner, " -> ")
		if len(parts) == 2 {
			updates[name] = strings.TrimSpace(parts[1])
		}
	}

	if len(updates) != 3 {
		t.Fatalf("expected 3 updates, got %d", len(updates))
	}
	if updates["musl"] != "1.2.5-r0" {
		t.Errorf("musl: got %q", updates["musl"])
	}
	if updates["busybox"] != "1.36.1-r28" {
		t.Errorf("busybox: got %q", updates["busybox"])
	}
}
