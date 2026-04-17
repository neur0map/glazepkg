package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
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
	if msg.String() == "esc" {
		return m, m.closeModal()
	}
	return m, nil
}
func renderExportModalBody(m *Model) ModalFrameOpts {
	return ModalFrameOpts{Title: "EXPORT", Body: "<pending migration>", Footer: "esc cancel"}
}

func handleDepsModalKey(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		return m, m.closeModal()
	}
	return m, nil
}
func renderDepsModalBody(m *Model) ModalFrameOpts {
	return ModalFrameOpts{Title: "DEPENDENCIES", Body: "<pending migration>", Footer: "esc close"}
}

func handlePkgHelpModalKey(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		return m, m.closeModal()
	}
	return m, nil
}
func renderPkgHelpModalBody(m *Model) ModalFrameOpts {
	return ModalFrameOpts{Title: "PACKAGE HELP", Body: "<pending migration>", Footer: "esc close"}
}

func handleThemeModalKey(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		return m, m.closeModal()
	}
	return m, nil
}
func renderThemeModalBody(m *Model) ModalFrameOpts {
	return ModalFrameOpts{Title: "THEME", Body: "<pending migration>", Footer: "esc cancel"}
}

func handleUpgradeConfirmModalKey(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		return m, m.closeModal()
	}
	return m, nil
}
func renderUpgradeConfirmModalBody(m *Model) ModalFrameOpts {
	return ModalFrameOpts{Title: "CONFIRM UPGRADE", Body: "<pending migration>", Footer: "esc cancel"}
}

func handleRemoveConfirmModalKey(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		return m, m.closeModal()
	}
	return m, nil
}
func renderRemoveConfirmModalBody(m *Model) ModalFrameOpts {
	return ModalFrameOpts{Title: "CONFIRM REMOVE", Body: "<pending migration>", Footer: "esc cancel"}
}

func handleBatchConfirmModalKey(m *Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		return m, m.closeModal()
	}
	return m, nil
}
func renderBatchConfirmModalBody(m *Model) ModalFrameOpts {
	return ModalFrameOpts{Title: "CONFIRM BATCH", Body: "<pending migration>", Footer: "esc cancel"}
}
