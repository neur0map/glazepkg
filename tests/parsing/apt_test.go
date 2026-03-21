package parsing

import (
	"bufio"
	"strings"
	"testing"
)

func TestAptDpkgListParsing(t *testing.T) {
	// dpkg-query -W -f '${Package}\t${Version}\n'
	output := `bash	5.2-6
curl	7.88.1-10+deb12u5
git	1:2.39.2-1.1
python3	3.11.2-1+b1`

	type pkg struct{ name, version string }
	var pkgs []pkg
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		pkgs = append(pkgs, pkg{parts[0], parts[1]})
	}

	if len(pkgs) != 4 {
		t.Fatalf("expected 4 packages, got %d", len(pkgs))
	}
	if pkgs[0].name != "bash" || pkgs[0].version != "5.2-6" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
	if pkgs[2].name != "git" || pkgs[2].version != "1:2.39.2-1.1" {
		t.Errorf("pkg 2: %+v", pkgs[2])
	}
}
