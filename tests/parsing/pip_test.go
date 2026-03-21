package parsing

import (
	"encoding/json"
	"testing"
)

func TestPipListJSON(t *testing.T) {
	data := `[
		{"name": "requests", "version": "2.31.0"},
		{"name": "numpy", "version": "1.24.3"},
		{"name": "pip", "version": "23.3.1"}
	]`

	var entries []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal([]byte(data), &entries); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Name != "requests" || entries[0].Version != "2.31.0" {
		t.Errorf("entry 0: %+v", entries[0])
	}
}

func TestPipOutdatedJSON(t *testing.T) {
	data := `[
		{"name": "requests", "version": "2.31.0", "latest_version": "2.32.0", "latest_filetype": "wheel"},
		{"name": "numpy", "version": "1.24.3", "latest_version": "1.26.0", "latest_filetype": "wheel"}
	]`

	var entries []struct {
		Name          string `json:"name"`
		LatestVersion string `json:"latest_version"`
	}
	if err := json.Unmarshal([]byte(data), &entries); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	updates := make(map[string]string)
	for _, e := range entries {
		updates[e.Name] = e.LatestVersion
	}
	if updates["requests"] != "2.32.0" {
		t.Errorf("requests: got %q", updates["requests"])
	}
	if updates["numpy"] != "1.26.0" {
		t.Errorf("numpy: got %q", updates["numpy"])
	}
}
