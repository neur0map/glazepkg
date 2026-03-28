package manager

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// --- parseUvToolList ---

func TestParseUvToolList(t *testing.T) {
	output := []byte("posting v2.10.0\n- posting\nruff v0.15.8\n- ruff\nblack v24.4.2\n- black\n- blackd\n")
	pkgs, err := parseUvToolList(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(pkgs))
	}
	expected := []struct{ name, version string }{
		{"posting", "2.10.0"},
		{"ruff", "0.15.8"},
		{"black", "24.4.2"},
	}
	for i, exp := range expected {
		if pkgs[i].Name != exp.name || pkgs[i].Version != exp.version {
			t.Errorf("pkg %d: got {%s %s}, want {%s %s}", i, pkgs[i].Name, pkgs[i].Version, exp.name, exp.version)
		}
	}
}

func TestParseUvToolListEmpty(t *testing.T) {
	pkgs, err := parseUvToolList([]byte("No tools installed\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("expected 0 tools, got %d", len(pkgs))
	}
}

func TestParseUvToolListVersionWithoutPrefix(t *testing.T) {
	pkgs, err := parseUvToolList([]byte("mytool 1.0.0\n- mytool\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(pkgs))
	}
	if pkgs[0].Name != "mytool" || pkgs[0].Version != "1.0.0" {
		t.Errorf("got {%s %s}, want {mytool 1.0.0}", pkgs[0].Name, pkgs[0].Version)
	}
}

// --- pypiLatestVersion ---

func withTestPyPI(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	origURL := uvBaseURL
	origClient := uvHTTPClient
	uvBaseURL = srv.URL
	uvHTTPClient = srv.Client()
	t.Cleanup(func() {
		uvBaseURL = origURL
		uvHTTPClient = origClient
	})
}

func TestPypiLatestVersionSuccess(t *testing.T) {
	withTestPyPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pypi/ruff/json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"info": map[string]any{"version": "0.15.8"},
		})
	})
	ver := pypiLatestVersion("ruff")
	if ver != "0.15.8" {
		t.Errorf("version: got %q, want %q", ver, "0.15.8")
	}
}

func TestPypiLatestVersionNotFound(t *testing.T) {
	withTestPyPI(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	ver := pypiLatestVersion("nonexistent")
	if ver != "" {
		t.Errorf("expected empty version for 404, got %q", ver)
	}
}

func TestPypiLatestVersionRateLimitThenSuccess(t *testing.T) {
	var attempts atomic.Int32
	withTestPyPI(t, func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"info": map[string]any{"version": "1.0.0"},
		})
	})
	ver := pypiLatestVersion("testpkg")
	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
	if ver != "1.0.0" {
		t.Errorf("version: got %q, want %q", ver, "1.0.0")
	}
}

func TestPypiLatestVersionRateLimitExhausted(t *testing.T) {
	var attempts atomic.Int32
	withTestPyPI(t, func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	})
	ver := pypiLatestVersion("testpkg")
	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
	if ver != "" {
		t.Errorf("expected empty version, got %q", ver)
	}
}

func TestPypiLatestVersionTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	t.Cleanup(srv.Close)
	origURL := uvBaseURL
	origClient := uvHTTPClient
	uvBaseURL = srv.URL
	uvHTTPClient = &http.Client{Timeout: 50 * time.Millisecond}
	t.Cleanup(func() {
		uvBaseURL = origURL
		uvHTTPClient = origClient
	})
	ver := pypiLatestVersion("slowpkg")
	if ver != "" {
		t.Errorf("expected empty version on timeout, got %q", ver)
	}
}

func TestPypiLatestVersionMalformedJSON(t *testing.T) {
	withTestPyPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"info": broken`))
	})
	ver := pypiLatestVersion("badpkg")
	if ver != "" {
		t.Errorf("expected empty version on malformed JSON, got %q", ver)
	}
}

func TestPypiVersionBatchConcurrency(t *testing.T) {
	var inflight atomic.Int32
	var maxInflight atomic.Int32

	withTestPyPI(t, func(w http.ResponseWriter, r *http.Request) {
		cur := inflight.Add(1)
		defer inflight.Add(-1)
		for {
			old := maxInflight.Load()
			if cur <= old || maxInflight.CompareAndSwap(old, cur) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"info": map[string]any{"version": "1.0.0"},
		})
	})

	pkgs := make([]modelPkg, 25)
	for i := range pkgs {
		pkgs[i] = modelPkg{name: "pkg" + string(rune('a'+i))}
	}
	results := pypiVersionBatch(toModelPkgs(pkgs))

	if len(results) != 25 {
		t.Errorf("expected 25 results, got %d", len(results))
	}
	if peak := maxInflight.Load(); peak > int32(uvMaxWorkers) {
		t.Errorf("peak concurrency %d exceeded max workers %d", peak, uvMaxWorkers)
	}
}

type modelPkg struct{ name string }

func toModelPkgs(pkgs []modelPkg) []model.Package {
	out := make([]model.Package, len(pkgs))
	for i, p := range pkgs {
		out[i] = model.Package{Name: p.name, Source: model.SourceUv}
	}
	return out
}

// --- parseMetadataSummary ---

func TestParseMetadataSummary(t *testing.T) {
	content := "Metadata-Version: 2.4\nName: posting\nVersion: 2.10.0\nSummary: The modern API client that lives in your terminal.\nProject-URL: Homepage, https://example.com\n\nLong description here.\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "METADATA")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	got := parseMetadataSummary(path)
	want := "The modern API client that lives in your terminal."
	if got != want {
		t.Errorf("summary: got %q, want %q", got, want)
	}
}

func TestParseMetadataSummaryMissing(t *testing.T) {
	content := "Metadata-Version: 2.4\nName: nosummary\nVersion: 1.0.0\n\nBody text.\n"
	dir := t.TempDir()
	path := filepath.Join(dir, "METADATA")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	got := parseMetadataSummary(path)
	if got != "" {
		t.Errorf("expected empty summary, got %q", got)
	}
}

func TestParseMetadataSummaryFileNotFound(t *testing.T) {
	got := parseMetadataSummary("/nonexistent/METADATA")
	if got != "" {
		t.Errorf("expected empty summary for missing file, got %q", got)
	}
}
