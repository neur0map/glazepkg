package ui

import (
	"os/exec"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"

	"github.com/neur0map/glazepkg/internal/model"
)

func TestQueuedOpLabel(t *testing.T) {
	up := queuedOp{upgrade: upgradeRequest{pkg: model.Package{Name: "curl"}, opLabel: "upgrade"}}
	if up.label() != "upgrade curl" {
		t.Errorf("upgrade label = %q", up.label())
	}
	rm := queuedOp{isRemove: true, remove: removeRequest{pkg: model.Package{Name: "vim"}}}
	if rm.label() != "remove vim" {
		t.Errorf("remove label = %q", rm.label())
	}
}

func TestExecutePendingUpgradeEnqueuesWhenBusy(t *testing.T) {
	m := &Model{upgradeInFlight: true}
	m.passwordInput = textinput.New()
	m.pendingUpgrade = &upgradeRequest{
		pkg:     model.Package{Name: "curl"},
		cmd:     exec.Command("echo", "x"),
		opLabel: "upgrade",
	}
	if cmd := m.executePendingUpgrade(); cmd != nil {
		t.Error("expected nil command when the op is queued")
	}
	if len(m.opQueue) != 1 {
		t.Fatalf("expected 1 queued op, got %d", len(m.opQueue))
	}
	if m.opQueue[0].upgrade.pkg.Name != "curl" {
		t.Errorf("queued wrong package: %q", m.opQueue[0].upgrade.pkg.Name)
	}
	if m.pendingUpgrade != nil {
		t.Error("pendingUpgrade should be cleared after queuing")
	}
}

func TestStartNextQueuedDrains(t *testing.T) {
	m := &Model{spinner: spinner.New()}
	m.opQueue = []queuedOp{
		{upgrade: upgradeRequest{pkg: model.Package{Name: "curl"}, cmd: exec.Command("echo", "x"), opLabel: "upgrade"}},
	}
	cmd := m.startNextQueued()
	if cmd == nil {
		t.Fatal("expected a command from startNextQueued")
	}
	if !m.upgradeInFlight {
		t.Error("expected upgradeInFlight to be set")
	}
	if len(m.opQueue) != 0 {
		t.Errorf("expected queue drained, got %d", len(m.opQueue))
	}
	if m.startNextQueued() != nil {
		t.Error("expected nil from an empty queue")
	}
}

func TestCancelQueuedOp(t *testing.T) {
	m := &Model{
		opQueue: []queuedOp{
			{upgrade: upgradeRequest{pkg: model.Package{Name: "a"}}},
			{upgrade: upgradeRequest{pkg: model.Package{Name: "b"}}},
			{upgrade: upgradeRequest{pkg: model.Package{Name: "c"}}},
		},
		queueCursor: 2,
	}
	m.cancelQueuedOp(2)
	if len(m.opQueue) != 2 {
		t.Fatalf("expected 2 ops left, got %d", len(m.opQueue))
	}
	if m.queueCursor != 1 {
		t.Errorf("cursor should clamp to 1, got %d", m.queueCursor)
	}
	if m.opQueue[1].upgrade.pkg.Name != "b" {
		t.Errorf("unexpected remaining op: %q", m.opQueue[1].upgrade.pkg.Name)
	}
	m.cancelQueuedOp(9) // out of range: no-op
	if len(m.opQueue) != 2 {
		t.Error("out-of-range cancel should be a no-op")
	}
}

func TestOpenQueueEmpty(t *testing.T) {
	m := &Model{}
	if cmd := m.openQueue(); cmd != nil {
		t.Error("empty queue should not open a modal")
	}
	if m.modal == ModalQueue {
		t.Error("modal should not open for an empty queue")
	}
	if m.statusMsg == "" {
		t.Error("expected a status hint for an empty queue")
	}
}

func TestOpenQueueNonEmpty(t *testing.T) {
	m := &Model{opQueue: []queuedOp{{upgrade: upgradeRequest{pkg: model.Package{Name: "a"}}}}}
	if cmd := m.openQueue(); cmd == nil {
		t.Fatal("expected a command opening the queue modal")
	}
	if m.modal != ModalQueue {
		t.Errorf("expected ModalQueue, got %v", m.modal)
	}
}
