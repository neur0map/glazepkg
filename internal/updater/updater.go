package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
)

const repoAPI = "https://api.github.com/repos/neur0map/glazepkg/releases/latest"

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// LatestVersion fetches the latest release tag from GitHub.
func LatestVersion() (string, error) {
	rel, err := fetchRelease()
	if err != nil {
		return "", err
	}
	return rel.TagName, nil
}

// Update downloads the latest release binary and replaces the current executable.
// Returns the new version string.
func Update(currentVersion string) (string, error) {
	rel, err := fetchRelease()
	if err != nil {
		return "", fmt.Errorf("failed to check for updates: %w", err)
	}

	latest := rel.TagName

	// Normalize both versions for comparison: strip leading "v"
	if normalizeVersion(latest) == normalizeVersion(currentVersion) {
		return latest, fmt.Errorf("already up to date (%s)", latest)
	}

	assetName := binaryName()
	var downloadURL string
	for _, a := range rel.Assets {
		// Case-insensitive match to be safe
		if strings.EqualFold(a.Name, assetName) {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		// Build a helpful list of what IS available
		var available []string
		for _, a := range rel.Assets {
			available = append(available, a.Name)
		}
		return latest, fmt.Errorf(
			"no binary found for %s/%s in release %s (looked for %q)\navailable assets: %s",
			runtime.GOOS, runtime.GOARCH, latest, assetName,
			strings.Join(available, ", "),
		)
	}

	execPath, err := os.Executable()
	if err != nil {
		return latest, fmt.Errorf("cannot find current binary path: %w", err)
	}

	resolved, err := resolveExecPath(execPath)
	if err != nil {
		return latest, err
	}

	if err := downloadAndReplace(downloadURL, resolved); err != nil {
		return latest, err
	}

	return latest, nil
}

func fetchRelease() (*ghRelease, error) {
	req, err := http.NewRequest(http.MethodGet, repoAPI, nil)
	if err != nil {
		return nil, err
	}
	// Ask GitHub for v3 JSON explicitly; also avoids rate-limit issues
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github API returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// binaryName returns the exact asset filename used in GitHub releases.
//
// Naming convention in releases:
//
//	Linux / macOS : gpk-{os}-{arch}          (no extension)
//	Windows       : gpk-{os}-{arch}.exe
func binaryName() string {
	name := fmt.Sprintf("gpk-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// normalizeVersion strips a leading "v" so "v0.3.20" == "0.3.20".
func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func resolveExecPath(execPath string) (string, error) {
	info, err := os.Lstat(execPath)
	if err != nil {
		// Path not stat-able; proceed with raw path
		return execPath, nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		resolved, err := os.Readlink(execPath)
		if err != nil {
			return "", fmt.Errorf("cannot resolve symlink: %w", err)
		}
		return resolved, nil
	}
	return execPath, nil
}

func downloadAndReplace(url, destPath string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("cannot build download request: %w", err)
	}
	// GitHub redirects release asset downloads; Go's http client follows them,
	// but setting Accept here avoids any accidental HTML error pages.
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %s", resp.Status)
	}

	// Write to a temp file next to the binary, then atomic rename.
	tmpPath := destPath + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			return fmt.Errorf("permission denied — try: sudo gpk update")
		}
		return fmt.Errorf("cannot create temp file: %w", err)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("download interrupted: %w", err)
	}
	f.Close()

	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cannot replace binary: %w", err)
	}

	return nil
}
