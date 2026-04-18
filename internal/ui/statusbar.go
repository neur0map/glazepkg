package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

// renderHeader returns the centered title block for the list view: the
// animated "GlazePKG" wordmark on its own line, followed by a dim summary
// line (manager count + pending updates) underneath. Both are centered to
// the terminal width so they group visually with the tabs and panel below.
func (m Model) renderHeader() string {
	const titleText = "GlazePKG"
	reveal := m.titleReveal
	if reveal > len(titleText) {
		reveal = len(titleText)
	}
	visible := titleText[:reveal]
	hidden := strings.Repeat(" ", len(titleText)-reveal)

	caret := ""
	if reveal < len(titleText) {
		caret = lipgloss.NewStyle().Foreground(ColorBlue).Bold(true).Render("▏")
		// Steal one space from hidden so the total width stays constant
		// during the reveal (prevents the title from "jumping" as chars
		// land). Once reveal catches up, the caret disappears.
		if len(hidden) > 0 {
			hidden = hidden[1:]
		}
	}

	titleStyle := lipgloss.NewStyle().Foreground(ColorBlue).Bold(true)
	title := titleStyle.Render(visible) + caret + titleStyle.Render(hidden)
	if m.updateBanner != "" {
		title += "  " + StyleUpdateBanner.Render(m.updateBanner)
	}
	title = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, title)

	summary := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, m.renderSummary())

	return title + "\n" + summary
}

// renderSummary returns a dim one-liner shown under the centered title in
// the list view: distinct active managers and count of packages with
// updates available. (Total package count is already shown in the ALL tab,
// so we don't duplicate it here.)
func (m Model) renderSummary() string {
	managersSeen := map[model.Source]bool{}
	updates := 0
	for _, p := range m.allPkgs {
		managersSeen[p.Source] = true
		if p.LatestVersion != "" && p.LatestVersion != p.Version {
			updates++
		}
	}
	dim := lipgloss.NewStyle().Foreground(ColorSubtext)
	accent := lipgloss.NewStyle().Foreground(ColorText).Bold(true)
	dot := dim.Render(" · ")

	out := accent.Render(fmt.Sprintf("%d", len(managersSeen))) + dim.Render(" managers")
	if updates > 0 {
		up := lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
		out += dot + up.Render(fmt.Sprintf("↑ %d updates", updates))
	}
	return out
}

// statusKeys implements bubbles/help.KeyMap. Short returns a flat slice that
// the help component will auto-wrap to width; Full returns column groups
// shown when the user toggles `?`.
type statusKeys struct {
	short []key.Binding
	full  [][]key.Binding
}

func (s statusKeys) ShortHelp() []key.Binding  { return s.short }
func (s statusKeys) FullHelp() [][]key.Binding { return s.full }

// contextKeys returns the keymap appropriate for the current view+modal state.
func (m Model) contextKeys() statusKeys {
	switch m.view {
	case viewList:
		if m.multiSelect {
			return statusKeys{
				short: []key.Binding{
					Keys.Space, Keys.Upgrade, Keys.Remove,
					Keys.Filter, Keys.MultiSelect, Keys.Quit,
				},
				full: [][]key.Binding{
					{Keys.Space, Keys.Upgrade, Keys.Remove},
					{Keys.Filter, Keys.MultiSelect, Keys.Quit},
				},
			}
		}
		short := []key.Binding{
			Keys.Filter, Keys.Tab, Keys.SizeFilter,
			Keys.Enter, Keys.Rescan, Keys.Snapshot,
			Keys.MultiSelect, Keys.Install, Keys.Diff,
			Keys.Export, Keys.Theme, Keys.Help, Keys.Quit,
		}
		full := [][]key.Binding{
			{Keys.Up, Keys.Down, Keys.PageUp, Keys.PageDown, Keys.Home, Keys.End},
			{Keys.Filter, Keys.Tab, Keys.SizeFilter, Keys.Enter},
			{Keys.Rescan, Keys.Snapshot, Keys.MultiSelect, Keys.Install},
			{Keys.Diff, Keys.Export, Keys.Theme, Keys.Help, Keys.Quit},
		}
		return statusKeys{short: short, full: full}

	case viewDetail:
		if m.editingDesc {
			return statusKeys{short: []key.Binding{saveKey(), Keys.Back}}
		}
		if m.modal == ModalPkgHelp {
			return statusKeys{short: []key.Binding{scrollKey(), pageKey(), closeKey()}}
		}
		if m.modal == ModalDeps {
			return statusKeys{short: []key.Binding{navigateKey(), closeKey()}}
		}
		var short []key.Binding
		if mgr := manager.BySource(m.detailPkg.Source); mgr != nil {
			if _, ok := mgr.(manager.Upgrader); ok {
				short = append(short, Keys.Upgrade)
			}
			if _, ok := mgr.(manager.Remover); ok {
				short = append(short, Keys.Remove)
			}
		}
		short = append(short, editDescKey())
		if len(m.detailPkg.DependsOn) > 0 || len(m.detailPkg.RequiredBy) > 0 {
			short = append(short, Keys.Deps)
		}
		short = append(short, Keys.PkgHelp, Keys.Back, Keys.Quit)
		return statusKeys{short: short, full: [][]key.Binding{short}}

	case viewDiff:
		return statusKeys{short: []key.Binding{Keys.Back, Keys.Quit}}

	case viewSearch:
		if m.searchInput.Focused() {
			return statusKeys{short: []key.Binding{searchSubmitKey(), Keys.Back}}
		}
		short := []key.Binding{
			navigateKey(), expandKey(), Keys.Install, Keys.PreRelease, newSearchKey(), Keys.Back,
		}
		return statusKeys{short: short, full: [][]key.Binding{short}}
	}
	return statusKeys{}
}

// renderStatusBar returns the bottom status line. If a transient status
// message is set (operation in flight), it takes priority; otherwise the
// context keybinds are rendered via bubbles/help.
func (m Model) renderStatusBar() string {
	if m.statusMsg != "" {
		return StyleStatusBar.Render(m.statusMsg)
	}

	hp := m.help
	if m.width > 0 {
		hp.Width = m.width - 2
	}
	bar := hp.View(m.contextKeys())

	// Optional prefix badges (multi-select count, size filter, pre-release).
	var prefix string
	switch m.view {
	case viewList:
		if m.multiSelect {
			selectStyle := lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
			prefix = selectStyle.Render(fmt.Sprintf("[%d selected]", m.selectionCount())) + "  "
		} else if m.sizeFilter > 0 {
			filterStyle := lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
			prefix = filterStyle.Render("["+sizeFilters[m.sizeFilter].Label+"]") + "  "
		}
	case viewSearch:
		if m.showPreRelease {
			preStyle := lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
			prefix = preStyle.Render("[pre-release]") + "  "
		}
	}

	return " " + prefix + bar
}

// Contextual bindings built on the fly. These reuse the existing Key structs
// where possible but override the help text for the current context (e.g.
// "enter" means "save" when editing, "expand" in search results, "search"
// while typing, etc.).
func saveKey() key.Binding {
	return key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save"))
}
func searchSubmitKey() key.Binding {
	return key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "search"))
}
func expandKey() key.Binding {
	return key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "expand"))
}
func newSearchKey() key.Binding {
	return key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "new search"))
}
func navigateKey() key.Binding {
	return key.NewBinding(key.WithKeys("j", "k"), key.WithHelp("j/k", "navigate"))
}
func scrollKey() key.Binding {
	return key.NewBinding(key.WithKeys("j", "k"), key.WithHelp("j/k", "scroll"))
}
func pageKey() key.Binding {
	return key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("pgdn/pgup", "page"))
}
func closeKey() key.Binding {
	return key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close"))
}
func editDescKey() key.Binding {
	return key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit description"))
}
