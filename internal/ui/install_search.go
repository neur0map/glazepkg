package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

func (m *Model) enterSearchView() tea.Cmd {
	m.view = viewSearch
	m.searchInput.SetValue("")
	m.searchInput.Focus()
	m.searchResults = nil
	m.searchCursor = 0
	m.searchActive = false
	m.searchPending = 0
	return textinput.Blink
}

func (m *Model) executeSearch() tea.Cmd {
	query := strings.TrimSpace(m.searchInput.Value())
	if query == "" {
		return nil
	}

	m.searchActive = true
	m.searchResults = nil
	m.searchCursor = 0
	m.searchInput.Blur()
	m.searchPending = 0

	mgrs := manager.All()
	var cmds []tea.Cmd
	for _, mgr := range mgrs {
		searcher, ok := mgr.(manager.Searcher)
		if !ok || !mgr.Available() {
			continue
		}
		m.searchPending++
		s := searcher
		source := mgr.Name()
		cmds = append(cmds, func() tea.Msg {
			pkgs, err := s.Search(query)
			return searchResultMsg{source: source, pkgs: pkgs, err: err}
		})
	}

	if len(cmds) == 0 {
		m.searchActive = false
		m.statusMsg = "no managers support search"
		return nil
	}

	cmds = append(cmds, m.spinner.Tick)
	return tea.Batch(cmds...)
}

func (m *Model) handleSearchResult(msg searchResultMsg) {
	m.searchPending--
	if msg.err == nil && len(msg.pkgs) > 0 {
		m.mergeSearchResults(msg.pkgs)
	}
	if m.searchPending <= 0 {
		m.searchActive = false
		m.searchPending = 0
	}
}

func (m *Model) mergeSearchResults(pkgs []model.Package) {
	groupIdx := make(map[string]int)
	for i, g := range m.searchResults {
		groupIdx[g.name] = i
	}

	for _, p := range pkgs {
		if idx, ok := groupIdx[p.Name]; ok {
			m.searchResults[idx].entries = append(m.searchResults[idx].entries, p)
		} else {
			groupIdx[p.Name] = len(m.searchResults)
			m.searchResults = append(m.searchResults, searchResultGroup{
				name:    p.Name,
				entries: []model.Package{p},
			})
		}
	}

	sort.Slice(m.searchResults, func(i, j int) bool {
		return m.searchResults[i].name < m.searchResults[j].name
	})

	for i := range m.searchResults {
		entries := m.searchResults[i].entries
		sort.Slice(entries, func(a, b int) bool {
			return compareVersions(entries[a].Version, entries[b].Version) > 0
		})
	}

	// Reorder groups by tier relevance to the current query. Runs after the
	// per-group version sort so the description tier classifies against the
	// highest-version entry.
	m.searchResults = rankGroupsByName(m.searchResults, m.searchInput.Value())
}

func (m *Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := normalizeHotkey(msg.String())

	if m.searchInput.Focused() {
		switch key {
		case "esc":
			if m.searchInput.Value() != "" {
				m.searchInput.SetValue("")
				m.searchResults = nil
				return m, nil
			}
			m.view = viewList
			m.searchInput.Blur()
			return m, nil
		case "enter":
			return m, m.executeSearch()
		default:
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd
		}
	}

	rows := m.searchRowCount()
	switch key {
	case "esc", "/":
		m.searchInput.Focus()
		return m, textinput.Blink
	case "q":
		m.view = viewList
		m.searchInput.Blur()
		return m, nil
	case "j", "down":
		if m.searchCursor < rows-1 {
			m.searchCursor++
		}
	case "k", "up":
		if m.searchCursor > 0 {
			m.searchCursor--
		}
	case "g", "home":
		m.searchCursor = 0
	case "G", "end":
		if rows > 0 {
			m.searchCursor = rows - 1
		}
	case "enter", "right", "l":
		m.toggleOrSelectSearch()
	case "left", "h":
		m.collapseSearchGroup()
	case "p":
		m.showPreRelease = !m.showPreRelease
	case "i":
		return m, m.installFromSearch()
	}
	return m, nil
}

func (m *Model) searchRowCount() int {
	count := 0
	for _, g := range m.searchResults {
		count++
		if g.expanded {
			count += len(g.entries)
		}
	}
	return count
}

func (m *Model) searchRowAt(row int) (groupIdx, entryIdx int) {
	pos := 0
	for gi, g := range m.searchResults {
		if pos == row {
			return gi, -1
		}
		pos++
		if g.expanded {
			for ei := range g.entries {
				if pos == row {
					return gi, ei
				}
				pos++
			}
		}
	}
	return -1, -1
}

func (m *Model) toggleOrSelectSearch() {
	gi, ei := m.searchRowAt(m.searchCursor)
	if gi < 0 {
		return
	}
	if ei == -1 {
		m.searchResults[gi].expanded = !m.searchResults[gi].expanded
	}
}

func (m *Model) collapseSearchGroup() {
	gi, _ := m.searchRowAt(m.searchCursor)
	if gi >= 0 && m.searchResults[gi].expanded {
		m.searchResults[gi].expanded = false
		pos := 0
		for i := 0; i < gi; i++ {
			pos++
			if m.searchResults[i].expanded {
				pos += len(m.searchResults[i].entries)
			}
		}
		m.searchCursor = pos
	}
}

func (m *Model) installFromSearch() tea.Cmd {
	if m.installInFlight || m.upgradeInFlight || m.removeInFlight {
		m.statusMsg = "operation already in progress"
		return nil
	}

	gi, ei := m.searchRowAt(m.searchCursor)
	if gi < 0 {
		return nil
	}

	var pkg model.Package
	if ei >= 0 {
		pkg = m.searchResults[gi].entries[ei]
	} else {
		if len(m.searchResults[gi].entries) == 0 {
			return nil
		}
		pkg = m.searchResults[gi].entries[0]
	}

	mgr := manager.BySource(pkg.Source)
	if mgr == nil {
		m.statusMsg = fmt.Sprintf("manager not found for %s", pkg.Source)
		return nil
	}

	installer, ok := mgr.(manager.Installer)
	if !ok {
		m.statusMsg = "this manager does not support installing packages"
		return nil
	}

	cmd := installer.InstallCmd(pkg.Name)
	cmdStr := strings.Join(cmd.Args, " ")
	needsSudo := len(cmd.Args) > 0 && cmd.Args[0] == "sudo"

	m.pendingUpgrade = &upgradeRequest{
		pkg:        pkg,
		cmd:        cmd,
		cmdStr:     cmdStr,
		privileged: isPrivilegedSource(pkg.Source),
		opLabel:    "install",
	}
	m.passwordInput.SetValue("")
	if needsSudo {
		m.confirmFocus = 0
		m.passwordInput.Focus()
		return tea.Batch(m.openModal(ModalConfirmUpgrade), textinput.Blink)
	}
	m.confirmFocus = 1
	m.passwordInput.Blur()
	return m.openModal(ModalConfirmUpgrade)
}

func (m Model) renderSearchView() string {
	w, h := m.width, m.height

	// Match detail.go's outer-panel sizing: border(2) + padding(4) = 6 cols of
	// chrome, so the inner content area is outerMaxW - 6.
	outerMaxW := w - 6
	if outerMaxW < 40 {
		outerMaxW = 40
	}
	if outerMaxW > 92 {
		outerMaxW = 92
	}
	usable := outerMaxW - 6

	var body strings.Builder
	body.WriteString(m.searchInput.View())
	body.WriteString("\n")

	if m.searchActive {
		body.WriteString("\n")
		body.WriteString(m.spinner.View())
		if m.searchPending > 0 {
			body.WriteString(StyleDim.Render(fmt.Sprintf(" searching %d managers...", m.searchPending)))
		}
		body.WriteString("\n")
	}

	if len(m.searchResults) == 0 {
		if !m.searchActive && !m.searchInput.Focused() {
			body.WriteString("\n")
			body.WriteString(StyleDim.Render("no results found"))
		}
		return m.composeSearchBlock(m.wrapSearchPanel(body.String(), usable))
	}

	body.WriteString("\n")

	colName := usable * 25 / 100
	colVer := usable * 12 / 100
	colBadge := badgeWidth + 2
	colDesc := usable - colName - colVer - colBadge

	header := padCell(StyleTableHeader.Render("PACKAGE"), colName) +
		padCell(StyleTableHeader.Render("VERSION"), colVer) +
		padCell(StyleTableHeader.Render("SOURCE"), colBadge) +
		StyleTableHeader.Render("DESCRIPTION")
	body.WriteString(header)
	body.WriteString("\n")
	body.WriteString(StyleDim.Render(strings.Repeat("─", usable)))
	body.WriteString("\n")

	installed := make(map[string]bool)
	for _, p := range m.allPkgs {
		installed[p.Name+":"+string(p.Source)] = true
	}

	listHeight := h - 14
	if listHeight < 5 {
		listHeight = 5
	}
	start := 0
	if m.searchCursor >= listHeight {
		start = m.searchCursor - listHeight + 1
	}

	row := 0
	for _, g := range m.searchResults {
		if row >= start+listHeight {
			break
		}

		if row >= start {
			best := g.entries[0]
			name := truncate(g.name, colName-4)
			ver := truncate(best.Version, colVer-2)
			badge := renderFixedBadge(best.Source)
			desc := truncate(best.Description, colDesc-1)
			isInst := installed[best.Name+":"+string(best.Source)]

			expandIcon := "▸"
			if g.expanded {
				expandIcon = "▾"
			}
			if len(g.entries) <= 1 {
				expandIcon = " "
			}

			if row == m.searchCursor {
				line := padCell(StyleSelected.Render(expandIcon+" "+name), colName) +
					padCell(StyleSelected.Render(ver), colVer) +
					padCell(badge, colBadge)
				if isInst {
					line += StyleDim.Render("✓ installed")
				} else {
					line += StyleSelected.Render(desc)
				}
				body.WriteString(line)
			} else {
				nameStyle := StyleNormal
				if isInst {
					nameStyle = StyleDim
				}
				line := padCell(nameStyle.Render(expandIcon+" "+name), colName) +
					padCell(StyleDim.Render(ver), colVer) +
					padCell(badge, colBadge)
				if isInst {
					line += StyleDim.Render("✓ installed")
				} else {
					line += StyleDim.Render(desc)
				}
				body.WriteString(line)
			}
			body.WriteString("\n")
		}
		row++

		if g.expanded {
			for ei, entry := range g.entries {
				if row >= start+listHeight {
					break
				}
				if row >= start {
					prefix := "├─"
					if ei == len(g.entries)-1 {
						prefix = "└─"
					}
					ver := truncate(entry.Version, colVer-2)
					badge := renderFixedBadge(entry.Source)
					desc := truncate(entry.Description, colDesc-1)
					isInst := installed[entry.Name+":"+string(entry.Source)]

					if row == m.searchCursor {
						line := padCell(StyleSelected.Render("  "+prefix), colName) +
							padCell(StyleSelected.Render(ver), colVer) +
							padCell(badge, colBadge)
						if isInst {
							line += StyleDim.Render("✓ installed")
						} else {
							line += StyleSelected.Render(desc)
						}
						body.WriteString(line)
					} else {
						line := padCell(StyleDim.Render("  "+prefix), colName) +
							padCell(StyleDim.Render(ver), colVer) +
							padCell(badge, colBadge)
						if isInst {
							line += StyleDim.Render("✓ installed")
						} else {
							line += StyleDim.Render(desc)
						}
						body.WriteString(line)
					}
					body.WriteString("\n")
				}
				row++
			}
		}
	}

	totalRows := m.searchRowCount()
	if totalRows > listHeight {
		pct := (m.searchCursor + 1) * 100 / totalRows
		body.WriteString(StyleDim.Render(fmt.Sprintf("%d/%d (%d%%)", m.searchCursor+1, totalRows, pct)))
		body.WriteString("\n")
	}

	return m.composeSearchBlock(m.wrapSearchPanel(body.String(), usable))
}

// wrapSearchPanel wraps the raw search body in a bordered rounded panel
// matching the detail view's outer panel, then horizontally centers the
// panel within the terminal width. innerW fixes the content-area width so
// the panel doesn't visibly resize when the body's longest line shrinks
// (e.g. short search input vs. full results table).
func (m Model) wrapSearchPanel(body string, innerW int) string {
	w := m.width
	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSubtext).
		Padding(1, 2).
		Width(innerW).
		Render(body)

	panelWidth := lipgloss.Width(panel)
	centerPad := (w - panelWidth) / 2
	if centerPad < 0 {
		centerPad = 0
	}
	pad := strings.Repeat(" ", centerPad)

	lines := strings.Split(panel, "\n")
	var out strings.Builder
	for i, line := range lines {
		out.WriteString(pad)
		out.WriteString(line)
		if i < len(lines)-1 {
			out.WriteString("\n")
		}
	}
	return out.String()
}

// composeSearchBlock wraps the horizontally-centered search content with a
// centered title at the top and centered keybind bar at the bottom, then
// vertically centers the whole block in the terminal to match the detail view.
func (m Model) composeSearchBlock(content string) string {
	w, h := m.width, m.height

	title := StyleTitle.Render("GlazePKG")
	if m.updateBanner != "" {
		title += "  " + StyleUpdateBanner.Render(m.updateBanner)
	}
	title = lipgloss.PlaceHorizontal(w, lipgloss.Center, title)

	keybinds := lipgloss.PlaceHorizontal(w, lipgloss.Center, strings.TrimLeft(m.renderStatusBar(), " "))

	block := lipgloss.JoinVertical(lipgloss.Left, title, "", content, "", keybinds)
	topFill := (h - lipgloss.Height(block)) / 2
	if topFill < 0 {
		topFill = 0
	}

	var out strings.Builder
	if topFill > 0 {
		out.WriteString(strings.Repeat("\n", topFill))
	}
	out.WriteString(block)
	return out.String()
}

func compareVersions(a, b string) int {
	partsA := strings.FieldsFunc(a, func(r rune) bool { return r == '.' || r == '-' || r == '_' })
	partsB := strings.FieldsFunc(b, func(r rune) bool { return r == '.' || r == '-' || r == '_' })

	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}

	for i := 0; i < maxLen; i++ {
		var pa, pb string
		if i < len(partsA) {
			pa = partsA[i]
		}
		if i < len(partsB) {
			pb = partsB[i]
		}

		na, okA := parseVersionNum(pa)
		nb, okB := parseVersionNum(pb)
		if okA && okB {
			if na != nb {
				return na - nb
			}
			continue
		}

		if pa != pb {
			if pa < pb {
				return -1
			}
			return 1
		}
	}
	return 0
}

func parseVersionNum(s string) (int, bool) {
	if s == "" {
		return 0, true
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}
