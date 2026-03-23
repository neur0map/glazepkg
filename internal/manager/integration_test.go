package manager

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/model"
)

// Integration tests call Scan() against the real package manager binary.
// Each test skips automatically when the tool is not installed, so these are
// safe to include in regular `go test ./...` runs on any machine.
//
// To run only integration tests:
//
//	go test -v -run TestIntegration_ ./internal/manager/

func integrationScan(t *testing.T, m Manager) {
	t.Helper()
	if !m.Available() {
		t.Skipf("%s not available on this machine", m.Name())
	}

	pkgs, err := m.Scan()
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	// A freshly-installed package manager with nothing installed is valid.
	// We just check structural invariants on whatever is returned.
	for i, p := range pkgs {
		if p.Name == "" {
			t.Errorf("pkg[%d] has empty Name", i)
		}
		if p.Source != m.Name() {
			t.Errorf("pkg[%d] Source = %q, want %q", i, p.Source, m.Name())
		}
	}

	t.Logf("%s: %d packages found", m.Name(), len(pkgs))
}

func TestIntegration_Winget(t *testing.T)         { integrationScan(t, &Winget{}) }
func TestIntegration_Chocolatey(t *testing.T)     { integrationScan(t, &Chocolatey{}) }
func TestIntegration_Scoop(t *testing.T)          { integrationScan(t, &Scoop{}) }
func TestIntegration_Nuget(t *testing.T)          { integrationScan(t, &Nuget{}) }
func TestIntegration_PowerShell(t *testing.T)     { integrationScan(t, &PowerShell{}) }
func TestIntegration_WindowsUpdates(t *testing.T) { integrationScan(t, &WindowsUpdates{}) }
func TestIntegration_InstalledAtZero(t *testing.T) {
	// Specifically verify the CodeRabbit fix: WindowsUpdates must not populate
	// InstalledAt, since pending updates have not been installed yet.
	w := &WindowsUpdates{}
	if !w.Available() {
		t.Skip("WindowsUpdates not available on this machine")
	}
	pkgs, err := w.Scan()
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	for i, p := range pkgs {
		if !p.InstalledAt.IsZero() {
			t.Errorf("pkg[%d] %q: InstalledAt should be zero for pending updates, got %v", i, p.Name, p.InstalledAt)
		}
	}
}

// Verify that Available() never panics regardless of platform.
func TestIntegration_AvailableNoPanic(t *testing.T) {
	for _, m := range []Manager{
		&Winget{}, &Chocolatey{}, &Scoop{},
		&Nuget{}, &PowerShell{}, &WindowsUpdates{},
	} {
		m := m
		func(m Manager) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s Available() panicked: %v", m.Name(), r)
				}
			}() // nolint:errcheck

			_ = m.Available()
		}(m)
	}
}

// Verify Source values are set correctly on returned packages.
func TestIntegration_SourceField(t *testing.T) {
	for _, m := range []Manager{
		&Winget{}, &Chocolatey{}, &Scoop{},
		&Nuget{}, &PowerShell{}, &WindowsUpdates{},
	} {
		m := m
		t.Run(string(m.Name()), func(t *testing.T) {
			if !m.Available() {
				t.Skipf("%s not available", m.Name())
			}
			pkgs, err := m.Scan()
			if err != nil {
				t.Fatalf("Scan() error: %v", err)
			}
			for i, p := range pkgs {
				if p.Source != m.Name() {
					t.Errorf("pkg[%d]: Source = %q, want %q", i, p.Source, m.Name())
				}
				if p.Source == model.Source("") {
					t.Errorf("pkg[%d]: Source is empty", i)
				}
			}
		})
	}
}
