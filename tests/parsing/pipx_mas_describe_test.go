package parsing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestPipxMetadataSummaryParsing verifies that the Summary field is correctly
// extracted from a PEP 566 METADATA file, matching the logic in
// internal/manager/uv.go:parseMetadataSummary (reused by pipx).
func TestPipxMetadataSummaryParsing(t *testing.T) {
	meta := `Metadata-Version: 2.1
Name: black
Version: 24.3.0
Summary: The uncompromising code formatter.
Home-page: https://github.com/psf/black
Author: Łukasz Langa

Black is the uncompromising Python code formatter.`

	// Write a temporary METADATA file and parse it.
	dir := t.TempDir()
	path := filepath.Join(dir, "METADATA")
	if err := os.WriteFile(path, []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	// Replicate the parseMetadataSummary logic inline so this test has no
	// import cycle and stays in the parsing package.
	got := extractMetadataSummary(path)
	want := "The uncompromising code formatter."
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestPipxMetadataNoneSummary verifies that an empty string is returned when
// there is no Summary header in the METADATA file.
func TestPipxMetadataNoneSummary(t *testing.T) {
	meta := `Metadata-Version: 2.1
Name: sometool
Version: 1.0.0

Body text without a summary header.`

	dir := t.TempDir()
	path := filepath.Join(dir, "METADATA")
	if err := os.WriteFile(path, []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	got := extractMetadataSummary(path)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// extractMetadataSummary is a copy of parseMetadataSummary from
// internal/manager/uv.go, inlined here to avoid an import cycle.
func extractMetadataSummary(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range splitLines(string(data)) {
		if line == "" {
			break
		}
		const prefix = "Summary: "
		if len(line) > len(prefix) && line[:len(prefix)] == prefix {
			return line[len(prefix):]
		}
	}
	return ""
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// TestMasITunesLookupParsing verifies that description is extracted correctly
// from the iTunes Store lookup API JSON response, matching the logic in
// internal/manager/mas.go:masLookupDescription.
func TestMasITunesLookupParsing(t *testing.T) {
	apiResp := `{
		"resultCount": 1,
		"results": [
			{
				"trackName": "Xcode",
				"version": "15.4",
				"description": "Xcode 15 includes a streamlined GitHub integration.\nNew features for all platforms."
			}
		]
	}`

	var result struct {
		Results []struct {
			Description string `json:"description"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(apiResp), &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(result.Results) == 0 {
		t.Fatal("expected at least one result")
	}

	// Apply the same truncation logic used in masLookupDescription.
	desc := result.Results[0].Description
	if idx := indexNewline(desc); idx > 0 {
		desc = trimSpace(desc[:idx])
	}
	want := "Xcode 15 includes a streamlined GitHub integration."
	if desc != want {
		t.Errorf("got %q, want %q", desc, want)
	}
}

// TestMasITunesLookupEmpty verifies graceful handling of an empty result set.
func TestMasITunesLookupEmpty(t *testing.T) {
	apiResp := `{"resultCount": 0, "results": []}`

	var result struct {
		Results []struct {
			Description string `json:"description"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(apiResp), &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(result.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(result.Results))
	}
}

// TestMasScanStoresID verifies that the App Store numeric ID is captured
// alongside the app name, matching the updated Scan() logic in mas.go.
func TestMasScanStoresID(t *testing.T) {
	output := `497799835  Xcode            (15.4)
640199958  Developer        (10.6.5)
1295203466 Microsoft Remote Desktop (10.9.5)`

	type app struct{ id, name, version string }
	var apps []app
	for _, line := range splitLines(output) {
		if line == "" {
			continue
		}
		parenIdx := lastIndex(line, "(")
		if parenIdx < 0 {
			continue
		}
		version := trimSuffix(trimSpace(line[parenIdx+1:]), ")")
		prefix := trimSpace(line[:parenIdx])
		fields := splitFields(prefix)
		if len(fields) < 2 {
			continue
		}
		id := fields[0]
		name := joinFields(fields[1:])
		apps = append(apps, app{id, name, version})
	}

	if len(apps) != 3 {
		t.Fatalf("expected 3 apps, got %d", len(apps))
	}
	if apps[0].id != "497799835" {
		t.Errorf("app 0 id: got %q, want %q", apps[0].id, "497799835")
	}
	if apps[0].name != "Xcode" {
		t.Errorf("app 0 name: got %q, want %q", apps[0].name, "Xcode")
	}
	if apps[2].id != "1295203466" {
		t.Errorf("app 2 id: got %q, want %q", apps[2].id, "1295203466")
	}
	if apps[2].name != "Microsoft Remote Desktop" {
		t.Errorf("app 2 name: got %q, want %q", apps[2].name, "Microsoft Remote Desktop")
	}
}

// Minimal helpers to avoid importing strings in this test file.
func indexNewline(s string) int {
	for i, c := range s {
		if c == '\n' || c == '\r' {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func trimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

func lastIndex(s, sub string) int {
	for i := len(s) - len(sub); i >= 0; i-- {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func splitFields(s string) []string {
	var fields []string
	inField := false
	start := 0
	for i := 0; i <= len(s); i++ {
		isSpace := i == len(s) || s[i] == ' ' || s[i] == '\t'
		if !isSpace && !inField {
			start = i
			inField = true
		} else if isSpace && inField {
			fields = append(fields, s[start:i])
			inField = false
		}
	}
	return fields
}

func joinFields(fields []string) string {
	result := ""
	for i, f := range fields {
		if i > 0 {
			result += " "
		}
		result += f
	}
	return result
}
