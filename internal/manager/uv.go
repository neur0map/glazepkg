package manager

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

var uvHTTPClient = &http.Client{Timeout: 10 * time.Second}

var uvBaseURL = "https://pypi.org"

const uvMaxWorkers = 10

const uvMaxRetries = 3

// Uv manages Python tools installed via uv (https://docs.astral.sh/uv/).
type Uv struct{}

func (u *Uv) Name() model.Source { return model.SourceUv }

func (u *Uv) Available() bool { return commandExists("uv") }

func (u *Uv) Scan() ([]model.Package, error) {
	out, err := exec.Command("uv", "tool", "list").Output()
	if err != nil {
		return nil, err
	}
	return parseUvToolList(out)
}

// parseUvToolList parses the output of "uv tool list".
//
// Output format:
//
//	name v1.2.3
//	- executable1
//	- executable2
//	another-tool v4.5.6
//	- another-tool
func parseUvToolList(out []byte) ([]model.Package, error) {
	var pkgs []model.Package
	now := time.Now()
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ver := parts[1]
		if strings.HasPrefix(ver, "v") {
			ver = ver[1:]
		}
		if len(ver) == 0 || ver[0] < '0' || ver[0] > '9' {
			continue
		}

		pkgs = append(pkgs, model.Package{
			Name:        parts[0],
			Version:     ver,
			Source:      model.SourceUv,
			InstalledAt: now,
		})
	}
	if err := scanner.Err(); err != nil {
		return pkgs, fmt.Errorf("uv tool list: scan: %w", err)
	}
	return pkgs, nil
}

func (u *Uv) CheckUpdates(pkgs []model.Package) map[string]string {
	infos := pypiVersionBatch(pkgs)
	updates := make(map[string]string)
	for _, p := range pkgs {
		latest := infos[p.Name]
		if latest != "" && latest != p.Version {
			updates[p.Name] = latest
		}
	}
	return updates
}

// Describe reads package summaries from local dist-info METADATA files
// inside each uv tool's virtual environment. No network calls needed.
func (u *Uv) Describe(pkgs []model.Package) map[string]string {
	toolDir := uvToolDir()
	if toolDir == "" {
		return nil
	}
	descs := make(map[string]string, len(pkgs))
	for _, p := range pkgs {
		if desc := uvLocalSummary(toolDir, p.Name); desc != "" {
			descs[p.Name] = desc
		}
	}
	return descs
}

var (
	uvToolDirMu   sync.Mutex
	uvToolDirVal  string
	uvToolDirDone bool
)

func uvToolDir() string {
	uvToolDirMu.Lock()
	defer uvToolDirMu.Unlock()
	if uvToolDirDone {
		return uvToolDirVal
	}
	out, err := exec.Command("uv", "tool", "dir").Output()
	if err == nil {
		uvToolDirVal = strings.TrimSpace(string(out))
		uvToolDirDone = true
	}
	return uvToolDirVal
}

// uvLocalSummary reads the Summary field from a tool's installed METADATA.
// The dist-info directory uses the normalized package name (hyphens to
// underscores) per PEP 427.
func uvLocalSummary(toolDir, name string) string {
	normalized := strings.ReplaceAll(name, "-", "_")
	pattern := filepath.Join(toolDir, name, "lib", "python*", "site-packages", normalized+"-*.dist-info", "METADATA")
	matches, err := filepath.Glob(pattern)
	if (err != nil || len(matches) == 0) && normalized != name {
		pattern = filepath.Join(toolDir, name, "lib", "python*", "site-packages", name+"-*.dist-info", "METADATA")
		matches, _ = filepath.Glob(pattern)
	}
	if len(matches) == 0 {
		return ""
	}
	return parseMetadataSummary(matches[0])
}

// parseMetadataSummary extracts the Summary field from a PEP 566 METADATA file.
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
			break // End of headers, body follows.
		}
		if after, ok := strings.CutPrefix(line, "Summary: "); ok {
			return after
		}
	}
	return ""
}

// pypiVersionBatch fetches latest versions from PyPI concurrently.
func pypiVersionBatch(pkgs []model.Package) map[string]string {
	results := make(map[string]string, len(pkgs))
	if len(pkgs) == 0 {
		return results
	}

	var mu sync.Mutex
	sem := make(chan struct{}, uvMaxWorkers)
	var wg sync.WaitGroup

	for _, p := range pkgs {
		wg.Add(1)
		sem <- struct{}{}
		go func(name string) {
			defer wg.Done()
			defer func() { <-sem }()
			ver := pypiLatestVersion(name)
			if ver != "" {
				mu.Lock()
				results[name] = ver
				mu.Unlock()
			}
		}(p.Name)
	}
	wg.Wait()
	return results
}

// pypiLatestVersion queries PyPI for the latest version of a package,
// retrying with exponential backoff on HTTP 429 (rate limit).
func pypiLatestVersion(name string) string {
	apiURL := uvBaseURL + "/pypi/" + name + "/json"
	backoff := 500 * time.Millisecond

	for attempt := range uvMaxRetries {
		resp, err := uvHTTPClient.Get(apiURL)
		if err != nil {
			return ""
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			if attempt < uvMaxRetries-1 {
				time.Sleep(backoff)
				backoff *= 2
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return ""
		}

		var result struct {
			Info struct {
				Version string `json:"version"`
			} `json:"info"`
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if err != nil {
			return ""
		}
		return result.Info.Version
	}
	return ""
}

func (u *Uv) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("uv", "tool", "install", name+"@latest")
}

func (u *Uv) InstallCmd(name string) *exec.Cmd {
	return exec.Command("uv", "tool", "install", name)
}

func (u *Uv) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("uv", "tool", "uninstall", name)
}
