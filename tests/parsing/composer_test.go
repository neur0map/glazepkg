package parsing

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestComposerShowJSON(t *testing.T) {
	data := `{
		"installed": [
			{"name": "laravel/installer", "version": "v4.5.0", "description": "A CLI tool to install Laravel"},
			{"name": "phpunit/phpunit", "version": "10.5.17", "description": "The PHP Unit Testing framework"}
		]
	}`

	var result struct {
		Installed []struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description"`
		} `json:"installed"`
	}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(result.Installed) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(result.Installed))
	}
	// Version should strip leading "v"
	v := strings.TrimPrefix(result.Installed[0].Version, "v")
	if v != "4.5.0" {
		t.Errorf("laravel version: got %q", v)
	}
	if result.Installed[1].Name != "phpunit/phpunit" {
		t.Errorf("pkg 1 name: got %q", result.Installed[1].Name)
	}
}

func TestComposerOutdatedJSON(t *testing.T) {
	data := `{
		"installed": [
			{"name": "phpunit/phpunit", "version": "10.5.17", "latest": "11.3.1", "description": "Testing"},
			{"name": "sebastian/comparator", "version": "5.0.1", "latest": "6.0.0", "description": "Compare"}
		]
	}`

	var result struct {
		Installed []struct {
			Name   string `json:"name"`
			Latest string `json:"latest"`
		} `json:"installed"`
	}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	updates := make(map[string]string)
	for _, p := range result.Installed {
		updates[p.Name] = strings.TrimPrefix(p.Latest, "v")
	}
	if updates["phpunit/phpunit"] != "11.3.1" {
		t.Errorf("phpunit: got %q", updates["phpunit/phpunit"])
	}
}
