package ui

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
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
		return tea.Batch(m.openModal(ModalConfirmBatch), textinput.Blink)
	}
	m.batchFocus = 1 // Yes
	m.passwordInput.Blur()
	return m.openModal(ModalConfirmBatch)
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
	m.pendingBatch = nil
	m.passwordInput.SetValue("")
	m.passwordInput.Blur()
	m.statusMsg = "batch operation cancelled"
}

func (m *Model) executeBatch() tea.Cmd {
	if m.pendingBatch == nil {
		return nil
	}
	batch := *m.pendingBatch
	password := ""
	if m.batchNeedsSudo() {
		password = m.passwordInput.Value()
	}

	m.pendingBatch = nil
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

// batchConfirmBody renders the body of ModalConfirmBatch. The package list
// is rendered into a scrollable region so long lists stay readable: the
// modal grows to fit the list when the terminal has room, and clamps to a
// scrollable window (driven by m.batchScroll) when it doesn't. The header
// ("Upgrade N packages?") and the footer (password field + Yes/No buttons)
// are always pinned — only the list in between scrolls.
func batchConfirmBody(m *Model) string {
	batch := m.pendingBatch
	if batch == nil {
		return ""
	}

	wrapW := m.width - 10
	if wrapW > 80 {
		wrapW = 80
	}
	if wrapW < 40 {
		wrapW = 40
	}

	// --- Header (always visible) ---
	header := StyleNormal.Render(fmt.Sprintf("%s %d packages?", batchOpTitle(batch.op), len(batch.ops)))

	// --- Package list (scrollable region) ---
	var privPkgs, unprivPkgs []batchOp
	for _, o := range batch.ops {
		if o.privileged {
			privPkgs = append(privPkgs, o)
		} else {
			unprivPkgs = append(unprivPkgs, o)
		}
	}

	var listBuf strings.Builder
	if len(privPkgs) > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(ColorYellow)
		listBuf.WriteString(warnStyle.Render("privileged (1 password for all):"))
		listBuf.WriteString("\n")
		writeSortedSourceLists(&listBuf, privPkgs, wrapW)
	}
	if len(unprivPkgs) > 0 {
		if listBuf.Len() > 0 {
			listBuf.WriteString("\n")
		}
		listBuf.WriteString(StyleDim.Render("unprivileged:"))
		listBuf.WriteString("\n")
		writeSortedSourceLists(&listBuf, unprivPkgs, wrapW)
	}
	if len(batch.skipped) > 0 {
		if listBuf.Len() > 0 {
			listBuf.WriteString("\n")
		}
		listBuf.WriteString(StyleDim.Render(fmt.Sprintf("skipped (%d):", len(batch.skipped))))
		listBuf.WriteString("\n")
		listBuf.WriteString(StyleDim.Render(wrapCommaList(batch.skipped, "  ", wrapW)))
	}
	listStr := strings.TrimRight(listBuf.String(), "\n")
	listLines := []string{}
	if listStr != "" {
		listLines = strings.Split(listStr, "\n")
	}

	// --- Footer (always visible): password + Yes/No buttons ---
	var footerBuf strings.Builder
	if m.batchNeedsSudo() {
		warnStyle := lipgloss.NewStyle().Foreground(ColorYellow)
		footerBuf.WriteString(warnStyle.Render("requires elevated privileges"))
		footerBuf.WriteString("\n\n")
		footerBuf.WriteString(m.passwordInput.View())
		footerBuf.WriteString("\n")
	}
	footerBuf.WriteString("\n")

	yesStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	noStyle := lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
	switch m.batchFocus {
	case 1:
		yesStyle = yesStyle.Background(ColorGreen).Foreground(ColorBase)
		noStyle = noStyle.Foreground(ColorSubtext)
	case 2:
		yesStyle = yesStyle.Foreground(ColorSubtext)
		noStyle = noStyle.Background(ColorRed).Foreground(ColorBase)
	default:
		yesStyle = yesStyle.Foreground(ColorSubtext)
		noStyle = noStyle.Foreground(ColorSubtext)
	}
	footerBuf.WriteString("    " + yesStyle.Render("  Yes  ") + "   " + noStyle.Render("  No  "))
	footer := footerBuf.String()

	// --- Compose with scroll ---
	// Budget breakdown: ModalFrame adds 4 rows of chrome (top border +
	// footer separator + footer row + bottom border); composition adds 1
	// blank row between header and list. maxListH is the upper bound on
	// list region height (including the scroll hint line when shown) — so
	// the modal grows until the list has no more room, then scrolls.
	headerH := lipgloss.Height(header)
	footerH := lipgloss.Height(footer)
	const chromeH = 5 // modal chrome (4) + blank row between header & list (1)
	maxListH := m.height - headerH - footerH - chromeH
	if maxListH < 3 {
		maxListH = 3
	}

	visibleList, scrollHint := sliceScrollable(listLines, m.batchScroll, maxListH)

	var out strings.Builder
	out.WriteString(header)
	out.WriteString("\n")
	if len(visibleList) > 0 {
		out.WriteString("\n")
		out.WriteString(strings.Join(visibleList, "\n"))
	}
	if scrollHint != "" {
		out.WriteString("\n")
		out.WriteString(StyleDim.Render(scrollHint))
	}
	out.WriteString("\n")
	out.WriteString(footer)

	return out.String()
}

// sliceScrollable returns the visible window of lines at the given scroll
// offset and an optional scroll indicator. When all lines fit in maxH,
// scrollHint is empty and all lines are returned. Scroll is clamped so we
// never leave a blank gap at the bottom.
func sliceScrollable(lines []string, scroll, maxH int) (visible []string, scrollHint string) {
	if len(lines) <= maxH {
		return lines, ""
	}
	// Reserve one line for the scroll hint.
	windowH := maxH - 1
	if windowH < 1 {
		windowH = 1
	}
	maxScroll := len(lines) - windowH
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	visible = lines[scroll : scroll+windowH]
	above := scroll
	below := len(lines) - scroll - windowH
	switch {
	case above > 0 && below > 0:
		scrollHint = fmt.Sprintf("── %d above · %d below (↑/↓ scroll) ──", above, below)
	case above > 0:
		scrollHint = fmt.Sprintf("── %d above (↑ scroll) ──", above)
	case below > 0:
		scrollHint = fmt.Sprintf("── %d below (↓ scroll) ──", below)
	}
	return visible, scrollHint
}


// batchOpTitle returns the Title-Case label for a known batch op verb.
// Avoids strings.Title (deprecated in Go 1.18+) for a small closed set of
// values we control.
func batchOpTitle(op string) string {
	switch op {
	case "upgrade":
		return "Upgrade"
	case "remove":
		return "Remove"
	default:
		return op
	}
}

// writeSortedSourceLists groups ops by manager source, then writes them in
// a deterministic order: sources alphabetical, names alphabetical within
// each source. Needed because ranging over the intermediate map would
// otherwise shuffle the package list between renders.
func writeSortedSourceLists(buf *strings.Builder, ops []batchOp, wrapW int) {
	bySource := make(map[model.Source][]string)
	for _, o := range ops {
		bySource[o.pkg.Source] = append(bySource[o.pkg.Source], o.pkg.Name)
	}
	sources := make([]model.Source, 0, len(bySource))
	for src := range bySource {
		sources = append(sources, src)
	}
	sort.Slice(sources, func(i, j int) bool { return string(sources[i]) < string(sources[j]) })
	for _, src := range sources {
		names := bySource[src]
		sort.Strings(names)
		buf.WriteString(formatSourceNameList(string(src), names, wrapW))
	}
}

// formatSourceNameList renders "  <src>: a, b, c, d" with continuation lines
// indented to align under the first name so long package lists wrap cleanly
// inside the confirm modal rather than being truncated to 50 chars.
func formatSourceNameList(src string, names []string, wrapW int) string {
	prefix := fmt.Sprintf("  %s: ", src)
	indent := strings.Repeat(" ", len(prefix))

	avail := wrapW - len(prefix)
	if avail < 20 {
		avail = 20
	}

	lines := packCommaList(names, avail)
	if len(lines) == 0 {
		return prefix + "\n"
	}
	var b strings.Builder
	for i, ln := range lines {
		if i == 0 {
			b.WriteString(prefix)
		} else {
			b.WriteString(indent)
		}
		b.WriteString(ln)
		b.WriteString("\n")
	}
	return b.String()
}

// wrapCommaList wraps a comma-separated list with a constant leading indent
// on every line. Used for the "skipped" list where there's no per-source prefix.
func wrapCommaList(items []string, indent string, wrapW int) string {
	avail := wrapW - len(indent)
	if avail < 20 {
		avail = 20
	}
	lines := packCommaList(items, avail)
	var b strings.Builder
	for _, ln := range lines {
		b.WriteString(indent)
		b.WriteString(ln)
		b.WriteString("\n")
	}
	return b.String()
}

// packCommaList greedily packs comma-separated items into lines no wider than
// maxW visible chars. Non-final lines keep their trailing comma so the list
// reads naturally across a wrap; the final line has no trailing punctuation.
func packCommaList(items []string, maxW int) []string {
	if len(items) == 0 {
		return nil
	}
	var lines []string
	cur := ""
	for _, it := range items {
		candidate := it
		if cur != "" {
			candidate = cur + ", " + it
		}
		if len(candidate) > maxW && cur != "" {
			lines = append(lines, cur+",")
			cur = it
		} else {
			cur = candidate
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}
