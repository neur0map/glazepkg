package manager

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type AUR struct{}

func (a *AUR) Name() model.Source { return model.SourceAUR }

func (a *AUR) Available() bool { return commandExists("pacman") }

func (a *AUR) Scan() ([]model.Package, error) {
	// Foreign packages = AUR or manually built
	out, err := exec.Command("pacman", "-Qm").Output()
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:        fields[0],
			Version:     fields[1],
			Source:      model.SourceAUR,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (a *AUR) RemoveCmd(name string) *exec.Cmd {
	return privilegedCmd("pacman", "-R", name)
}

func (a *AUR) CheckUpdates(pkgs []model.Package) map[string]string {
	// Check if an AUR helper is available (yay, paru)
	var cmd *exec.Cmd
	if commandExists("yay") {
		cmd = exec.Command("yay", "-Qum")
	} else if commandExists("paru") {
		cmd = exec.Command("paru", "-Qum")
	} else {
		return nil
	}

	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return nil
	}

	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		// Format: "name old_ver -> new_ver" or "name new_ver"
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 4 {
			updates[fields[0]] = fields[3]
		} else if len(fields) >= 2 {
			updates[fields[0]] = fields[1]
		}
	}
	return updates
}

func (a *AUR) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		out, err := exec.Command("pacman", "-Qi", pkg.Name).Output()
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			key, val, ok := parseField(scanner.Text())
			if !ok {
				continue
			}
			if key == "Depends On" {
				if val != "None" {
					deps[pkg.Name] = strings.Fields(val)
				} else {
					deps[pkg.Name] = nil
				}
				break
			}
		}
	}
	return deps
}

func (a *AUR) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		detail, err := QueryDetail(pkg.Name)
		if err == nil && detail.Description != "" {
			descs[pkg.Name] = detail.Description
		}
	}
	return descs
}

func aurHelper() string {
	if commandExists("yay") {
		return "yay"
	}
	if commandExists("paru") {
		return "paru"
	}
	return ""
}

// aurBuildCmd builds and installs an AUR package with makepkg when no helper
// (yay/paru) is present. The name is passed as a positional arg so it can't be
// interpolated into the shell script.
func aurBuildCmd(name string, noconfirm bool) *exec.Cmd {
	mk := "makepkg -si"
	if noconfirm {
		mk += " --noconfirm"
	}
	preflight := `command -v makepkg >/dev/null 2>&1 || { echo "gpk: building from the AUR needs makepkg (pacman -S base-devel) or an AUR helper (yay/paru)" >&2; exit 1; }; `
	script := preflight + `set -e; d=$(mktemp -d); git clone --depth 1 "https://aur.archlinux.org/$1.git" "$d/$1"; cd "$d/$1"; ` + mk
	return exec.Command("sh", "-c", script, "sh", name)
}

// AURWillBuild reports whether gpk would build AUR packages itself with makepkg
// (no yay/paru helper installed), in which case the caller should show the
// PKGBUILD for review first.
func AURWillBuild() bool {
	return commandExists("pacman") && aurHelper() == ""
}

// FetchPKGBUILD returns an AUR package's PKGBUILD so it can be reviewed before
// building. Capped to 64 KiB; an unknown name or split-package base returns an
// error the caller can degrade on.
func FetchPKGBUILD(name string) (string, error) {
	u := "https://aur.archlinux.org/cgit/aur.git/plain/PKGBUILD?h=" + url.QueryEscape(name)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("pkgbuild not found")
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (a *AUR) UpgradeCmd(name string) *exec.Cmd {
	if h := aurHelper(); h != "" {
		return exec.Command(h, "-S", name)
	}
	return aurBuildCmd(name, false)
}

// Search queries the AUR RPC directly so it works without an AUR helper
// installed, returning name, version, and description for matching packages.
func (a *AUR) Search(query string) ([]model.Package, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	u := "https://aur.archlinux.org/rpc/v5/search/" + url.PathEscape(q) + "?by=name-desc"
	client := &http.Client{Timeout: 5 * time.Second}
	var resp *http.Response
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		resp, err = client.Get(u)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("aur rpc: " + resp.Status)
	}
	var body struct {
		Results []struct {
			Name        string `json:"Name"`
			Version     string `json:"Version"`
			Description string `json:"Description"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	pkgs := make([]model.Package, 0, len(body.Results))
	for _, r := range body.Results {
		pkgs = append(pkgs, model.Package{
			Name:        r.Name,
			Version:     r.Version,
			Description: r.Description,
			Source:      model.SourceAUR,
		})
	}
	return pkgs, nil
}

func (a *AUR) InstallCmd(name string) *exec.Cmd {
	if h := aurHelper(); h != "" {
		return exec.Command(h, "-S", name)
	}
	return aurBuildCmd(name, false)
}

func (a *AUR) InstallCmdYes(name string) *exec.Cmd {
	if h := aurHelper(); h != "" {
		return exec.Command(h, "-S", "--noconfirm", name)
	}
	return aurBuildCmd(name, true)
}

func (a *AUR) UpgradeCmdYes(name string) *exec.Cmd {
	if h := aurHelper(); h != "" {
		return exec.Command(h, "-S", "--noconfirm", name)
	}
	return aurBuildCmd(name, true)
}

func (a *AUR) RemoveCmdYes(name string) *exec.Cmd {
	return privilegedCmd("pacman", "-R", "--noconfirm", name)
}
