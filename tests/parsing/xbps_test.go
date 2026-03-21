package parsing

import (
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
)

func TestSplitXbpsNameVersion(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"acl-2.3.1_1", "acl", "2.3.1_1"},
		{"acpid-2.0.33_2", "acpid", "2.0.33_2"},
		{"xorg-server-xwayland-1.20.10_2", "xorg-server-xwayland", "1.20.10_2"},
		{"bash-5.1.016_1", "bash", "5.1.016_1"},
		{"noversion", "noversion", ""},
	}
	for _, tt := range tests {
		name, ver := manager.SplitXbpsNameVersion(tt.input)
		if name != tt.wantName || ver != tt.wantVersion {
			t.Errorf("SplitXbpsNameVersion(%q) = (%q, %q), want (%q, %q)",
				tt.input, name, ver, tt.wantName, tt.wantVersion)
		}
	}
}

func TestXbpsListParsing(t *testing.T) {
	output := `ii acl-2.3.1_1                     Access Control List filesystem support
ii acpid-2.0.33_2                  ACPI Daemon With Netlink Support
ii bash-5.1.016_1                  GNU Bourne Again SHell`

	type pkg struct{ name, version, desc string }
	var pkgs []pkg
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name, version := manager.SplitXbpsNameVersion(fields[1])
		desc := ""
		if len(fields) > 2 {
			desc = strings.Join(fields[2:], " ")
		}
		pkgs = append(pkgs, pkg{name, version, desc})
	}

	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}
	if pkgs[0].name != "acl" || pkgs[0].version != "2.3.1_1" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
	if pkgs[2].name != "bash" || pkgs[2].desc != "GNU Bourne Again SHell" {
		t.Errorf("pkg 2: %+v", pkgs[2])
	}
}

func TestXbpsUpdateParsing(t *testing.T) {
	output := `xorg-server-xwayland-1.20.10_2 update x86_64 https://repo.voidlinux.org/current 1979576 909924
bash-5.2.000_1 update x86_64 https://repo.voidlinux.org/current 4194304 1048576`

	updates := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[1] != "update" {
			continue
		}
		name, version := manager.SplitXbpsNameVersion(fields[0])
		if name != "" {
			updates[name] = version
		}
	}

	if updates["xorg-server-xwayland"] != "1.20.10_2" {
		t.Errorf("xorg-server-xwayland: got %q", updates["xorg-server-xwayland"])
	}
	if updates["bash"] != "5.2.000_1" {
		t.Errorf("bash: got %q", updates["bash"])
	}
}
