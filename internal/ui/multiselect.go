package ui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

type batchResultMsg struct {
	succeeded []string
	failed    map[string]string // name → error
	op        string            // "upgrade" or "remove"
}

type batchProgressMsg struct {
	name   string
	status string // "running", "done", "failed"
	err    string
}

type batchNotifClearMsg struct{}

func (m *Model) toggleMultiSelect() {
	m.multiSelect = !m.multiSelect
	if !m.multiSelect {
		m.selections = nil
	} else if m.selections == nil {
		m.selections = make(map[string]bool)
	}
}

func (m *Model) toggleSelection() {
	pkg, ok := m.selectedPackage()
	if !ok {
		return
	}
	key := pkg.Key()
	if m.selections[key] {
		delete(m.selections, key)
	} else {
		m.selections[key] = true
	}
}

func (m *Model) selectionCount() int {
	return len(m.selections)
}

func (m *Model) selectedPkgs() []model.Package {
	var pkgs []model.Package
	for _, p := range m.allPkgs {
		if m.selections[p.Key()] {
			pkgs = append(pkgs, p)
		}
	}
	return pkgs
}

type batchOp struct {
	pkg        model.Package
	cmd        *exec.Cmd
	privileged bool
}

func (m *Model) batchUpgradeSelected() tea.Cmd {
	if m.upgradeInFlight || m.removeInFlight {
		m.statusMsg = "operation already in progress"
		return nil
	}

	selected := m.selectedPkgs()
	if len(selected) == 0 {
		m.statusMsg = "no packages selected"
		return nil
	}

	var ops []batchOp
	var skipped []string
	for _, pkg := range selected {
		mgr := manager.BySource(pkg.Source)
		if mgr == nil {
			continue
		}
		upgrader, ok := mgr.(manager.Upgrader)
		if !ok {
			skipped = append(skipped, pkg.Name)
			continue
		}
		cmd := upgrader.UpgradeCmd(pkg.Name)
		ops = append(ops, batchOp{
			pkg:        pkg,
			cmd:        cmd,
			privileged: isPrivilegedSource(pkg.Source) && len(cmd.Args) > 0 && cmd.Args[0] == "sudo",
		})
	}

	if len(ops) == 0 {
		m.statusMsg = "none of the selected packages support upgrade"
		return nil
	}

	return m.showBatchConfirm(ops, "upgrade", skipped)
}

func (m *Model) batchRemoveSelected() tea.Cmd {
	if m.upgradeInFlight || m.removeInFlight {
		m.statusMsg = "operation already in progress"
		return nil
	}

	selected := m.selectedPkgs()
	if len(selected) == 0 {
		m.statusMsg = "no packages selected"
		return nil
	}

	var ops []batchOp
	var skipped []string
	for _, pkg := range selected {
		mgr := manager.BySource(pkg.Source)
		if mgr == nil {
			continue
		}
		remover, ok := mgr.(manager.Remover)
		if !ok {
			skipped = append(skipped, pkg.Name)
			continue
		}
		cmd := remover.RemoveCmd(pkg.Name)
		ops = append(ops, batchOp{
			pkg:        pkg,
			cmd:        cmd,
			privileged: isPrivilegedSource(pkg.Source) && len(cmd.Args) > 0 && cmd.Args[0] == "sudo",
		})
	}

	if len(ops) == 0 {
		m.statusMsg = "none of the selected packages support remove"
		return nil
	}

	return m.showBatchConfirm(ops, "remove", skipped)
}

// Batch confirmation state is stored on the Model
type batchConfirmState struct {
	ops     []batchOp
	op      string // "upgrade" or "remove"
	skipped []string
}

func (m *Model) showBatchConfirm(ops []batchOp, op string, skipped []string) tea.Cmd {
	m.pendingBatch = &batchConfirmState{ops: ops, op: op, skipped: skipped}
	m.confirmingBatch = true

	// Check if any ops need sudo
	needsSudo := false
	for _, o := range ops {
		if o.privileged {
			needsSudo = true
			break
		}
	}

	m.passwordInput.SetValue("")
	if needsSudo {
		m.batchFocus = 0 // password
		m.passwordInput.Focus()
		return textinput.Blink
	}
	m.batchFocus = 1 // Yes
	m.passwordInput.Blur()
	return nil
}

func (m *Model) handleBatchConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	hasPw := m.batchNeedsSudo()

	// Password field focused
	if hasPw && m.batchFocus == 0 {
		switch key {
		case "esc":
			m.cancelBatchConfirm()
			return m, nil
		case "tab":
			m.batchFocus = 1
			m.passwordInput.Blur()
			return m, nil
		case "enter":
			if m.passwordInput.Value() != "" {
				m.batchFocus = 1
				m.passwordInput.Blur()
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.passwordInput, cmd = m.passwordInput.Update(msg)
			return m, cmd
		}
	}

	switch key {
	case "enter":
		if m.batchFocus == 1 { // Yes
			if hasPw && m.passwordInput.Value() == "" {
				m.batchFocus = 0
				m.passwordInput.Focus()
				return m, textinput.Blink
			}
			return m, m.executeBatch()
		}
		m.cancelBatchConfirm()
	case "esc":
		m.cancelBatchConfirm()
	case "tab", "right", "l":
		if m.batchFocus == 1 {
			m.batchFocus = 2
		} else {
			m.batchFocus = 1
		}
	case "shift+tab", "left", "h":
		if m.batchFocus == 2 {
			m.batchFocus = 1
		} else if hasPw {
			m.batchFocus = 0
			m.passwordInput.Focus()
			return m, textinput.Blink
		}
	}
	return m, nil
}

func (m *Model) batchNeedsSudo() bool {
	if m.pendingBatch == nil {
		return false
	}
	for _, o := range m.pendingBatch.ops {
		if o.privileged {
			return true
		}
	}
	return false
}

func (m *Model) cancelBatchConfirm() {
	m.confirmingBatch = false
	m.pendingBatch = nil
	m.passwordInput.SetValue("")
	m.passwordInput.Blur()
	m.statusMsg = "batch operation cancelled"
}

func (m *Model) executeBatch() tea.Cmd {
	if m.pendingBatch == nil {
		m.confirmingBatch = false
		return nil
	}
	batch := *m.pendingBatch
	password := ""
	if m.batchNeedsSudo() {
		password = m.passwordInput.Value()
	}

	m.pendingBatch = nil
	m.confirmingBatch = false
	m.passwordInput.SetValue("")
	m.passwordInput.Blur()
	m.upgradeInFlight = true
	m.batchLog = nil
	m.upgradeNotifErr = false

	// Order: privileged first, then unprivileged
	var ordered []batchOp
	for _, o := range batch.ops {
		if o.privileged {
			ordered = append(ordered, o)
		}
	}
	for _, o := range batch.ops {
		if !o.privileged {
			ordered = append(ordered, o)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.upgradeCancel = cancel
	m.batchOps = ordered
	m.batchPassword = password
	m.batchOpLabel = batch.op
	m.batchCtx = ctx
	m.batchCurrentPkg = ordered[0].pkg.Name
	m.upgradingPkgName = ordered[0].pkg.Name
	m.upgradeNotifMsg = fmt.Sprintf("%s %s (1/%d)...", gerund(batch.op), ordered[0].pkg.Name, len(ordered))

	return tea.Batch(m.spinner.Tick, runBatchOp(ctx, ordered, 0, password, batch.op))
}

func runBatchOp(ctx context.Context, ops []batchOp, idx int, password, op string) tea.Cmd {
	return func() tea.Msg {
		o := ops[idx]
		ctxCmd := exec.CommandContext(ctx, o.cmd.Args[0], o.cmd.Args[1:]...)
		if o.privileged && password != "" {
			ctxCmd.Stdin = strings.NewReader(password + "\n")
		}
		out, err := ctxCmd.CombinedOutput()
		errStr := ""
		status := "done"
		if err != nil {
			status = "failed"
			errStr = extractErrorLines(string(out))
			if errStr == "" {
				errStr = err.Error()
			}
		}
		return batchProgressMsg{
			name:   o.pkg.Name,
			status: status,
			err:    errStr,
		}
	}
}

func (m *Model) handleBatchProgress(msg batchProgressMsg) (tea.Model, tea.Cmd) {
	m.batchLog = append(m.batchLog, msg)
	completed := len(m.batchLog)
	total := len(m.batchOps)

	// Always continue to next op regardless of failure
	if completed < total {
		next := m.batchOps[completed]
		m.batchCurrentPkg = next.pkg.Name
		m.upgradingPkgName = next.pkg.Name
		m.upgradeNotifMsg = fmt.Sprintf("%s %s (%d/%d)...", gerund(m.batchOpLabel), next.pkg.Name, completed+1, total)
		return m, runBatchOp(m.batchCtx, m.batchOps, completed, m.batchPassword, m.batchOpLabel)
	}

	// All done
	m.upgradeInFlight = false
	m.upgradeCancel = nil
	m.batchCurrentPkg = ""
	m.upgradingPkgName = ""
	m.batchPassword = ""

	var succeeded []string
	failed := make(map[string]string)
	for _, entry := range m.batchLog {
		if entry.status == "done" {
			succeeded = append(succeeded, entry.name)
		} else {
			failed[entry.name] = entry.err
		}
	}

	m.multiSelect = false
	m.selections = nil

	summary := fmt.Sprintf("%d %s", len(succeeded), pastTense(m.batchOpLabel))
	if len(failed) > 0 {
		summary += fmt.Sprintf(", %d failed", len(failed))
		m.upgradeNotifErr = true
	} else {
		m.upgradeNotifErr = false
	}
	m.upgradeNotifMsg = summary
	m.batchOpLabel = ""
	m.batchOps = nil

	// Rescan affected sources
	rescanned := make(map[model.Source]bool)
	var cmds []tea.Cmd
	for _, name := range succeeded {
		for _, p := range m.allPkgs {
			if p.Name == name && !rescanned[p.Source] {
				rescanned[p.Source] = true
				cmds = append(cmds, m.rescanManager(p.Source))
			}
		}
	}
	dismissTime := 10 * time.Second
	if len(failed) > 0 {
		dismissTime = 30 * time.Second
	}
	cmds = append(cmds, tea.Tick(dismissTime, func(time.Time) tea.Msg {
		return upgradeNotifClearMsg{}
	}))
	return m, tea.Batch(cmds...)
}

func (m Model) renderBatchConfirmOverlay() string {
	batch := m.pendingBatch
	if batch == nil {
		return ""
	}

	var b strings.Builder
	title := "Batch " + strings.Title(batch.op)
	b.WriteString(StyleOverlayTitle.Render("  " + title))
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  " + strings.Repeat("─", 44)))
	b.WriteString("\n\n")
	b.WriteString(StyleNormal.Render(fmt.Sprintf("  %s %d packages?", strings.Title(batch.op), len(batch.ops))))
	b.WriteString("\n")

	overlayHeight := 11

	// Group by privilege
	var privPkgs, unprivPkgs []batchOp
	for _, o := range batch.ops {
		if o.privileged {
			privPkgs = append(privPkgs, o)
		} else {
			unprivPkgs = append(unprivPkgs, o)
		}
	}

	if len(privPkgs) > 0 {
		b.WriteString("\n")
		warnStyle := lipgloss.NewStyle().Foreground(ColorYellow)
		b.WriteString("  " + warnStyle.Render("privileged (1 password for all):"))
		b.WriteString("\n")
		// Group by source
		bySource := make(map[model.Source][]string)
		for _, o := range privPkgs {
			bySource[o.pkg.Source] = append(bySource[o.pkg.Source], o.pkg.Name)
		}
		for src, names := range bySource {
			nameList := strings.Join(names, ", ")
			if len(nameList) > 50 {
				nameList = nameList[:50] + "..."
			}
			b.WriteString(fmt.Sprintf("    %s: %s\n", src, nameList))
			overlayHeight++
		}
		overlayHeight += 2
	}

	if len(unprivPkgs) > 0 {
		b.WriteString("\n")
		b.WriteString("  " + StyleDim.Render("unprivileged:"))
		b.WriteString("\n")
		bySource := make(map[model.Source][]string)
		for _, o := range unprivPkgs {
			bySource[o.pkg.Source] = append(bySource[o.pkg.Source], o.pkg.Name)
		}
		for src, names := range bySource {
			nameList := strings.Join(names, ", ")
			if len(nameList) > 50 {
				nameList = nameList[:50] + "..."
			}
			b.WriteString(fmt.Sprintf("    %s: %s\n", src, nameList))
			overlayHeight++
		}
		overlayHeight += 2
	}

	if len(batch.skipped) > 0 {
		b.WriteString("\n")
		b.WriteString("  " + StyleDim.Render(fmt.Sprintf("skipped (%d): %s", len(batch.skipped), strings.Join(batch.skipped, ", "))))
		b.WriteString("\n")
		overlayHeight += 2
	}

	// Password
	if m.batchNeedsSudo() {
		warnStyle := lipgloss.NewStyle().Foreground(ColorYellow)
		b.WriteString("\n  " + warnStyle.Render("requires elevated privileges"))
		b.WriteString("\n\n")
		b.WriteString(m.passwordInput.View())
		b.WriteString("\n")
		overlayHeight += 4
	}

	b.WriteString("\n")

	yesStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	noStyle := lipgloss.NewStyle().Foreground(ColorRed).Bold(true)

	if m.batchFocus == 1 {
		yesStyle = yesStyle.Background(ColorGreen).Foreground(ColorBase)
		noStyle = noStyle.Foreground(ColorSubtext)
	} else if m.batchFocus == 2 {
		yesStyle = yesStyle.Foreground(ColorSubtext)
		noStyle = noStyle.Background(ColorRed).Foreground(ColorBase)
	} else {
		yesStyle = yesStyle.Foreground(ColorSubtext)
		noStyle = noStyle.Foreground(ColorSubtext)
	}

	b.WriteString("      " + yesStyle.Render("  Yes  ") + "   " + noStyle.Render("  No  "))

	content := b.String()
	overlayWidth := 52
	if overlayWidth > m.width-4 {
		overlayWidth = m.width - 4
	}

	overlay := StyleOverlay.
		Width(overlayWidth).
		Height(overlayHeight).
		Render(content)

	return placeOverlay(m.width, m.height, overlay)
}
