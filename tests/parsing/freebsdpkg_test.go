package parsing

import (
	"strings"
	"testing"
)

func TestFreeBSDPkgInfoParsing(t *testing.T) {
	output := `apache24-2.4.57                Apache HTTP Server
bash-5.2.15                    GNU Project's Bourne Again SHell
curl-8.0.1                     Command line tool for transferring data with URLs`

	type pkg struct{ name, version, desc string }
	var pkgs []pkg
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		nameVer := fields[0]
		idx := strings.LastIndex(nameVer, "-")
		if idx <= 0 {
			continue
		}
		desc := ""
		if len(fields) > 1 {
			desc = strings.Join(fields[1:], " ")
		}
		pkgs = append(pkgs, pkg{nameVer[:idx], nameVer[idx+1:], desc})
	}

	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}
	if pkgs[0].name != "apache24" || pkgs[0].version != "2.4.57" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
	if pkgs[1].name != "bash" || pkgs[1].version != "5.2.15" {
		t.Errorf("pkg 1: %+v", pkgs[1])
	}
	if pkgs[2].desc != "Command line tool for transferring data with URLs" {
		t.Errorf("pkg 2 desc: %q", pkgs[2].desc)
	}
}

func TestFreeBSDPkgUpgradeParsing(t *testing.T) {
	output := `Installed packages to be UPGRADED:
	cbsd: 14.2.4 -> 14.2.6
	py38-setuptools: 63.1.0_1 -> 63.1.0_2`

	updates := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "\t") {
			continue
		}
		line = strings.TrimSpace(line)
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		name := line[:colonIdx]
		rest := strings.TrimSpace(line[colonIdx+1:])
		parts := strings.Split(rest, " -> ")
		if len(parts) == 2 {
			updates[name] = strings.TrimSpace(parts[1])
		}
	}

	if updates["cbsd"] != "14.2.6" {
		t.Errorf("cbsd: got %q", updates["cbsd"])
	}
	if updates["py38-setuptools"] != "63.1.0_2" {
		t.Errorf("py38-setuptools: got %q", updates["py38-setuptools"])
	}
}
