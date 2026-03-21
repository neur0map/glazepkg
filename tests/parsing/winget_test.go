package parsing

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWingetJSONSchemaB(t *testing.T) {
	data := `{"Sources":[{"Packages":[{"Name":"Git","Id":"Git.Git","Version":"2.44.0","Source":"winget"},{"Name":"Node.js","Id":"OpenJS.NodeJS","Version":"20.11.0","Source":"winget"}]}]}`

	var schemaB struct {
		Sources []struct {
			Packages []struct {
				Name    string `json:"Name"`
				Version string `json:"Version"`
			} `json:"Packages"`
		} `json:"Sources"`
	}
	if err := json.Unmarshal([]byte(data), &schemaB); err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, s := range schemaB.Sources {
		total += len(s.Packages)
	}
	if total != 2 {
		t.Fatalf("expected 2 packages, got %d", total)
	}
	if schemaB.Sources[0].Packages[0].Name != "Git" {
		t.Errorf("got %q", schemaB.Sources[0].Packages[0].Name)
	}
}

func TestWingetTextParsing(t *testing.T) {
	output := `Name                  Id                    Version    Available  Source
--------------------  --------------------  ---------  ---------  ------
Git                   Git.Git               2.44.0     2.45.0     winget
Node.js               OpenJS.NodeJS         20.11.0               winget
`
	// Find separator line, derive columns
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
		if len(fields) < 3 || fields[0] == "" {
			continue
		}
		pkgs = append(pkgs, pkg{fields[0], fields[2]})
	}

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	if pkgs[0].name != "Git" || pkgs[0].version != "2.44.0" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
	if pkgs[1].name != "Node.js" || pkgs[1].version != "20.11.0" {
		t.Errorf("pkg 1: %+v", pkgs[1])
	}
}

// helpers mirroring winget.go logic
func isSepLine(line string) bool {
	if line == "" {
		return false
	}
	hasDash := false
	for _, c := range line {
		switch c {
		case '-':
			hasDash = true
		case ' ':
		default:
			return false
		}
	}
	return hasDash
}

func deriveColumns(sep string) []int {
	var starts []int
	prev := byte(' ')
	for i := 0; i < len(sep); i++ {
		if sep[i] == '-' && prev == ' ' {
			starts = append(starts, i)
		}
		prev = sep[i]
	}
	return starts
}

func extractFields(line string, starts []int) []string {
	fields := make([]string, len(starts))
	for i, start := range starts {
		end := len(line)
		if i+1 < len(starts) && starts[i+1] < len(line) {
			end = starts[i+1]
		}
		if start < len(line) {
			fields[i] = strings.TrimSpace(line[start:end])
		}
	}
	return fields
}
