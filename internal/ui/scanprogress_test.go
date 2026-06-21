package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/progress"

	"github.com/neur0map/glazepkg/internal/model"
)

// A mid-scan manager completion must kick off the progress spring (a non-nil
// command) and FrameMsg must advance it without panicking, so the bar glides
// at 60FPS instead of jumping.
func TestScanProgressAnimates(t *testing.T) {
	m := Model{progress: newScanProgress(), scanning: true, scanTotal: 3}

	upd, cmd := m.Update(scanManagerDoneMsg{source: model.SourcePacman})
	m2 := upd.(Model)
	if m2.scanCompleted != 1 {
		t.Fatalf("scanCompleted = %d, want 1", m2.scanCompleted)
	}
	if cmd == nil {
		t.Error("incomplete scan should return a progress animation command")
	}

	upd2, _ := m2.Update(progress.FrameMsg{})
	if _, ok := upd2.(Model); !ok {
		// Update returns (tea.Model, tea.Cmd); the first value must stay a Model.
		t.Error("FrameMsg should be handled and return a Model")
	}
}
