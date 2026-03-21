package parsing

import (
	"bufio"
	"strings"
	"testing"
)

func TestDnfListParsing(t *testing.T) {
	// dnf list installed output (after header)
	output := `bash.x86_64                    5.2.26-3.fc40           @anaconda
curl.x86_64                    8.6.0-7.fc40            @updates
git.x86_64                     2.44.0-1.fc40           @updates
python3.x86_64                 3.12.3-2.fc40           @anaconda`

	type pkg struct{ name, version string }
	var pkgs []pkg
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		nameArch := fields[0]
		dotIdx := strings.LastIndex(nameArch, ".")
		name := nameArch
		if dotIdx > 0 {
			name = nameArch[:dotIdx]
		}
		// Version may include release: "5.2.26-3.fc40"
		version := fields[1]
		pkgs = append(pkgs, pkg{name, version})
	}

	if len(pkgs) != 4 {
		t.Fatalf("expected 4 packages, got %d", len(pkgs))
	}
	if pkgs[0].name != "bash" || pkgs[0].version != "5.2.26-3.fc40" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
	if pkgs[2].name != "git" {
		t.Errorf("pkg 2 name: got %q", pkgs[2].name)
	}
}
