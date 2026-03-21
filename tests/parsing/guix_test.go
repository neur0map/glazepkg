package parsing

import (
	"strings"
	"testing"
)

func TestGuixListParsing(t *testing.T) {
	output := "nethack\t3.6.6\tout\t/gnu/store/r7if10kgajw-nethack-3.6.6\nnyxt\t2-pre-release-5\tout\t/gnu/store/z1yfwmwh5b-nyxt-2-pre-release-5\nsqlite\t3.32.3\tout\t/gnu/store/g9gf1ndxry-sqlite-3.32.3"

	type pkg struct{ name, version string }
	var pkgs []pkg
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		pkgs = append(pkgs, pkg{strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1])})
	}

	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}
	if pkgs[0].name != "nethack" || pkgs[0].version != "3.6.6" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
	if pkgs[1].name != "nyxt" || pkgs[1].version != "2-pre-release-5" {
		t.Errorf("pkg 1: %+v", pkgs[1])
	}
}

func TestGuixUpgradeParsing(t *testing.T) {
	output := `The following packages would be upgraded:
   emacs 27.2 → 28.1
   fontconfig 2.13.1 → 2.13.94
   ungoogled-chromium 96.0.4664.45-1 → 97.0.4692.71-1

The following packages would be installed:
   libfoo 1.2.3`

	updates := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		if !strings.Contains(line, "→") {
			continue
		}
		fields := strings.Fields(line)
		arrowIdx := -1
		for i, f := range fields {
			if f == "→" {
				arrowIdx = i
				break
			}
		}
		if arrowIdx >= 1 && arrowIdx+1 < len(fields) {
			updates[fields[0]] = fields[arrowIdx+1]
		}
	}

	if len(updates) != 3 {
		t.Fatalf("expected 3 updates, got %d", len(updates))
	}
	if updates["emacs"] != "28.1" {
		t.Errorf("emacs: got %q", updates["emacs"])
	}
	if updates["fontconfig"] != "2.13.94" {
		t.Errorf("fontconfig: got %q", updates["fontconfig"])
	}
	if updates["ungoogled-chromium"] != "97.0.4692.71-1" {
		t.Errorf("chromium: got %q", updates["ungoogled-chromium"])
	}
}

func TestGuixShowDescriptionParsing(t *testing.T) {
	output := `name: hello
version: 2.12.2
outputs:
+ out
systems: x86_64-linux i686-linux
dependencies:
location: gnu/packages/base.scm:88:2
home-page: https://www.gnu.org/software/hello/
license: GPL-3.0+
synopsis: Hello, GNU world: An example GNU package
description: GNU Hello prints the message "Hello, world!" and then exits.
+ It serves as an example of standard GNU coding practices.`

	var synopsis string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "synopsis: ") {
			synopsis = strings.TrimPrefix(line, "synopsis: ")
			break
		}
	}

	if synopsis != "Hello, GNU world: An example GNU package" {
		t.Errorf("synopsis: got %q", synopsis)
	}
}
