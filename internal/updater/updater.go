package updater

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	repoAPI     = "https://api.github.com/repos/neur0map/glazepkg/releases/latest"
	releasesURL = "https://github.com/neur0map/glazepkg/releases/latest"
)

var httpClient = &http.Client{Timeout: 60 * time.Second}

// ErrUpToDate is returned by Update when the installed version is already latest.
var ErrUpToDate = errors.New("already up to date")

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
		return latest, ErrUpToDate
	}

	assetName := binaryName()
	var downloadURL, checksumURL string
	for _, a := range rel.Assets {
		switch a.Name {
		case assetName:
			downloadURL = a.BrowserDownloadURL
		case "checksums.txt":
			checksumURL = a.BrowserDownloadURL
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

	if err := downloadAndReplace(downloadURL, resolved, checksumURL); err != nil {
		return latest, err
	}

	return latest, nil
}

func fetchRelease() (*ghRelease, error) {
	resp, err := httpClient.Get(repoAPI)
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
	resolved, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return execPath, nil
	}
	return resolved, nil
}

// downloadAndReplace stages the new binary in the system temp directory and
// then hands off to the platform-specific replaceBinary. Staging in os.TempDir
// rather than beside the destination means the download itself never needs
// write access to the install directory — we only need that permission for
// the final swap, and any failure there produces a clear, actionable error.
func downloadAndReplace(url, destPath, checksumURL string) error {
	tmpPath, err := stageDownload(url)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath) // no-op once replaceBinary succeeds

	if err := verifyChecksum(tmpPath, binaryName(), checksumURL); err != nil {
		return err
	}
	if err := replaceBinary(tmpPath, destPath); err != nil {
		return wrapReplaceErr(err, destPath)
	}
	return nil
}

// verifyChecksum compares the sha256 of file against the hash listed for
// assetName in the release's checksums.txt. It is skipped when no checksum
// URL is available, the file can't be fetched, or the asset isn't listed, so
// releases without a checksums file still update; only a real hash mismatch
// aborts the install.
func verifyChecksum(file, assetName, checksumURL string) error {
	if checksumURL == "" {
		return nil
	}
	resp, err := httpClient.Get(checksumURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	want := checksumFor(string(data), assetName)
	if want == "" {
		return nil
	}
	got, err := sha256File(file)
	if err != nil {
		return fmt.Errorf("cannot verify download: %w", err)
	}
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch for %s: refusing to install a corrupt or tampered binary", assetName)
	}
	return nil
}

// checksumFor returns the hash listed for name in sha256sum output, where each
// line is "<hash>  <name>", or "" when not present.
func checksumFor(checksums, name string) string {
	for _, line := range strings.Split(checksums, "\n") {
		if fields := strings.Fields(line); len(fields) == 2 && fields[1] == name {
			return fields[0]
		}
	}
	return ""
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// stageDownload streams url into a new file in the system temp dir and
// returns its path. Caller is responsible for removing it.
func stageDownload(url string) (string, error) {
	resp, err := httpClient.Get(url)
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

// wrapReplaceErr maps permission errors to platform-appropriate guidance via
// errors.Is against fs.ErrPermission.
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
