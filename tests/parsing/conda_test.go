package parsing

import (
	"encoding/json"
	"testing"
)

func TestCondaListJSON(t *testing.T) {
	data := `[
		{"name": "numpy", "version": "1.23.3", "build": "py310hd5efca6_1", "channel": "defaults"},
		{"name": "python", "version": "3.10.6", "build": "haa1d7c7_1", "channel": "defaults"}
	]`

	var entries []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(data), &entries); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Name != "numpy" || entries[0].Version != "1.23.3" {
		t.Errorf("entry 0: %+v", entries[0])
	}
}

func TestCondaUpdateDryRunJSON(t *testing.T) {
	data := `{
		"actions": {
			"LINK": [
				{"name": "numpy", "version": "1.24.0"},
				{"name": "python", "version": "3.10.8"}
			],
			"UNLINK": [
				{"name": "numpy", "version": "1.23.3"},
				{"name": "python", "version": "3.10.6"}
			]
		},
		"dry_run": true,
		"success": true
	}`

	var result struct {
		Actions struct {
			Link []struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"LINK"`
			Unlink []struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"UNLINK"`
		} `json:"actions"`
	}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	installed := make(map[string]string)
	for _, p := range result.Actions.Unlink {
		installed[p.Name] = p.Version
	}
	updates := make(map[string]string)
	for _, p := range result.Actions.Link {
		if oldVer, ok := installed[p.Name]; ok && oldVer != p.Version {
			updates[p.Name] = p.Version
		}
	}

	if updates["numpy"] != "1.24.0" {
		t.Errorf("numpy: got %q", updates["numpy"])
	}
	if updates["python"] != "3.10.8" {
		t.Errorf("python: got %q", updates["python"])
	}
}
