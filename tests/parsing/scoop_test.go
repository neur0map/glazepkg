package parsing

import (
	"strings"
	"testing"
)

func TestScoopListTabularParsing(t *testing.T) {
	// Modern scoop list output with column headers
	output := `Name     Version   Source   Updated          Info
----     -------   ------   -------          ----
7zip     23.01     main     2024-01-15 10:00
git      2.44.0    main     2024-02-01 09:30
nodejs   20.11.0   main     2024-01-20 14:15 Update available
`
	var colStarts []int
	type pkg struct{ name, version string }
	var pkgs []pkg
	for _, line := range strings.Split(output, "\n") {
		if colStarts == nil {
			if isSepLine(line) {
				colStarts = deriveColumns(line)
			}
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := extractFields(line, colStarts)
		if len(fields) < 2 || fields[0] == "" {
			continue
		}
		pkgs = append(pkgs, pkg{fields[0], fields[1]})
	}

	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}
	if pkgs[0].name != "7zip" || pkgs[0].version != "23.01" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
	if pkgs[2].name != "nodejs" || pkgs[2].version != "20.11.0" {
		t.Errorf("pkg 2: %+v", pkgs[2])
	}
}
