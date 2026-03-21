package parsing

import (
	"strings"
	"testing"
)

func TestOpamListParsing(t *testing.T) {
	// opam list --installed --columns=package --short
	output := `dune.3.14.2
merlin.4.14-414
ocaml.4.14.1
ocamlfind.1.9.6`

	type pkg struct{ name, version string }
	var pkgs []pkg
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		dotIdx := strings.Index(line, ".")
		if dotIdx < 0 {
			pkgs = append(pkgs, pkg{line, "base"})
			continue
		}
		pkgs = append(pkgs, pkg{line[:dotIdx], line[dotIdx+1:]})
	}

	if len(pkgs) != 4 {
		t.Fatalf("expected 4 packages, got %d", len(pkgs))
	}
	if pkgs[0].name != "dune" || pkgs[0].version != "3.14.2" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
	if pkgs[1].name != "merlin" || pkgs[1].version != "4.14-414" {
		t.Errorf("pkg 1: %+v", pkgs[1])
	}
}

func TestOpamUpgradeParsing(t *testing.T) {
	output := `The following actions would be performed:
=== upgrade 2 packages
  - upgrade dune          3.14.2 to 3.15.0
  - upgrade merlin        4.14-414 to 4.14-502
===== 0 to install | 0 to reinstall | 2 to upgrade | 0 to downgrade | 0 to remove =====
Dry run: exiting now.`

	updates := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- upgrade") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 6 && fields[4] == "to" {
			updates[fields[2]] = fields[5]
		}
	}

	if len(updates) != 2 {
		t.Fatalf("expected 2 updates, got %d", len(updates))
	}
	if updates["dune"] != "3.15.0" {
		t.Errorf("dune: got %q", updates["dune"])
	}
	if updates["merlin"] != "4.14-502" {
		t.Errorf("merlin: got %q", updates["merlin"])
	}
}
