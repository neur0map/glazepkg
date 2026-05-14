package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestEnvelopeSchema(t *testing.T) {
	var buf bytes.Buffer
	err := writeEnvelope(&buf, "0.6.4", []int{1, 2, 3})
	if err != nil {
		t.Fatalf("writeEnvelope: %v", err)
	}
	var got struct {
		GpkVersion string `json:"gpk_version"`
		Schema     int    `json:"schema"`
		Data       []int  `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Schema != 1 {
		t.Errorf("schema = %d, want 1", got.Schema)
	}
	if got.GpkVersion != "0.6.4" {
		t.Errorf("gpk_version = %q, want %q", got.GpkVersion, "0.6.4")
	}
	if len(got.Data) != 3 {
		t.Errorf("data length = %d, want 3", len(got.Data))
	}
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Error("envelope output missing trailing newline")
	}
}

func TestToCLIPackageCopiesLatestVersion(t *testing.T) {
	p := model.Package{
		Name:          "foo",
		Version:       "1.0",
		Source:        model.SourcePacman,
		LatestVersion: "1.1",
		InstalledAt:   time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC),
	}
	c := toCLIPackage(p)
	if c.LatestVersion != "1.1" {
		t.Errorf("LatestVersion = %q, want %q", c.LatestVersion, "1.1")
	}
	if c.Name != "foo" || c.Version != "1.0" || c.Source != model.SourcePacman {
		t.Errorf("field copy wrong: %+v", c)
	}
}

func TestCLIPackageJSONIncludesLatestVersion(t *testing.T) {
	c := toCLIPackage(model.Package{
		Name:          "foo",
		Version:       "1.0",
		Source:        model.SourceBrew,
		LatestVersion: "1.1",
		InstalledAt:   time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC),
	})
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"latest_version":"1.1"`) {
		t.Errorf("JSON missing latest_version: %s", data)
	}
}

func TestCLIPackageJSONOmitsEmptyLatestVersion(t *testing.T) {
	c := toCLIPackage(model.Package{
		Name:        "foo",
		Version:     "1.0",
		Source:      model.SourceBrew,
		InstalledAt: time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC),
	})
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "latest_version") {
		t.Errorf("JSON should omit empty latest_version: %s", data)
	}
}

func TestStripANSI(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"plain", "plain"},
		{"\x1b[31mred\x1b[0m", "red"},
		{"a\x1b[1;32mb\x1b[0mc", "abc"},
	}
	for _, c := range cases {
		got := stripANSI(c.in)
		if got != c.want {
			t.Errorf("stripANSI(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
