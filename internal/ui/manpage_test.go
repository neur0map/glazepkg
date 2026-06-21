package ui

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestStripManOverstrike(t *testing.T) {
	// "NAME" bold (each char overstruck with itself) and "foo" underlined (_\bX).
	in := "N\bNA\bAM\bME\bE _\bf_\bo_\bo"
	got := stripManOverstrike(in)
	want := "NAME foo"
	if got != want {
		t.Errorf("stripManOverstrike = %q, want %q", got, want)
	}
}

func TestStripManOverstrikePlain(t *testing.T) {
	if got := stripManOverstrike("plain text"); got != "plain text" {
		t.Errorf("got %q", got)
	}
}

func TestManPageMsgOpensModal(t *testing.T) {
	m := &Model{detailPkg: model.Package{Name: "ls"}}
	updated, _ := m.Update(manPageMsg{lines: []string{"LS(1)", "name ls"}})
	mm := updated.(Model)
	if !mm.pkgHelpIsMan {
		t.Error("expected pkgHelpIsMan true after manPageMsg")
	}
	if len(mm.pkgHelpLines) != 2 {
		t.Errorf("expected 2 help lines, got %d", len(mm.pkgHelpLines))
	}
}
