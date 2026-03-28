package parsing

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- PyPI version fetching (mirrors pypiLatestVersion in uv.go) ---

type pypiResponse struct {
	Info struct {
		Version string `json:"version"`
	} `json:"info"`
}

func fetchPypiVersion(client *http.Client, baseURL, name string, maxRetries int) string {
	url := baseURL + "/pypi/" + name + "/json"
	backoff := 50 * time.Millisecond

	for attempt := range maxRetries {
		resp, err := client.Get(url)
		if err != nil {
			return ""
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			if attempt < maxRetries-1 {
				time.Sleep(backoff)
				backoff *= 2
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return ""
		}

		var result pypiResponse
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if err != nil {
			return ""
		}
		return result.Info.Version
	}
	return ""
}

func TestPypiVersionSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pypi/ruff/json" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pypiResponse{
			Info: struct {
				Version string `json:"version"`
			}{
				Version: "0.15.8",
			},
		})
	}))
	defer srv.Close()

	ver := fetchPypiVersion(srv.Client(), srv.URL, "ruff", 3)
	if ver != "0.15.8" {
		t.Errorf("version: got %q, want %q", ver, "0.15.8")
	}
}

func TestPypiVersionNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	ver := fetchPypiVersion(srv.Client(), srv.URL, "nonexistent", 3)
	if ver != "" {
		t.Errorf("expected empty version for 404, got %q", ver)
	}
}

func TestPypiVersionRateLimitThenSuccess(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pypiResponse{
			Info: struct {
				Version string `json:"version"`
			}{
				Version: "1.0.0",
			},
		})
	}))
	defer srv.Close()

	ver := fetchPypiVersion(srv.Client(), srv.URL, "testpkg", 3)

	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
	if ver != "1.0.0" {
		t.Errorf("version: got %q, want %q", ver, "1.0.0")
	}
}

func TestPypiVersionRateLimitExhausted(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	ver := fetchPypiVersion(srv.Client(), srv.URL, "testpkg", 3)

	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
	if ver != "" {
		t.Errorf("expected empty version after exhausting retries, got %q", ver)
	}
}

func TestPypiVersionTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 50 * time.Millisecond}
	ver := fetchPypiVersion(client, srv.URL, "slowpkg", 1)

	if ver != "" {
		t.Errorf("expected empty version on timeout, got %q", ver)
	}
}

func TestPypiVersionMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"info": broken`))
	}))
	defer srv.Close()

	ver := fetchPypiVersion(srv.Client(), srv.URL, "badpkg", 1)

	if ver != "" {
		t.Errorf("expected empty version on malformed JSON, got %q", ver)
	}
}

func TestPypiVersionConcurrentBatch(t *testing.T) {
	var inflight atomic.Int32
	var maxInflight atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		json.NewEncoder(w).Encode(pypiResponse{
			Info: struct {
				Version string `json:"version"`
			}{
				Version: "1.0.0",
			},
		})
	}))
	defer srv.Close()

	const numPkgs = 25
	const maxWorkers = 10

	type pkg struct{ Name string }
	pkgs := make([]pkg, numPkgs)
	for i := range pkgs {
		pkgs[i] = pkg{Name: "pkg" + string(rune('a'+i))}
	}

	results := make(map[string]string, numPkgs)
	sem := make(chan struct{}, maxWorkers)
	done := make(chan struct{})
	go func() {
		mu := new(sync.Mutex)
		var wg sync.WaitGroup
		for _, p := range pkgs {
			wg.Add(1)
			sem <- struct{}{}
			go func(name string) {
				defer wg.Done()
				defer func() { <-sem }()
				ver := fetchPypiVersion(srv.Client(), srv.URL, name, 1)
				mu.Lock()
				results[name] = ver
				mu.Unlock()
			}(p.Name)
		}
		wg.Wait()
		close(done)
	}()
	<-done

	if len(results) != numPkgs {
		t.Errorf("expected %d results, got %d", numPkgs, len(results))
	}
	if peak := maxInflight.Load(); peak > int32(maxWorkers) {
		t.Errorf("peak concurrency %d exceeded max workers %d", peak, maxWorkers)
	}
}

// --- Local METADATA parsing (mirrors parseMetadataSummary in uv.go) ---

func parseMetadataSummary(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		if strings.HasPrefix(line, "Summary: ") {
			return strings.TrimPrefix(line, "Summary: ")
		}
	}
	return ""
}

func TestMetadataSummaryParsing(t *testing.T) {
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

func TestMetadataSummaryMissing(t *testing.T) {
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

func TestMetadataSummaryFileNotFound(t *testing.T) {
	got := parseMetadataSummary("/nonexistent/METADATA")
	if got != "" {
		t.Errorf("expected empty summary for missing file, got %q", got)
	}
}
