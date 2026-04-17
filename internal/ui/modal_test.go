package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// init forces lipgloss to emit 256-color ANSI escapes even in non-TTY test
// environments. Without this, go test on CI may detect no color profile and
// emit plain text, which would break the dim-color assertion in TestOverlay_DimsBase.
func init() {
	lipgloss.SetColorProfile(termenv.ANSI256)
}

func TestStripAnsi_PlainPassthrough(t *testing.T) {
	got := stripAnsi("hello world")
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestStripAnsi_RemovesSGR(t *testing.T) {
	styled := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("red")
	if !strings.Contains(styled, "\x1b[") {
		t.Fatal("lipgloss did not emit ANSI; test cannot verify stripping")
	}
	if got := stripAnsi(styled); got != "red" {
		t.Errorf("got %q, want %q", got, "red")
	}
}

func TestModalFrame_TitleBodyFooter(t *testing.T) {
	out := ModalFrame(ModalFrameOpts{Title: "HI", Body: "x\ny", Footer: "esc"})
	lines := strings.Split(out, "\n")
	if len(lines) < 6 {
		t.Fatalf("expected >=6 rows, got %d: %q", len(lines), out)
	}
	if !strings.Contains(lines[0], "HI") {
		t.Errorf("title row missing title: %q", lines[0])
	}
	if !strings.Contains(out, "esc") {
		t.Error("footer row missing footer text")
	}
}

func TestModalFrame_NoTitle(t *testing.T) {
	out := ModalFrame(ModalFrameOpts{Body: "x", Footer: "esc"})
	lines := strings.Split(out, "\n")
	if strings.Contains(lines[0], "HI") {
		t.Error("top border should not contain title when opts.Title is empty")
	}
}

func TestModalFrame_NoFooter(t *testing.T) {
	out := ModalFrame(ModalFrameOpts{Title: "HI", Body: "x"})
	if strings.Contains(out, "├") {
		t.Error("separator row should not appear when opts.Footer is empty")
	}
}

func TestOverlay_DimsBase(t *testing.T) {
	base := strings.Repeat("abc\n", 4) + "abc"
	top := "XX\nXX"
	out := Overlay(base, top, 3, 5)
	// Dim is applied with color 244 — match that ANSI escape.
	if !strings.Contains(out, "\x1b[38;5;244m") {
		t.Error("Overlay did not apply dim foreground to base")
	}
}

func TestOverlay_CentersTop(t *testing.T) {
	base := strings.Repeat("..........\n", 9) + ".........."
	top := "MM\nMM" // 2x2
	out := Overlay(base, top, 10, 10)
	rows := strings.Split(out, "\n")
	if len(rows) != 10 {
		t.Fatalf("expected 10 rows, got %d", len(rows))
	}
	if !strings.Contains(rows[4], "MM") || !strings.Contains(rows[5], "MM") {
		t.Errorf("top not placed at rows 4-5; got row4=%q row5=%q", rows[4], rows[5])
	}
}

func TestOverlay_ClampsOversizedTop(t *testing.T) {
	base := strings.Repeat("..........\n", 4) + ".........." // 10x5
	oversized := strings.Repeat("M", 20)                     // 20 wide, base is 10
	out := Overlay(base, oversized, 10, 5)
	// Each output row should be at most `width` (10) cells of visible content.
	for i, row := range strings.Split(out, "\n") {
		if w := lipgloss.Width(row); w > 10 {
			t.Errorf("row %d width=%d exceeds width=10: %q", i, w, row)
		}
	}
}

func TestClipModalByAnim_Anim0(t *testing.T) {
	box := "a\nb\nc\nd\ne"
	out := clipModalByAnim(box, 0.0)
	rows := strings.Split(out, "\n")
	if len(rows) != 1 {
		t.Errorf("anim=0 should yield 1 row (minimum), got %d", len(rows))
	}
}

func TestClipModalByAnim_Anim1(t *testing.T) {
	box := "a\nb\nc\nd\ne"
	out := clipModalByAnim(box, 1.0)
	if out != box {
		t.Errorf("anim=1 should yield full box, got %q", out)
	}
}

func TestClipModalByAnim_GrowsFromCenter(t *testing.T) {
	box := "a\nb\nc\nd\ne" // middle = 'c'
	out := clipModalByAnim(box, 0.4) // reveal = 2 rows
	if !strings.Contains(out, "c") {
		t.Errorf("center row 'c' should always be in reveal; got %q", out)
	}
}
