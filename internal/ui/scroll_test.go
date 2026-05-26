package ui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/neur0map/glazepkg/internal/model"
)

// makePkgs returns n dummy packages for scroll tests.
func makePkgs(n int) []model.Package {
	pkgs := make([]model.Package, n)
	for i := range pkgs {
		pkgs[i] = model.Package{Name: fmt.Sprintf("pkg-%d", i)}
	}
	return pkgs
}

// scrollModel builds a Model with the scroll offsets that WindowSizeMsg would
// compute for the given tableHeight, mimicing production
func scrollModel(tableHeight, scroll, cursor, numPkgs int) *Model {
	m := &Model{
		filteredPkgs: makePkgs(numPkgs),
		tableHeight:  tableHeight,
		scroll:       scroll,
		cursor:       cursor,
	}
	tc := tableHeight / 2
	m.topScrollOff = tc - 4
	m.botScrollOff = tc + 5
	if tableHeight%2 == 0 {
		m.botScrollOff--
	}
	if m.topScrollOff < 0 {
		m.topScrollOff = 0
	}
	if m.botScrollOff > tableHeight {
		m.botScrollOff = tableHeight
	}
	return m
}

func runeKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// ---------------------------------------------------------------------------
// calculateScroll
// ---------------------------------------------------------------------------

func TestCalculateScroll_CursorInMiddleIsNoop(t *testing.T) {
	// cursor well inside [topScrollOff, botScrollOff) - scroll must not change.
	m := scrollModel(20, 0, 10, 30)
	if got := m.calculateScroll(); got != 0 {
		t.Errorf("scroll = %d, want 0", got)
	}
}

func TestCalculateScroll_CursorAtBotThresholdAdvancesScroll(t *testing.T) {
	// cursor-scroll == botScrollOff triggers a downward scroll.
	m := scrollModel(20, 0, 0, 30)
	m.cursor = m.scroll + m.botScrollOff
	got := m.calculateScroll()
	if got <= 0 {
		t.Errorf("expected scroll > 0 when cursor at botScrollOff, got %d", got)
	}
}

func TestCalculateScroll_CursorInsideTopDeadZoneRetreatsScroll(t *testing.T) {
	// cursor-scroll < topScrollOff triggers an upward scroll.
	m := scrollModel(20, 10, 0, 30)
	m.cursor = m.scroll + m.topScrollOff - 1 // one inside the dead zone
	initialScroll := m.scroll
	got := m.calculateScroll()
	if got >= initialScroll {
		t.Errorf("expected scroll to retreat from %d, got %d", initialScroll, got)
	}
}

func TestCalculateScroll_ResultNeverNegative(t *testing.T) {
	// cursor at row 0 with topScrollOff > 0 would produce negative scroll without the clamp.
	m := scrollModel(20, 0, 0, 30)
	if got := m.calculateScroll(); got < 0 {
		t.Errorf("scroll = %d, must not be negative", got)
	}
}

func TestCalculateScroll_ResultNeverExceedsMaxScroll(t *testing.T) {
	// cursor at last item - scroll must not exceed len-tableHeight.
	m := scrollModel(20, 0, 0, 30)
	m.cursor = len(m.filteredPkgs) - 1
	got := m.calculateScroll()
	maxScroll := len(m.filteredPkgs) - m.tableHeight
	if got > maxScroll {
		t.Errorf("scroll = %d, exceeds maxScroll %d", got, maxScroll)
	}
}

func TestCalculateScroll_EmptyListNoPanic(t *testing.T) {
	m := scrollModel(20, 0, 0, 0)
	got := m.calculateScroll()
	if got != 0 {
		t.Errorf("empty list: scroll = %d, want 0", got)
	}
}

func TestCalculateScroll_ShortListReturnsZero(t *testing.T) {
	// List shorter than viewport - maxScroll is negative, result must be 0.
	m := scrollModel(20, 0, 2, 5)
	got := m.calculateScroll()
	if got != 0 {
		t.Errorf("short list: scroll = %d, want 0", got)
	}
}

// ---------------------------------------------------------------------------
// j / k navigation
// ---------------------------------------------------------------------------

func TestJKey_CursorAdvances(t *testing.T) {
	m := scrollModel(20, 0, 0, 10)
	_, _ = m.handleKey(runeKey("j"))
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}
}

func TestJKey_CursorClampsAtLastItem(t *testing.T) {
	m := scrollModel(20, 0, 9, 10)
	_, _ = m.handleKey(runeKey("j"))
	if m.cursor != 9 {
		t.Errorf("cursor = %d, want 9 (last item)", m.cursor)
	}
}

func TestJKey_EmptyListNoPanic(t *testing.T) {
	m := scrollModel(20, 0, 0, 0)
	_, _ = m.handleKey(runeKey("j"))
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 on empty list", m.cursor)
	}
}

func TestJKey_ScrollAdvancesAfterCursorPassesBotScrollOff(t *testing.T) {
	m := scrollModel(20, 0, 0, 40)
	// Press j until cursor is one past botScrollOff - scroll must advance.
	for range m.botScrollOff + 1 {
		_, _ = m.handleKey(runeKey("j"))
	}
	if m.scroll == 0 {
		t.Errorf("scroll stayed 0 after cursor passed botScrollOff (%d)", m.botScrollOff)
	}
}

func TestKKey_CursorRetreats(t *testing.T) {
	m := scrollModel(20, 0, 5, 10)
	_, _ = m.handleKey(runeKey("k"))
	if m.cursor != 4 {
		t.Errorf("cursor = %d, want 4", m.cursor)
	}
}

func TestKKey_CursorClampsAtZero(t *testing.T) {
	m := scrollModel(20, 0, 0, 10)
	_, _ = m.handleKey(runeKey("k"))
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestKKey_EmptyListNoPanic(t *testing.T) {
	m := scrollModel(20, 0, 0, 0)
	_, _ = m.handleKey(runeKey("k"))
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0 on empty list", m.cursor)
	}
}

func TestKKey_ScrollRetreatsWhenCursorEntersTopDeadZone(t *testing.T) {
	// Place cursor exactly at the top-dead-zone boundary, then press k to
	// enter the zone - scroll must retreat.
	m := scrollModel(20, 10, 0, 40)
	m.cursor = m.scroll + m.topScrollOff // cursor-scroll == topScrollOff: at boundary
	initialScroll := m.scroll
	_, _ = m.handleKey(runeKey("k")) // cursor-scroll drops to topScrollOff-1: inside zone
	if m.scroll >= initialScroll {
		t.Errorf("scroll = %d, expected retreat from %d", m.scroll, initialScroll)
	}
}

// ---------------------------------------------------------------------------
// ctrl+d / ctrl+u - half-page cursor jump (scroll follows via calculateScroll)
// ---------------------------------------------------------------------------

func TestCtrlD_CursorAdvancesByHalfTableHeight(t *testing.T) {
	// Starting well below botScrollOff so calculateScroll doesn't fire yet.
	m := scrollModel(20, 0, 0, 40)
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlD})
	want := m.tableHeight / 2
	if m.cursor != want {
		t.Errorf("cursor = %d, want %d", m.cursor, want)
	}
}

func TestCtrlD_CursorClampsAtLastItem(t *testing.T) {
	m := scrollModel(20, 0, 35, 40)
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlD})
	if m.cursor != 39 {
		t.Errorf("cursor = %d, want 39 (last item)", m.cursor)
	}
}

func TestCtrlD_ScrollStaysInBounds(t *testing.T) {
	// Mash ctrl+d repeatedly - scroll must never exceed maxScroll.
	m := scrollModel(20, 0, 0, 40)
	for range 10 {
		_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlD})
	}
	maxScroll := len(m.filteredPkgs) - m.tableHeight
	if m.scroll < 0 || m.scroll > maxScroll {
		t.Errorf("scroll %d out of range [0, %d]", m.scroll, maxScroll)
	}
}

func TestCtrlD_EmptyListNoPanic(t *testing.T) {
	m := scrollModel(20, 0, 0, 0)
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlD})
	if m.scroll != 0 || m.cursor != 0 {
		t.Errorf("empty list: scroll=%d cursor=%d, want 0/0", m.scroll, m.cursor)
	}
}

func TestCtrlU_CursorRetreats(t *testing.T) {
	m := scrollModel(20, 0, 20, 40)
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	want := 20 - m.tableHeight/2
	if m.cursor != want {
		t.Errorf("cursor = %d, want %d", m.cursor, want)
	}
}

func TestCtrlU_CursorClampsAtZero(t *testing.T) {
	m := scrollModel(20, 0, 5, 40)
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestCtrlU_ScrollStaysInBounds(t *testing.T) {
	// Mash ctrl+u from a high scroll position - scroll must never go negative.
	m := scrollModel(20, 15, 25, 40)
	for range 10 {
		_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	}
	if m.scroll < 0 {
		t.Errorf("scroll = %d, must not be negative", m.scroll)
	}
}

func TestCtrlU_EmptyListNoPanic(t *testing.T) {
	m := scrollModel(20, 0, 0, 0)
	_, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	if m.scroll != 0 || m.cursor != 0 {
		t.Errorf("empty list: scroll=%d cursor=%d, want 0/0", m.scroll, m.cursor)
	}
}

// ---------------------------------------------------------------------------
// Window resize
// ---------------------------------------------------------------------------

func TestWindowResize_OffsetsDerivedFromPostFloorTableHeight(t *testing.T) {
	// height=20 → height-18=2, floored to 5. Offsets must be computed from 5,
	// not 2; otherwise botScrollOff gets clamped to 2 and the cursor soft-locks.
	m := Model{}
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m2 := result.(Model)
	if m2.tableHeight != 5 {
		t.Fatalf("tableHeight = %d, want 5", m2.tableHeight)
	}
	if m2.topScrollOff < 0 {
		t.Errorf("topScrollOff = %d, must not be negative", m2.topScrollOff)
	}
	if m2.botScrollOff > m2.tableHeight {
		t.Errorf("botScrollOff %d > tableHeight %d", m2.botScrollOff, m2.tableHeight)
	}
}

func TestWindowResize_ScrollReclampedWhenWindowGrows(t *testing.T) {
	// 15 packages, scroll=10. Resize so tableHeight=12 → maxScroll=3; scroll
	// must be pulled down so the last page is filled instead of left empty.
	pkgs := makePkgs(15)
	m := Model{filteredPkgs: pkgs, scroll: 10, cursor: 10, tableHeight: 5}
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30}) // height-18=12
	m2 := result.(Model)
	maxScroll := len(pkgs) - m2.tableHeight
	if m2.scroll > maxScroll {
		t.Errorf("scroll = %d, exceeds maxScroll %d after window grew", m2.scroll, maxScroll)
	}
}

func TestWindowResize_CursorPulledIntoViewport(t *testing.T) {
	// cursor beyond the new viewport bottom must be clamped in.
	pkgs := makePkgs(30)
	m := Model{filteredPkgs: pkgs, scroll: 0, cursor: 25, tableHeight: 30}
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 26}) // height-18=8
	m2 := result.(Model)
	if m2.cursor >= m2.scroll+m2.tableHeight {
		t.Errorf("cursor %d outside viewport [%d, %d)", m2.cursor, m2.scroll, m2.scroll+m2.tableHeight)
	}
}

func TestWindowResize_EmptyListNoPanic(t *testing.T) {
	m := Model{}
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m2 := result.(Model)
	if m2.scroll != 0 || m2.cursor != 0 {
		t.Errorf("empty list: scroll=%d cursor=%d, want 0/0", m2.scroll, m2.cursor)
	}
}
