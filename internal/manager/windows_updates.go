package manager

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/neur0map/glazepkg/internal/model"
)

// WindowsUpdates surfaces pending Windows system updates via the Windows Update API.
// Requires Windows 10/11 and PowerShell; gracefully returns nothing on other platforms.
const maxWindowsUpdates = 50 // guard against unexpectedly large update lists

type WindowsUpdates struct{}

func (w *WindowsUpdates) Name() model.Source { return model.SourceWindowsUpdates }

func (w *WindowsUpdates) Available() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	return commandExists("powershell") || commandExists("pwsh")
}

func (w *WindowsUpdates) psExe() string {
	if commandExists("pwsh") {
		return "pwsh"
	}
	return "powershell"
}

type winUpdate struct {
	Title      string `json:"Title"`
	KBArticle  string `json:"KBArticle"`
	Size       int64  `json:"Size"`
	Severity   string `json:"Severity"`
	Categories string `json:"Categories"`
}

// Scan queries for pending Windows updates via the Windows Update COM API.
// Returns one Package entry per pending update; returns nil on any failure.
func (w *WindowsUpdates) Scan() ([]model.Package, error) {
	script := `
$ErrorActionPreference = 'Stop'
try {
    $session  = New-Object -ComObject "Microsoft.Update.Session"
    $searcher = $session.CreateUpdateSearcher()
    $result   = $searcher.Search("IsInstalled=0")  # WU COM query syntax: pending (not yet installed) updates
    $list = @()
    foreach ($u in $result.Updates) {
        $kb  = $(if ($u.KBArticleIDs.Count -gt 0) { $u.KBArticleIDs[0] } else { "N/A" })
        $sev = $(if ($u.MsrcSeverity)              { $u.MsrcSeverity }    else { "Unspecified" })
        $cat = $(if ($u.Categories.Count -gt 0)   { $u.Categories[0].Name } else { "System Update" })
        $list += @{
            Title      = $u.Title
            KBArticle  = $kb
            Size       = [long]$u.MaxDownloadSize
            Severity   = $sev
            Categories = $cat
        }
    }
    if ($list.Count -eq 0) { Write-Output "[]"; exit 0 }
    ConvertTo-Json -InputObject @($list) -Compress  # @() required: single item would otherwise emit an object, not an array
} catch {
    Write-Output "[]"
    exit 0
}
`
	out, err := exec.Command(w.psExe(), "-NoProfile", "-Command", script).Output()
	if err != nil {
		// WU is unreliable on corporate/offline/policy-restricted machines.
		// A PowerShell execution failure is expected degradation — return nothing silently.
		return nil, nil
	}

	var updates []winUpdate
	if err := json.Unmarshal(out, &updates); err != nil {
		// JSON parse failure is unexpected: the script returned malformed output.
		// Surface this so callers can distinguish it from a clean empty result.
		return nil, err
	}
	if len(updates) == 0 {
		return nil, nil
	}
	return w.buildPackages(updates), nil
}

// buildPackages converts a slice of winUpdate entries into model.Package values.
// Extracted so tests can exercise the mapping logic without invoking PowerShell.
func (w *WindowsUpdates) buildPackages(updates []winUpdate) []model.Package {
	pkgs := make([]model.Package, 0, len(updates))
	for i, u := range updates {
		if i >= maxWindowsUpdates {
			break
		}
		name := u.Title
		if u.KBArticle != "N/A" && u.KBArticle != "" {
			name = fmt.Sprintf("%s [KB%s]", u.Title, u.KBArticle)
		}

		desc := u.Categories
		if u.Severity != "Unspecified" && u.Severity != "" {
			desc = fmt.Sprintf("%s — %s", u.Severity, u.Categories)
		}

		version := u.KBArticle
		if version == "" || version == "N/A" {
			version = "unknown"
		}

		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Description: desc,
			Size:        FormatBytes(u.Size),
			SizeBytes:   u.Size,
			Source:      model.SourceWindowsUpdates,
			Repository:  u.Categories,
		})
	}
	return pkgs
}
