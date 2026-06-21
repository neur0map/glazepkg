package snapshot

import (
	"testing"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestHistoryRoundTrip(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if got := LoadHistory(); len(got) != 0 {
		t.Fatalf("fresh history should be empty, got %d", len(got))
	}
	item := HistoryItem{Group: 1, Time: time.Now(), Op: OpInstall, Source: model.SourcePacman, Name: "ripgrep", Version: "14.0"}
	if err := AppendHistory(item); err != nil {
		t.Fatalf("append: %v", err)
	}
	got := LoadHistory()
	if len(got) != 1 || got[0].Name != "ripgrep" || got[0].Op != OpInstall {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestHistoryLimit(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	items := make([]HistoryItem, historyLimit+50)
	for i := range items {
		items[i] = HistoryItem{Group: int64(i), Op: OpInstall, Name: "p"}
	}
	if err := SaveHistory(items); err != nil {
		t.Fatalf("save: %v", err)
	}
	got := LoadHistory()
	if len(got) != historyLimit {
		t.Fatalf("history not trimmed: got %d, want %d", len(got), historyLimit)
	}
	// The newest entries must survive the trim.
	if got[len(got)-1].Group != int64(len(items)-1) {
		t.Errorf("trim dropped the newest entry: last group %d", got[len(got)-1].Group)
	}
}

func TestHolds(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	holds := []Hold{
		{Source: model.SourcePacman, Name: "linux"},
		{Source: "", Name: "firefox"},
	}
	if err := SaveHolds(holds); err != nil {
		t.Fatalf("save: %v", err)
	}
	got := LoadHolds()
	if len(got) != 2 {
		t.Fatalf("got %d holds, want 2", len(got))
	}
	if !IsHeld(got, model.SourcePacman, "linux") {
		t.Error("linux/pacman should be held")
	}
	if !IsHeld(got, model.SourceBrew, "firefox") {
		t.Error("firefox should be held under any source (wildcard)")
	}
	if IsHeld(got, model.SourcePacman, "vim") {
		t.Error("vim should not be held")
	}
	names := HeldNames(got, model.SourcePacman)
	if len(names) != 2 {
		t.Errorf("HeldNames(pacman) = %v, want linux and the wildcard firefox", names)
	}
}
