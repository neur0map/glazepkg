package manager

import (
	"encoding/json"
	"os/exec"
	"sync"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// PowerShell surfaces installed PowerShell modules via Get-Module -ListAvailable.
// Works with both Windows PowerShell (powershell.exe) and PowerShell Core (pwsh).
// pwsh runs on macOS and Linux too, so this manager is not Windows-only by design.
type PowerShell struct{}

func (ps *PowerShell) Name() model.Source { return model.SourcePowerShell }

func (ps *PowerShell) Available() bool {
	return commandExists("pwsh") || commandExists("powershell")
}

func (ps *PowerShell) exe() string {
	if commandExists("pwsh") {
		return "pwsh"
	}
	return "powershell"
}

// psModule holds a single module entry from PowerShell JSON output.
type psModule struct {
	Name        string `json:"Name"`
	Version     string `json:"Version"`
	Description string `json:"Description"`
}

// Package-level cache so Describe() can reuse data fetched during Scan()
// without spawning a second PowerShell process.
var (
	psModCache   map[string]psModule
	psModCacheMu sync.RWMutex
)

// Scan lists installed PowerShell modules.
// Fetches Name, Version, and Description in a single invocation and caches
// the result for Describe() to consume.
func (ps *PowerShell) Scan() ([]model.Package, error) {
	script := `
$m = Get-Module -ListAvailable | ForEach-Object {
    [PSCustomObject]@{
        Name        = $_.Name
        Version     = $(if ($_.Version) { $_.Version.ToString() } else { "" })
        Description = $_.Description
    }
}
if ($m -is [array]) { ConvertTo-Json -InputObject $m -Compress }
else { ConvertTo-Json -InputObject @($m) -Compress }
`
	// Note: the @() wrapper in the script is required — PowerShell's ConvertTo-Json
	// emits a bare object (not an array) when there is only one result.
	out, err := exec.Command(ps.exe(), "-NoProfile", "-Command", script).Output()
	if err != nil {
		return nil, err
	}

	var modules []psModule
	if err := json.Unmarshal(out, &modules); err != nil {
		return nil, err
	}

	// Deduplicate: keep only the highest version per module name
	best := make(map[string]psModule, len(modules))
	for _, m := range modules {
		if m.Name == "" {
			continue
		}
		if prev, ok := best[m.Name]; !ok || nugetCompare(m.Version, prev.Version) > 0 {
			best[m.Name] = m
		}
	}

	// Populate the shared description cache
	psModCacheMu.Lock()
	psModCache = best
	psModCacheMu.Unlock()

	pkgs := make([]model.Package, 0, len(best))
	for _, m := range best {
		v := m.Version
		if v == "" {
			v = "unknown"
		}
		pkgs = append(pkgs, model.Package{
			Name:        m.Name,
			Description: m.Description,
			Version:     v,
			Source:      model.SourcePowerShell,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (ps *PowerShell) RemoveCmd(name string) *exec.Cmd {
	return exec.Command(ps.exe(), "-Command", "Uninstall-Module -Name "+name+" -Force")
}

// Describe returns module descriptions, reusing cached Scan() data when available
// to avoid spawning a second PowerShell process.
func (ps *PowerShell) Describe(pkgs []model.Package) map[string]string {
	psModCacheMu.RLock()
	cache := psModCache
	psModCacheMu.RUnlock()

	if cache != nil {
		descs := make(map[string]string, len(pkgs))
		for _, p := range pkgs {
			if m, ok := cache[p.Name]; ok && m.Description != "" {
				descs[p.Name] = m.Description
			}
		}
		return descs
	}

	// Fallback: run a targeted describe script
	script := `
$m = Get-Module -ListAvailable | Select-Object -Property Name, Description
if ($m -is [array]) { ConvertTo-Json -InputObject $m -Compress }
else { ConvertTo-Json -InputObject @($m) -Compress }
`
	out, err := exec.Command(ps.exe(), "-NoProfile", "-Command", script).Output()
	if err != nil {
		return make(map[string]string)
	}

	var modules []psModule
	if err := json.Unmarshal(out, &modules); err != nil {
		return make(map[string]string)
	}

	descs := make(map[string]string, len(modules))
	for _, m := range modules {
		if m.Name != "" && m.Description != "" {
			descs[m.Name] = m.Description
		}
	}
	return descs
}
