package ui

import (
	"github.com/charmbracelet/bubbles/key"

	"github.com/neur0map/glazepkg/internal/manager"
)

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
