package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

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

	m.searchResults = rankGroupsByName(m.searchResults, m.searchInput.Value())

	for i := range m.searchResults {
		entries := m.searchResults[i].entries
		sort.Slice(entries, func(a, b int) bool {
			return compareVersions(entries[a].Version, entries[b].Version) > 0
		})
	}
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
	m.confirmingUpgrade = true
	m.passwordInput.SetValue("")
	if needsSudo {
		m.confirmFocus = 0
		m.passwordInput.Focus()
		return textinput.Blink
	}
	m.confirmFocus = 1
	m.passwordInput.Blur()
	return nil
}

func (m Model) renderSearchView(b *strings.Builder) {
	b.WriteString("\n  ")
	b.WriteString(m.searchInput.View())
	b.WriteString("\n")

	if m.searchActive {
		b.WriteString("\n  ")
		b.WriteString(m.spinner.View())
		if m.searchPending > 0 {
			b.WriteString(StyleDim.Render(fmt.Sprintf(" searching %d managers...", m.searchPending)))
		}
		b.WriteString("\n")
	}

	if len(m.searchResults) == 0 {
		if !m.searchActive && !m.searchInput.Focused() {
			b.WriteString("\n")
			b.WriteString(StyleDim.Render("  no results found"))
		}
		return
	}

	b.WriteString("\n")

	usable := m.width - 4
	if usable < 60 {
		usable = 60
	}
	colName := usable * 25 / 100
	colVer := usable * 12 / 100
	colBadge := badgeWidth + 2
	colDesc := usable - colName - colVer - colBadge

	header := "  " +
		padCell(StyleTableHeader.Render("PACKAGE"), colName) +
		padCell(StyleTableHeader.Render("VERSION"), colVer) +
		padCell(StyleTableHeader.Render("SOURCE"), colBadge) +
		StyleTableHeader.Render("DESCRIPTION")
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  " + strings.Repeat("─", min(usable, m.width-4))))
	b.WriteString("\n")

	installed := make(map[string]bool)
	for _, p := range m.allPkgs {
		installed[p.Name+":"+string(p.Source)] = true
	}

	listHeight := m.height - 14
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
				line := "  " +
					padCell(StyleSelected.Render(expandIcon+" "+name), colName) +
					padCell(StyleSelected.Render(ver), colVer) +
					padCell(badge, colBadge)
				if isInst {
					line += StyleDim.Render("✓ installed")
				} else {
					line += StyleSelected.Render(desc)
				}
				b.WriteString(line)
			} else {
				nameStyle := StyleNormal
				if isInst {
					nameStyle = StyleDim
				}
				line := "  " +
					padCell(nameStyle.Render(expandIcon+" "+name), colName) +
					padCell(StyleDim.Render(ver), colVer) +
					padCell(badge, colBadge)
				if isInst {
					line += StyleDim.Render("✓ installed")
				} else {
					line += StyleDim.Render(desc)
				}
				b.WriteString(line)
			}
			b.WriteString("\n")
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
						line := "  " +
							padCell(StyleSelected.Render("  "+prefix), colName) +
							padCell(StyleSelected.Render(ver), colVer) +
							padCell(badge, colBadge)
						if isInst {
							line += StyleDim.Render("✓ installed")
						} else {
							line += StyleSelected.Render(desc)
						}
						b.WriteString(line)
					} else {
						line := "  " +
							padCell(StyleDim.Render("  "+prefix), colName) +
							padCell(StyleDim.Render(ver), colVer) +
							padCell(badge, colBadge)
						if isInst {
							line += StyleDim.Render("✓ installed")
						} else {
							line += StyleDim.Render(desc)
						}
						b.WriteString(line)
					}
					b.WriteString("\n")
				}
				row++
			}
		}
	}

	totalRows := m.searchRowCount()
	if totalRows > listHeight {
		pct := (m.searchCursor + 1) * 100 / totalRows
		b.WriteString(StyleDim.Render(fmt.Sprintf("  %d/%d (%d%%)", m.searchCursor+1, totalRows, pct)))
		b.WriteString("\n")
	}
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
