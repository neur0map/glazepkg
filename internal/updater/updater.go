package updater

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const (
	repoAPI     = "https://api.github.com/repos/neur0map/glazepkg/releases/latest"
	releasesURL = "https://github.com/neur0map/glazepkg/releases/latest"
)

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

// downloadAndReplace stages the new binary in the system temp directory and
// then hands off to the platform-specific replaceBinary. Staging in os.TempDir
// rather than beside the destination means the download itself never needs
// write access to the install directory — we only need that permission for
// the final swap, and any failure there produces a clear, actionable error.
func downloadAndReplace(url, destPath string) error {
	tmpPath, err := stageDownload(url)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath) // no-op once replaceBinary succeeds

	if err := replaceBinary(tmpPath, destPath); err != nil {
		return wrapReplaceErr(err, destPath)
	}
	return nil
}

// stageDownload streams url into a new file in the system temp dir and
// returns its path. Caller is responsible for removing it.
func stageDownload(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download returned %s", resp.Status)
	}

	f, err := os.CreateTemp("", "gpk-update-*")
	if err != nil {
		return "", fmt.Errorf("cannot stage download: %w", err)
	}
	tmpPath := f.Name()

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("download interrupted: %w", err)
	}
	// Best effort — some filesystems (Windows) ignore unix-style mode bits.
	_ = f.Chmod(0o755)
	if err := f.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("cannot close staged file: %w", err)
	}
	return tmpPath, nil
}

// moveFile renames src to dest, falling back to copy+remove when rename
// fails (e.g. cross-device on Linux when os.TempDir is a different mount
// than the install directory).
func moveFile(src, dest string) error {
	if err := os.Rename(src, dest); err == nil {
		return nil
	}

	srcF, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcF.Close()

	destF, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(destF, srcF); err != nil {
		destF.Close()
		os.Remove(dest)
		return err
	}
	if err := destF.Close(); err != nil {
		os.Remove(dest)
		return err
	}
	return os.Remove(src)
}

// wrapReplaceErr turns raw permission errors into platform-appropriate
// guidance. Uses errors.Is against fs.ErrPermission so it works on every
// OS — the previous string match ("permission denied") never fired on
// Windows, where os.OpenFile returns "Access is denied".
func wrapReplaceErr(err error, destPath string) error {
	if !errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("cannot replace binary: %w", err)
	}
	if runtime.GOOS == "windows" {
		return fmt.Errorf(
			"cannot write to %s — download the latest installer from %s and run it to update",
			filepath.Dir(destPath), releasesURL,
		)
	}
	return fmt.Errorf("permission denied writing %s — try: sudo gpk update", destPath)
}
