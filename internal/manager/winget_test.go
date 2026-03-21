package manager

import (
	"testing"
)

// ── wingetIsSep ──────────────────────────────────────────────────────────────

func TestWingetIsSep(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"---  ---  ---", true},
		{"--------------------  --------------------  ---------  ---------  ------", true},
		{"-", true},
		{"", false},
		{"Name  Id  Version", false},
		{"Git   Git.Git  2.44.0", false},
		{"   ", false},
		{"--  -- x", false},
		{"  --  --  --  ", false}, // leading/trailing spaces contain non-dash non-space? No — spaces are ok
	}
	// Fix: leading/trailing spaces are fine since space is allowed
	tests[len(tests)-1].want = true

	for _, tt := range tests {
		got := wingetIsSep(tt.input)
		if got != tt.want {
			t.Errorf("wingetIsSep(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// ── wingetColumns ────────────────────────────────────────────────────────────

func TestWingetColumns(t *testing.T) {
	tests := []struct {
		sep  string
		want []int
	}{
		{
			"----  -------  --------",
			[]int{0, 6, 15},
		},
		{
			"-  -  -",
			[]int{0, 3, 6},
		},
		{
			"--------------------  --------------------  ---------  ---------  ------",
			[]int{0, 22, 44, 55, 66},
		},
		// Single column
		{"------", []int{0}},
	}
	for _, tt := range tests {
		got := wingetColumns(tt.sep)
		if len(got) != len(tt.want) {
			t.Errorf("wingetColumns(%q) = %v, want %v", tt.sep, got, tt.want)
			continue
		}
		for i := range tt.want {
			if got[i] != tt.want[i] {
				t.Errorf("wingetColumns(%q)[%d] = %d, want %d", tt.sep, i, got[i], tt.want[i])
			}
		}
	}
}

func TestWingetColumnsIndented(t *testing.T) {
	// Scoop uses indented separators; raw separator (after trim) is still parsed correctly
	sep := "----  -------  ------"
	got := wingetColumns(sep)
	if len(got) == 0 || got[0] != 0 {
		t.Errorf("expected first column at 0, got %v", got)
	}
}

// ── wingetExtract ────────────────────────────────────────────────────────────

func TestWingetExtract(t *testing.T) {
	sep := "--------------------  --------------------  ---------  ---------  ------"
	starts := wingetColumns(sep)

	tests := []struct {
		line   string
		col    int
		expect string
	}{
		{"Git                   Git.Git               2.44.0     2.45.0     winget", 0, "Git"},
		{"Git                   Git.Git               2.44.0     2.45.0     winget", 1, "Git.Git"},
		{"Git                   Git.Git               2.44.0     2.45.0     winget", 2, "2.44.0"},
		{"Git                   Git.Git               2.44.0     2.45.0     winget", 3, "2.45.0"},
		{"Git                   Git.Git               2.44.0     2.45.0     winget", 4, "winget"},
	}
	for _, tt := range tests {
		fields := wingetExtract(tt.line, starts)
		if tt.col >= len(fields) {
			t.Errorf("line %q: col %d out of range (got %d fields)", tt.line, tt.col, len(fields))
			continue
		}
		if fields[tt.col] != tt.expect {
			t.Errorf("col %d: got %q, want %q", tt.col, fields[tt.col], tt.expect)
		}
	}
}

func TestWingetExtractMultiWordName(t *testing.T) {
	sep := "----------------------------------------  ------------------------------------  ---------------  ---------------  ------"
	starts := wingetColumns(sep)
	line := "7-Zip 24.09 (x64)                         7zip.7zip                             24.09            24.09.1          winget"
	fields := wingetExtract(line, starts)
	if fields[0] != "7-Zip 24.09 (x64)" {
		t.Errorf("name = %q, want %q", fields[0], "7-Zip 24.09 (x64)")
	}
	if fields[2] != "24.09" {
		t.Errorf("version = %q, want %q", fields[2], "24.09")
	}
	if fields[3] != "24.09.1" {
		t.Errorf("available = %q, want %q", fields[3], "24.09.1")
	}
}

func TestWingetExtractShortLineNoPanic(t *testing.T) {
	starts := []int{0, 10, 25, 40}
	line := "ShortApp  1.0"
	fields := wingetExtract(line, starts)
	if fields[0] != "ShortApp" {
		t.Errorf("fields[0] = %q, want %q", fields[0], "ShortApp")
	}
	for i := 2; i < len(fields); i++ {
		if fields[i] != "" {
			t.Errorf("fields[%d] = %q, want empty string", i, fields[i])
		}
	}
}

// ── parseTextOutput ──────────────────────────────────────────────────────────

func TestWingetParseTextOutput(t *testing.T) {
	w := &Winget{}
	input := "Name                    Id                      Version    Available  Source\n" +
		"----------------------  ----------------------  ---------  ---------  ------\n" +
		"Git                     Git.Git                 2.44.0     2.45.0     winget\n" +
		"Mozilla Firefox         Mozilla.Firefox         121.0                 winget\n" +
		"7-Zip 24.09 (x64)       7zip.7zip               24.09                 winget\n"

	pkgs := w.parseTextOutput(input)
	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d: %v", len(pkgs), pkgs)
	}

	tests := []struct{ name, version string }{
		{"Git", "2.44.0"},
		{"Mozilla Firefox", "121.0"},
		{"7-Zip 24.09 (x64)", "24.09"},
	}
	for i, tt := range tests {
		if pkgs[i].Name != tt.name {
			t.Errorf("[%d] Name = %q, want %q", i, pkgs[i].Name, tt.name)
		}
		if pkgs[i].Version != tt.version {
			t.Errorf("[%d] Version = %q, want %q", i, pkgs[i].Version, tt.version)
		}
	}
}

func TestWingetParseTextOutputEmptyVersion(t *testing.T) {
	w := &Winget{}
	// Package with no version in the column should get "unknown"
	input := "Name        Id          Version\n" +
		"----------  ----------  -------\n" +
		"NoVersion   no.version          \n"

	pkgs := w.parseTextOutput(input)
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d", len(pkgs))
	}
	if pkgs[0].Version != "unknown" {
		t.Errorf("Version = %q, want %q", pkgs[0].Version, "unknown")
	}
}

func TestWingetParseTextOutputNoSeparator(t *testing.T) {
	w := &Winget{}
	// No separator line → no packages parsed
	input := "Git Git.Git 2.44.0\nFirefox Mozilla.Firefox 121.0\n"
	pkgs := w.parseTextOutput(input)
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages without separator, got %d", len(pkgs))
	}
}

// ── parseUpgradeOutput ───────────────────────────────────────────────────────

func TestWingetParseUpgradeOutput(t *testing.T) {
	w := &Winget{}
	input := "Name              Id               Version    Available  Source\n" +
		"----------------  ---------------  ---------  ---------  ------\n" +
		"Git               Git.Git          2.44.0     2.45.0     winget\n" +
		"Mozilla Firefox   Mozilla.Firefox  121.0      122.0      winget\n" +
		"\n" +
		"2 upgrades available.\n"

	updates := w.parseUpgradeOutput(input)
	if len(updates) != 2 {
		t.Fatalf("expected 2 updates, got %d: %v", len(updates), updates)
	}
	if updates["Git"] != "2.45.0" {
		t.Errorf("Git update = %q, want %q", updates["Git"], "2.45.0")
	}
	if updates["Mozilla Firefox"] != "122.0" {
		t.Errorf("Mozilla Firefox update = %q, want %q", updates["Mozilla Firefox"], "122.0")
	}
}

func TestWingetParseUpgradeOutputNoUpdates(t *testing.T) {
	w := &Winget{}
	// When there are no upgrades, winget prints a plain message with no table
	input := "No applicable upgrades were found.\n"
	updates := w.parseUpgradeOutput(input)
	if len(updates) != 0 {
		t.Errorf("expected 0 updates, got %d: %v", len(updates), updates)
	}
}

func TestWingetParseUpgradeOutputEmptyTable(t *testing.T) {
	w := &Winget{}
	// Table present but no data rows (all packages up to date)
	input := "Name  Id  Version  Available  Source\n" +
		"----  --  -------  ---------  ------\n"
	updates := w.parseUpgradeOutput(input)
	if len(updates) != 0 {
		t.Errorf("expected 0 updates for empty table, got %d: %v", len(updates), updates)
	}
}

// ── parseJSON ────────────────────────────────────────────────────────────────

func TestWingetParseJSONFlatArray(t *testing.T) {
	w := &Winget{}
	data := []byte(`[{"Name":"Git","Id":"Git.Git","Version":"2.44.0","Source":"winget"},{"Name":"7-Zip","Id":"7zip.7zip","Version":"24.09","Source":"winget"}]`)
	pkgs, err := w.parseJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	if pkgs[0].Name != "Git" || pkgs[0].Version != "2.44.0" {
		t.Errorf("pkg[0] = {%s %s}, want {Git 2.44.0}", pkgs[0].Name, pkgs[0].Version)
	}
}

func TestWingetParseJSONSourcesSchema(t *testing.T) {
	w := &Winget{}
	data := []byte(`{"Sources":[{"Packages":[{"Name":"Git","Id":"Git.Git","Version":"2.44.0"}]}]}`)
	pkgs, err := w.parseJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 || pkgs[0].Name != "Git" {
		t.Errorf("expected [{Git 2.44.0}], got %v", pkgs)
	}
}

func TestWingetParseJSONUnknownSchema(t *testing.T) {
	w := &Winget{}
	_, err := w.parseJSON([]byte(`{"unknown":"field"}`))
	if err == nil {
		t.Error("expected error for unrecognized schema, got nil")
	}
}

func TestWingetParseJSONMissingVersion(t *testing.T) {
	w := &Winget{}
	data := []byte(`[{"Name":"NoVer","Id":"no.ver","Version":""}]`)
	pkgs, err := w.parseJSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 || pkgs[0].Version != "unknown" {
		t.Errorf("expected version='unknown', got %q", pkgs[0].Version)
	}
}
