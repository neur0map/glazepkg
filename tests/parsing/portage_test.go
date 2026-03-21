package parsing

import (
	"strings"
	"testing"
)

func splitPortageCPV(cpv string) (string, string) {
	for i := len(cpv) - 1; i > 0; i-- {
		if cpv[i] == '-' && i+1 < len(cpv) && cpv[i+1] >= '0' && cpv[i+1] <= '9' {
			return cpv[:i], cpv[i+1:]
		}
	}
	return cpv, ""
}

func TestSplitPortageCPV(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"app-editors/vim-9.0.1678", "app-editors/vim", "9.0.1678"},
		{"dev-lang/python-3.11.5", "dev-lang/python", "3.11.5"},
		{"sys-libs/glibc-2.37-r3", "sys-libs/glibc", "2.37-r3"},
		{"app-portage/gentoolkit-0.6.3", "app-portage/gentoolkit", "0.6.3"},
		{"sys-apps/portage-2.3.99", "sys-apps/portage", "2.3.99"},
	}
	for _, tt := range tests {
		name, ver := splitPortageCPV(tt.input)
		if name != tt.wantName || ver != tt.wantVersion {
			t.Errorf("splitPortageCPV(%q) = (%q, %q), want (%q, %q)",
				tt.input, name, ver, tt.wantName, tt.wantVersion)
		}
	}
}

func TestPortageQlistParsing(t *testing.T) {
	output := `app-editors/vim-9.0.1678
app-portage/gentoolkit-0.6.3
dev-lang/python-3.11.5
sys-libs/glibc-2.37-r3`

	type pkg struct{ name, version string }
	var pkgs []pkg
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, version := splitPortageCPV(line)
		pkgs = append(pkgs, pkg{name, version})
	}

	if len(pkgs) != 4 {
		t.Fatalf("expected 4 packages, got %d", len(pkgs))
	}
	if pkgs[0].name != "app-editors/vim" || pkgs[0].version != "9.0.1678" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
	if pkgs[2].name != "dev-lang/python" || pkgs[2].version != "3.11.5" {
		t.Errorf("pkg 2: %+v", pkgs[2])
	}
	if pkgs[3].name != "sys-libs/glibc" || pkgs[3].version != "2.37-r3" {
		t.Errorf("pkg 3: %+v", pkgs[3])
	}
}

func TestPortageEmergeUpdateParsing(t *testing.T) {
	output := `These are the packages that would be merged, in order:

Calculating dependencies... done!
[ebuild     U  ] dev-lang/yasm-1.1.0-r1::gentoo [1.1.0::gentoo] USE="nls python" 1,377 kB
[ebuild     U  ] app-editors/vim-9.1.0::gentoo [9.0.1678::gentoo] USE="X" 15,000 kB
[ebuild   R    ] sys-apps/portage-2.3.99::gentoo 777 kB

Total: 3 packages (2 upgrades, 1 reinstall)`

	updates := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, "[ebuild") || !strings.Contains(line, "U") {
			continue
		}
		closeIdx := strings.Index(line, "] ")
		if closeIdx < 0 {
			continue
		}
		rest := strings.TrimSpace(line[closeIdx+2:])
		fields := strings.Fields(rest)
		if len(fields) < 1 {
			continue
		}
		cpv := fields[0]
		if colonIdx := strings.Index(cpv, "::"); colonIdx >= 0 {
			cpv = cpv[:colonIdx]
		}
		name, version := splitPortageCPV(cpv)
		if name != "" && version != "" {
			updates[name] = version
		}
	}

	if len(updates) != 2 {
		t.Fatalf("expected 2 updates, got %d: %v", len(updates), updates)
	}
	if updates["dev-lang/yasm"] != "1.1.0-r1" {
		t.Errorf("yasm: got %q", updates["dev-lang/yasm"])
	}
	if updates["app-editors/vim"] != "9.1.0" {
		t.Errorf("vim: got %q", updates["app-editors/vim"])
	}
}
