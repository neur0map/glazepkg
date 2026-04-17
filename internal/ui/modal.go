package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/neur0map/glazepkg/internal/config"
)

// ModalType identifies which modal is currently open. ModalNone means no modal.
type ModalType int

const (
	// ModalNone indicates no modal is currently open.
	ModalNone ModalType = iota
	// ModalHelp shows the keybindings reference.
	ModalHelp
	// ModalExport is the format-picker (Text / JSON) for exporting the package list.
	ModalExport
	// ModalDeps shows the dependency list of the currently-selected package.
	ModalDeps
	// ModalPkgHelp shows the output of `<pkg> --help` for the current package.
	ModalPkgHelp
	// ModalTheme is the theme picker with live preview.
	ModalTheme
	// ModalConfirmUpgrade confirms an upgrade action, optionally capturing sudo password.
	ModalConfirmUpgrade
	// ModalConfirmRemove confirms a remove action, optionally with deps selector + sudo password.
	ModalConfirmRemove
	// ModalConfirmBatch confirms a batch operation on multi-selected packages.
	ModalConfirmBatch
)

// ModalFrameOpts is the structural input for every modal. Pure string I/O.
type ModalFrameOpts struct {
	Title  string
	Body   string
	Footer string
	Width  int
}

// ModalFrame produces a self-contained bordered rectangle. It owns all border
// drawing; do NOT wrap its output in StyleOverlay (or any other style that
// also draws a border — you would get doubling).
func ModalFrame(opts ModalFrameOpts) string {
	bodyLines := strings.Split(opts.Body, "\n")

	inner := opts.Width
	if inner == 0 {
		// auto-fit: inner width is max of body/title/footer visible widths.
		// Use lipgloss.Width throughout (rune-aware + strips ANSI) — never len().
		for _, line := range bodyLines {
			if w := lipgloss.Width(line); w > inner {
				inner = w
			}
		}
		if w := lipgloss.Width(opts.Title) + 2; w > inner {
			inner = w
		}
		if w := lipgloss.Width(opts.Footer); w > inner {
			inner = w
		}
	}
	inner += 2 // minimum 2-space padding on each side

	border := lipgloss.NewStyle().Foreground(ColorSurface)

	var b strings.Builder

	b.WriteString(border.Render("╭"))
	if opts.Title != "" {
		title := " " + opts.Title + " "
		titleW := lipgloss.Width(title)
		leftFill := (inner - titleW) / 2
		rightFill := inner - titleW - leftFill
		b.WriteString(border.Render(strings.Repeat("─", leftFill)))
		b.WriteString(border.Render(title))
		b.WriteString(border.Render(strings.Repeat("─", rightFill)))
	} else {
		b.WriteString(border.Render(strings.Repeat("─", inner)))
	}
	b.WriteString(border.Render("╮"))
	b.WriteString("\n")

	for _, line := range bodyLines {
		b.WriteString(border.Render("│"))
		b.WriteString(" ")
		b.WriteString(line)
		pad := inner - 1 - lipgloss.Width(line)
		if pad < 0 {
			pad = 0
		}
		b.WriteString(strings.Repeat(" ", pad))
		b.WriteString(border.Render("│"))
		b.WriteString("\n")
	}

	if opts.Footer != "" {
		b.WriteString(border.Render("├"))
		b.WriteString(border.Render(strings.Repeat("─", inner)))
		b.WriteString(border.Render("┤"))
		b.WriteString("\n")

		b.WriteString(border.Render("│"))
		b.WriteString(" ")
		b.WriteString(StyleDim.Render(opts.Footer))
		pad := inner - 1 - lipgloss.Width(opts.Footer)
		if pad < 0 {
			pad = 0
		}
		b.WriteString(strings.Repeat(" ", pad))
		b.WriteString(border.Render("│"))
		b.WriteString("\n")
	}

	b.WriteString(border.Render("╰"))
	b.WriteString(border.Render(strings.Repeat("─", inner)))
	b.WriteString(border.Render("╯"))

	return b.String()
}

// stripAnsi strips ANSI escape sequences from s. Small state machine that
// handles the CSI sequences lipgloss emits. Returns a plain-text string.
func stripAnsi(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) {
				c := s[i]
				i++
				if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
					break
				}
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// sliceVisible returns the substring of s between visible (post-ANSI-strip)
// rune positions [from, to). Rune-aware so Unicode doesn't get cut mid-rune.
func sliceVisible(s string, from, to int) string {
	if to <= from {
		return ""
	}
	runes := []rune(stripAnsi(s))
	if from >= len(runes) {
		return ""
	}
	if to > len(runes) {
		to = len(runes)
	}
	return string(runes[from:to])
}

// Overlay composites top over base: base is flat-dimmed, top is centered.
func Overlay(base, top string, width, height int) string {
	baseLines := strings.Split(base, "\n")
	for len(baseLines) < height {
		baseLines = append(baseLines, "")
	}
	baseLines = baseLines[:height]

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	for i, line := range baseLines {
		plain := stripAnsi(line)
		if w := lipgloss.Width(plain); w < width {
			plain += strings.Repeat(" ", width-w)
		}
		baseLines[i] = dim.Render(plain)
	}

	topLines := strings.Split(top, "\n")
	topH := len(topLines)
	topW := 0
	for _, line := range topLines {
		if w := lipgloss.Width(line); w > topW {
			topW = w
		}
	}

	startY := (height - topH) / 2
	startX := (width - topW) / 2
	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	for i, topLine := range topLines {
		row := startY + i
		if row >= len(baseLines) {
			break
		}
		// Clamp topLine to available width so oversized modals don't break the terminal.
		clippedTop := topLine
		if lipgloss.Width(topLine) > width {
			clippedTop = sliceVisible(topLine, 0, width)
		}
		endX := startX + lipgloss.Width(clippedTop)
		if endX > width {
			endX = width
		}
		left := sliceVisible(baseLines[row], 0, startX)
		right := sliceVisible(baseLines[row], endX, width)
		baseLines[row] = dim.Render(left) + clippedTop + dim.Render(right)
	}

	return strings.Join(baseLines, "\n")
}

// clipModalByAnim returns a vertically-clipped view of box, revealing
// int(anim * rowCount) rows centered on the middle row. Used by the
// scale-from-center spring animation.
//
// Clamps:
//   - anim <= 0: returns the single middle row (never empty — callers rely on
//     at least one row being present for their layout).
//   - anim >= 1: returns the full box unclipped (allows spring overshoot past
//     1.0 without growing beyond the natural modal height).
func clipModalByAnim(box string, anim float64) string {
	rows := strings.Split(box, "\n")
	n := len(rows)
	if anim <= 0 {
		return rows[n/2]
	}
	if anim >= 1 {
		return box
	}
	reveal := int(float64(n)*anim + 0.5)
	if reveal < 1 {
		reveal = 1
	}
	top := n/2 - reveal/2
	bottom := top + reveal
	if top < 0 {
		top = 0
	}
	if bottom > n {
		bottom = n
	}
	return strings.Join(rows[top:bottom], "\n")
}

// openModal starts the entrance animation for t.
func (m *Model) openModal(t ModalType) tea.Cmd {
	m.modal = t
	m.modalAnim = 0
	m.modalAnimVel = 0
	m.modalOpening = true
	m.modalSpring = newModalSpring()
	return modalAnimTick()
}

// closeModal starts the exit animation. Any per-modal cleanup that depends on
// close reason must have happened BEFORE this is called.
func (m *Model) closeModal() tea.Cmd {
	m.modalOpening = false
	return modalAnimTick()
}

// resetTransientModalState clears per-modal scrolls, cursors, focus indices,
// and the password buffer. Does NOT perform close-reason-dependent actions.
func (m *Model) resetTransientModalState() {
	m.depsCursor = 0
	m.pkgHelpScroll = 0
	m.exportCursor = 0
	m.confirmFocus = 0
	m.removeFocus = 0
	m.batchFocus = 0
	m.passwordInput.SetValue("")
	m.passwordInput.Blur()
}

// handleModalKey is the central key dispatcher for any open modal.
func (m *Model) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.modal {
	case ModalHelp:
		return handleHelpModalKey(m, msg)
	case ModalExport:
		return handleExportModalKey(m, msg)
	case ModalDeps:
		return handleDepsModalKey(m, msg)
	case ModalPkgHelp:
		return handlePkgHelpModalKey(m, msg)
	case ModalTheme:
		return handleThemeModalKey(m, msg)
	case ModalConfirmUpgrade:
		return handleUpgradeConfirmModalKey(m, msg)
	case ModalConfirmRemove:
		return handleRemoveConfirmModalKey(m, msg)
	case ModalConfirmBatch:
		return handleBatchConfirmModalKey(m, msg)
	}
	return m, nil
}

// renderModal composites the current modal over base.
func (m *Model) renderModal(base string) string {
	if m.modal == ModalNone {
		return base
	}
	var opts ModalFrameOpts
	switch m.modal {
	case ModalHelp:
		opts = renderHelpModalBody(m)
	case ModalExport:
		opts = renderExportModalBody(m)
	case ModalDeps:
		opts = renderDepsModalBody(m)
	case ModalPkgHelp:
		opts = renderPkgHelpModalBody(m)
	case ModalTheme:
		opts = renderThemeModalBody(m)
	case ModalConfirmUpgrade:
		opts = renderUpgradeConfirmModalBody(m)
	case ModalConfirmRemove:
		opts = renderRemoveConfirmModalBody(m)
	case ModalConfirmBatch:
		opts = renderBatchConfirmModalBody(m)
	}
	box := ModalFrame(opts)
	clipped := clipModalByAnim(box, m.modalAnim)
	return Overlay(base, clipped, m.width, m.height)
}

// --- Per-modal stubs (filled in during per-modal migration tasks). ---

func handleHelpModalKey(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// any key closes (matches current app.go:910-913 behavior)
	return m, m.closeModal()
}
func renderHelpModalBody(m *Model) ModalFrameOpts {
	return ModalFrameOpts{
		Title:  "HELP",
		Body:   helpBody(),
		Footer: "any key close",
	}
}

func handleExportModalKey(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := normalizeHotkey(msg.String())
	switch key {
	case "esc", "q":
		return m, m.closeModal()
	case "j", "down":
		if m.exportCursor < len(exportFormats)-1 {
			m.exportCursor++
		}
	case "k", "up":
		if m.exportCursor > 0 {
			m.exportCursor--
		}
	case "enter":
		return m, tea.Batch(m.closeModal(), doExport(m.allPkgs, m.exportCursor))
	}
	return m, nil
}
func renderExportModalBody(m *Model) ModalFrameOpts {
	return ModalFrameOpts{
		Title:  "EXPORT PACKAGES",
		Body:   exportBody(m),
		Footer: "↑↓ pick · enter export · esc cancel",
	}
}

func handleDepsModalKey(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := normalizeHotkey(msg.String())
	total := len(m.detailPkg.DependsOn) + len(m.detailPkg.RequiredBy)
	switch key {
	case "esc", "q", "d":
		return m, m.closeModal()
	case "j", "down":
		if m.depsCursor < total-1 {
			m.depsCursor++
		}
	case "k", "up":
		if m.depsCursor > 0 {
			m.depsCursor--
		}
	case "g", "home":
		m.depsCursor = 0
	case "G", "end":
		if total > 0 {
			m.depsCursor = total - 1
		}
	}
	return m, nil
}
func renderDepsModalBody(m *Model) ModalFrameOpts {
	title := "DEPENDENCIES"
	if m.detailPkg.Name != "" {
		title = "DEPENDENCIES — " + m.detailPkg.Name
	}
	return ModalFrameOpts{
		Title:  title,
		Body:   depsBody(m),
		Footer: "↑↓ navigate · esc close",
	}
}

func handlePkgHelpModalKey(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := normalizeHotkey(msg.String())
	maxScroll := len(m.pkgHelpLines) - (m.height - 8)
	if maxScroll < 0 {
		maxScroll = 0
	}
	switch key {
	case "esc", "q", "h":
		return m, m.closeModal()
	case "j", "down":
		if m.pkgHelpScroll < maxScroll {
			m.pkgHelpScroll++
		}
	case "k", "up":
		if m.pkgHelpScroll > 0 {
			m.pkgHelpScroll--
		}
	case "ctrl+d", "pgdown":
		m.pkgHelpScroll += m.height / 2
		if m.pkgHelpScroll > maxScroll {
			m.pkgHelpScroll = maxScroll
		}
	case "ctrl+u", "pgup":
		m.pkgHelpScroll -= m.height / 2
		if m.pkgHelpScroll < 0 {
			m.pkgHelpScroll = 0
		}
	case "g", "home":
		m.pkgHelpScroll = 0
	case "G", "end":
		m.pkgHelpScroll = maxScroll
	}
	return m, nil
}
func renderPkgHelpModalBody(m *Model) ModalFrameOpts {
	title := "PACKAGE HELP"
	if m.detailPkg.Name != "" {
		title = strings.ToUpper(m.detailPkg.Name) + " --HELP"
	}
	return ModalFrameOpts{
		Title:  title,
		Body:   pkgHelpBody(m),
		Footer: "↑↓ scroll · esc close",
	}
}

func handleThemeModalKey(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := normalizeHotkey(msg.String())
	switch key {
	case "esc", "q":
		// CRITICAL: revert BEFORE closeModal so the exit animation plays with
		// the reverted theme, not the previewed one.
		ApplyTheme(config.ResolveTheme(m.prevThemeID))
		m.refreshInputStyles()
		return m, m.closeModal()
	case "j", "down":
		if m.themeCursor < len(m.themeList)-1 {
			m.themeCursor++
			ApplyTheme(m.themeList[m.themeCursor])
			m.refreshInputStyles()
		}
	case "k", "up":
		if m.themeCursor > 0 {
			m.themeCursor--
			ApplyTheme(m.themeList[m.themeCursor])
			m.refreshInputStyles()
		}
	case "enter":
		selected := m.themeList[m.themeCursor]
		ApplyTheme(selected)
		m.refreshInputStyles()
		m.appConfig.Appearance.Theme = selected.ID
		_ = config.Save(m.appConfig)
		return m, m.closeModal()
	}
	return m, nil
}
func renderThemeModalBody(m *Model) ModalFrameOpts {
	return ModalFrameOpts{
		Title:  "THEME",
		Body:   themeBody(m),
		Footer: "↑↓ preview · enter apply · esc revert",
	}
}

func handleUpgradeConfirmModalKey(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := normalizeHotkey(msg.String())
	hasPwField := m.needsSudoPassword()

	// Password field focused: textinput absorbs most keys, but we own tab/enter/esc.
	if hasPwField && m.confirmFocus == 0 {
		switch key {
		case "esc":
			m.cancelUpgradeConfirm()
			return m, m.closeModal()
		case "tab":
			m.confirmFocus = 1
			m.passwordInput.Blur()
			return m, nil
		case "enter":
			if m.passwordInput.Value() != "" {
				m.confirmFocus = 1
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
		if m.confirmFocus == 1 { // Yes
			if hasPwField && m.passwordInput.Value() == "" {
				m.confirmFocus = 0
				m.passwordInput.Focus()
				return m, textinput.Blink
			}
			return m, tea.Batch(m.closeModal(), m.executePendingUpgrade())
		}
		// Yes not focused → treat as No/cancel
		m.cancelUpgradeConfirm()
		return m, m.closeModal()
	case "esc":
		m.cancelUpgradeConfirm()
		return m, m.closeModal()
	case "tab", "right", "l":
		if m.confirmFocus == 1 {
			m.confirmFocus = 2
		} else {
			m.confirmFocus = 1
		}
	case "shift+tab", "left", "h":
		if m.confirmFocus == 2 {
			m.confirmFocus = 1
		} else if hasPwField {
			m.confirmFocus = 0
			m.passwordInput.Focus()
			return m, textinput.Blink
		}
	}
	return m, nil
}
func renderUpgradeConfirmModalBody(m *Model) ModalFrameOpts {
	title := "CONFIRM UPGRADE"
	if m.pendingUpgrade != nil && m.pendingUpgrade.opLabel == "install" {
		title = "CONFIRM INSTALL"
	}
	return ModalFrameOpts{
		Title:  title,
		Body:   upgradeConfirmBody(m),
		Footer: "tab cycle · enter confirm · esc cancel",
	}
}

func handleRemoveConfirmModalKey(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := normalizeHotkey(msg.String())
	hasDeep := m.pendingRemove != nil && m.pendingRemove.deepCmd != nil
	hasPw := m.removeNeedsSudo()

	// Zone 0: Mode selector (only when hasDeep)
	if hasDeep && m.removeFocus == 0 {
		switch key {
		case "esc":
			m.cancelRemoveConfirm()
			return m, m.closeModal()
		case "j", "down":
			if m.removeMode == 0 {
				m.removeMode = 1
			}
		case "k", "up":
			if m.removeMode == 1 {
				m.removeMode = 0
			}
		case "tab", "enter":
			if hasPw {
				m.removeFocus = 1
				m.passwordInput.Focus()
				return m, textinput.Blink
			}
			m.removeFocus = 2
		}
		return m, nil
	}

	// Zone 1: Password field (textinput absorbs most keys)
	if hasPw && m.removeFocus == 1 {
		switch key {
		case "esc":
			m.cancelRemoveConfirm()
			return m, m.closeModal()
		case "tab":
			m.removeFocus = 2
			m.passwordInput.Blur()
			return m, nil
		case "enter":
			if m.passwordInput.Value() != "" {
				m.removeFocus = 2
				m.passwordInput.Blur()
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.passwordInput, cmd = m.passwordInput.Update(msg)
			return m, cmd
		}
	}

	// Zones 2/3: Yes / No buttons
	switch key {
	case "enter":
		if m.removeFocus == 2 { // Yes
			if hasPw && m.passwordInput.Value() == "" {
				m.removeFocus = 1
				m.passwordInput.Focus()
				return m, textinput.Blink
			}
			return m, tea.Batch(m.closeModal(), m.executeRemove())
		}
		// No focused
		m.cancelRemoveConfirm()
		return m, m.closeModal()
	case "esc":
		m.cancelRemoveConfirm()
		return m, m.closeModal()
	case "tab", "right", "l":
		if m.removeFocus == 2 {
			m.removeFocus = 3
		} else {
			m.removeFocus = 2
		}
	case "shift+tab", "left", "h":
		if m.removeFocus == 3 {
			m.removeFocus = 2
		} else if hasPw {
			m.removeFocus = 1
			m.passwordInput.Focus()
			return m, textinput.Blink
		} else if hasDeep {
			m.removeFocus = 0
		}
	}
	return m, nil
}
func renderRemoveConfirmModalBody(m *Model) ModalFrameOpts {
	return ModalFrameOpts{
		Title:  "CONFIRM REMOVE",
		Body:   removeConfirmBody(m),
		Footer: "tab cycle · enter confirm · esc cancel",
	}
}

func handleBatchConfirmModalKey(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := normalizeHotkey(msg.String())
	hasPw := m.batchNeedsSudo()

	// Password field focused
	if hasPw && m.batchFocus == 0 {
		switch key {
		case "esc":
			m.cancelBatchConfirm()
			return m, m.closeModal()
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

	// Yes/No buttons
	switch key {
	case "enter":
		if m.batchFocus == 1 { // Yes
			if hasPw && m.passwordInput.Value() == "" {
				m.batchFocus = 0
				m.passwordInput.Focus()
				return m, textinput.Blink
			}
			return m, tea.Batch(m.closeModal(), m.executeBatch())
		}
		m.cancelBatchConfirm()
		return m, m.closeModal()
	case "esc":
		m.cancelBatchConfirm()
		return m, m.closeModal()
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
func renderBatchConfirmModalBody(m *Model) ModalFrameOpts {
	title := "CONFIRM BATCH"
	if m.pendingBatch != nil {
		title = "CONFIRM BATCH " + strings.ToUpper(m.pendingBatch.op)
	}
	return ModalFrameOpts{
		Title:  title,
		Body:   batchConfirmBody(m),
		Footer: "tab cycle · enter confirm · esc cancel",
	}
}
