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
	if latest == currentVersion {
		return latest, fmt.Errorf("already up to date (%s)", latest)
	}

	assetName := binaryName()
	var downloadURL string
	for _, a := range rel.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return latest, fmt.Errorf("no binary found for %s/%s in release %s", runtime.GOOS, runtime.GOARCH, latest)
	}

	execPath, err := os.Executable()
	if err != nil {
		return latest, fmt.Errorf("cannot find current binary path: %w", err)
	}
	// Resolve symlinks
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
	resp, err := http.Get(repoAPI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github API returned %s", resp.Status)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func binaryName() string {
	name := fmt.Sprintf("gpk-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func resolveExecPath(execPath string) (string, error) {
	info, err := os.Lstat(execPath)
	if err != nil {
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
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned %s", resp.Status)
	}

	// Write to a temp file next to the binary, then atomic rename
	tmpPath := destPath + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		// If we can't write next to it, might need sudo
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

	// Atomic replace
	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cannot replace binary: %w", err)
	}

	return nil
}
