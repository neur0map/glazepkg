package parsing

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

// Verbatim output of `dotnet tool list --global` (SDK 9.0.317), including a
// package id far wider than the header to exercise variable column widths.
func TestDotnetToolListParsing(t *testing.T) {
	output := `Package Id                            Version      Commands 
------------------------------------------------------------
dotnet-ef                             10.0.11      dotnet-ef
dotnetsay                             3.0.3        dotnetsay
microsoft.web.librarymanager.cli      3.0.114      libman   
`
	pkgs := manager.ParseDotnetToolList([]byte(output))
	if len(pkgs) != 3 {
		t.Fatalf("expected 3 tools, got %d: %+v", len(pkgs), pkgs)
	}

	if pkgs[0].Name != "dotnet-ef" || pkgs[0].Version != "10.0.11" {
		t.Errorf("pkg 0: got %q %q", pkgs[0].Name, pkgs[0].Version)
	}
	if pkgs[0].Source != model.SourceDotnetTool {
		t.Errorf("pkg 0 source = %q, want dotnet-tool", pkgs[0].Source)
	}
	if pkgs[0].Description != "dotnet-ef" {
		t.Errorf("pkg 0 description = %q, want the command name", pkgs[0].Description)
	}
	if pkgs[1].Name != "dotnetsay" || pkgs[1].Version != "3.0.3" {
		t.Errorf("pkg 1: got %q %q", pkgs[1].Name, pkgs[1].Version)
	}

	// The wide id must not bleed into the version column.
	if pkgs[2].Name != "microsoft.web.librarymanager.cli" || pkgs[2].Version != "3.0.114" {
		t.Errorf("pkg 2: got %q %q", pkgs[2].Name, pkgs[2].Version)
	}
	if pkgs[2].Description != "libman" {
		t.Errorf("pkg 2 description = %q, want libman", pkgs[2].Description)
	}
}

// Verbatim output when no global tools are installed: header and separator
// only, no data rows.
func TestDotnetToolListParsingEmpty(t *testing.T) {
	output := `Package Id      Version      Commands 
--------------------------------------
`
	pkgs := manager.ParseDotnetToolList([]byte(output))
	if len(pkgs) != 0 {
		t.Fatalf("expected 0 tools, got %d: %+v", len(pkgs), pkgs)
	}
}

// Verbatim output of `DOTNET_CLI_UI_LANGUAGE=de dotnet tool list --global`: the
// header is translated (DOTNET_CLI_UI_LANGUAGE overrides gpk's LC_ALL=C), so
// parsing must key off the dashed separator, not the header text.
func TestDotnetToolListParsingLocalizedHeader(t *testing.T) {
	output := `Paket-ID                              Version      Befehle  
------------------------------------------------------------
dotnet-ef                             10.0.11      dotnet-ef
dotnetsay                             3.0.3        dotnetsay
`
	pkgs := manager.ParseDotnetToolList([]byte(output))
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 tools, got %d: %+v", len(pkgs), pkgs)
	}
	if pkgs[0].Name != "dotnet-ef" || pkgs[1].Name != "dotnetsay" {
		t.Errorf("localized header leaked into data: %+v", pkgs)
	}
}
