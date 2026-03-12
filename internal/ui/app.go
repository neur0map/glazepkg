package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
	"github.com/neur0map/glazepkg/internal/snapshot"
)

type view int

const (
	viewList view = iota
	viewDetail
	viewDiff
)

type scanDoneMsg struct {
	pkgs      []model.Package
	err       error
	fromCache bool
}

type snapshotSavedMsg struct {
	path string
	err  error
}

type diffComputedMsg struct {
	diff  model.Diff
	since time.Time
	err   error
}

type detailLoadedMsg struct {
	pkg model.Package
	err error
}

type descriptionsDoneMsg struct {
	descs map[string]string
}

type exportDoneMsg struct {
	path string
	err  error
}

type Model struct {
	width  int
	height int

	// State
	allPkgs      []model.Package
	filteredPkgs []model.Package
	tabs         []tabItem
	activeTab    int
	cursor       int
	view         view
	scanning     bool
	statusMsg    string

	// Detail
	detailPkg model.Package

	// Diff
	currentDiff model.Diff
	diffSince   time.Time

	// Filter / Search
	filterInput textinput.Model
	filtering   bool

	// Overlays
	showHelp     bool
	showExport   bool
	exportCursor int

	// Descriptions
	loadingDescs bool
	descCache    *manager.DescriptionCache

	// Spinner
	spinner spinner.Model
}

func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "fuzzy search..."
	ti.CharLimit = 64
	ti.Prompt = "/ "
	ti.PromptStyle = StyleFilterPrompt
	ti.TextStyle = StyleFilterText

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(ColorBlue)

	return Model{
		spinner:     sp,
		filterInput: ti,
		view:        viewList,
		scanning:    true,
		descCache:   manager.NewDescriptionCache(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadOrScan)
}

// loadOrScan tries the scan cache first; if fresh, returns cached packages instantly.
// Otherwise does a full live scan and saves the result to cache.
func loadOrScan() tea.Msg {
	if cached := manager.LoadScanCache(); cached != nil {
		return scanDoneMsg{pkgs: cached, fromCache: true}
	}
	return freshScan()
}

func freshScan() tea.Msg {
	managers := manager.All()
	var all []model.Package

	for _, mgr := range managers {
		if !mgr.Available() {
			continue
		}
		pkgs, err := mgr.Scan()
		if err != nil {
			continue
		}
		all = append(all, pkgs...)
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})

	manager.SaveScanCache(all)
	return scanDoneMsg{pkgs: all}
}

// forceRescan always does a live scan, ignoring cache.
func forceRescan() tea.Msg {
	return freshScan()
}

func saveSnapshot(pkgs []model.Package) tea.Cmd {
	return func() tea.Msg {
		snap := snapshot.New(pkgs)
		path, err := snapshot.Save(snap)
		return snapshotSavedMsg{path: path, err: err}
	}
}

func computeDiff(pkgs []model.Package) tea.Cmd {
	return func() tea.Msg {
		prev, err := snapshot.Latest()
		if err != nil {
			return diffComputedMsg{err: err}
		}
		if prev == nil {
			return diffComputedMsg{err: fmt.Errorf("no previous snapshot")}
		}

		current := snapshot.New(pkgs)
		diff := model.ComputeDiff(prev, current)
		return diffComputedMsg{diff: diff, since: prev.Timestamp}
	}
}

func loadDetail(name string) tea.Cmd {
	return func() tea.Msg {
		pkg, err := manager.QueryDetail(name)
		return detailLoadedMsg{pkg: pkg, err: err}
	}
}

func fetchDescriptions(pkgs []model.Package, cache *manager.DescriptionCache) tea.Cmd {
	return func() tea.Msg {
		mgrs := manager.All()
		descs := manager.FetchDescriptions(mgrs, pkgs, cache)
		return descriptionsDoneMsg{descs: descs}
	}
}

func doExport(pkgs []model.Package, format int) tea.Cmd {
	return func() tea.Msg {
		path, err := exportPackages(pkgs, format)
		return exportDoneMsg{path: path, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		if m.scanning || m.loadingDescs {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case scanDoneMsg:
		m.scanning = false
		if msg.err != nil {
			m.statusMsg = "scan error: " + msg.err.Error()
			return m, nil
		}
		m.allPkgs = msg.pkgs
		m.tabs = buildTabs(m.allPkgs)
		m.applyFilter()
		if msg.fromCache {
			age := manager.ScanCacheAge()
			m.statusMsg = fmt.Sprintf("loaded from cache (%s old) — press r to rescan", formatDuration(age))
		}
		// Dispatch background description fetch
		m.loadingDescs = true
		return m, fetchDescriptions(m.allPkgs, m.descCache)

	case descriptionsDoneMsg:
		m.loadingDescs = false
		// Merge descriptions into packages
		for i := range m.allPkgs {
			if desc, ok := msg.descs[m.allPkgs[i].Key()]; ok {
				m.allPkgs[i].Description = desc
			}
		}
		m.applyFilter()
		return m, nil

	case snapshotSavedMsg:
		if msg.err != nil {
			m.statusMsg = "snapshot error: " + msg.err.Error()
		} else {
			m.statusMsg = "snapshot saved: " + msg.path
		}
		return m, nil

	case diffComputedMsg:
		if msg.err != nil {
			m.statusMsg = msg.err.Error()
			return m, nil
		}
		m.currentDiff = msg.diff
		m.diffSince = msg.since
		m.view = viewDiff
		return m, nil

	case detailLoadedMsg:
		if msg.err != nil {
			m.statusMsg = "detail error: " + msg.err.Error()
			return m, nil
		}
		m.detailPkg = msg.pkg
		m.view = viewDetail
		return m, nil

	case exportDoneMsg:
		m.showExport = false
		if msg.err != nil {
			m.statusMsg = "export error: " + msg.err.Error()
		} else {
			m.statusMsg = "exported: " + msg.path
		}
		return m, nil
	}

	if m.filtering {
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.applyFilter()
		return m, cmd
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Clear status message on any keypress
	m.statusMsg = ""

	// Quit always works
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	// Help overlay intercepts all keys
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	// Export overlay has its own cursor
	if m.showExport {
		switch key {
		case "esc", "q":
			m.showExport = false
		case "j", "down":
			if m.exportCursor < len(exportFormats)-1 {
				m.exportCursor++
			}
		case "k", "up":
			if m.exportCursor > 0 {
				m.exportCursor--
			}
		case "enter":
			return m, doExport(m.allPkgs, m.exportCursor)
		}
		return m, nil
	}

	// Filter mode intercepts keys
	if m.filtering {
		switch key {
		case "esc":
			m.filtering = false
			m.filterInput.Blur()
			m.filterInput.SetValue("")
			m.applyFilter()
			return m, nil
		case "enter":
			m.filtering = false
			m.filterInput.Blur()
			return m, nil
		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			m.applyFilter()
			return m, cmd
		}
	}

	switch m.view {
	case viewList:
		return m.handleListKey(key)
	case viewDetail:
		return m.handleDetailKey(key)
	case viewDiff:
		return m.handleDiffKey(key)
	}

	return m, nil
}

func (m *Model) handleListKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "/":
		m.filtering = true
		m.filterInput.Focus()
		return m, textinput.Blink
	case "?":
		m.showHelp = true
	case "tab":
		m.activeTab = (m.activeTab + 1) % len(m.tabs)
		m.cursor = 0
		m.applyFilter()
	case "shift+tab":
		m.activeTab--
		if m.activeTab < 0 {
			m.activeTab = len(m.tabs) - 1
		}
		m.cursor = 0
		m.applyFilter()
	case "j", "down":
		if m.cursor < len(m.filteredPkgs)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		if len(m.filteredPkgs) > 0 {
			m.cursor = len(m.filteredPkgs) - 1
		}
	case "ctrl+d", "pgdown":
		m.cursor += m.height / 2
		if m.cursor >= len(m.filteredPkgs) {
			m.cursor = len(m.filteredPkgs) - 1
		}
	case "ctrl+u", "pgup":
		m.cursor -= m.height / 2
		if m.cursor < 0 {
			m.cursor = 0
		}
	case "enter":
		if len(m.filteredPkgs) > 0 && m.cursor < len(m.filteredPkgs) {
			pkg := m.filteredPkgs[m.cursor]
			if pkg.Source == model.SourcePacman || pkg.Source == model.SourceAUR {
				return m, loadDetail(pkg.Name)
			}
			// For non-pacman, show what we have
			m.detailPkg = pkg
			m.view = viewDetail
		}
	case "r":
		m.scanning = true
		m.statusMsg = "rescanning..."
		return m, tea.Batch(m.spinner.Tick, forceRescan)
	case "s":
		m.statusMsg = "saving snapshot..."
		return m, saveSnapshot(m.allPkgs)
	case "d":
		m.statusMsg = "computing diff..."
		return m, computeDiff(m.allPkgs)
	case "e":
		m.showExport = true
		m.exportCursor = 0
	}
	return m, nil
}

func (m *Model) handleDetailKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.view = viewList
	}
	return m, nil
}

func (m *Model) handleDiffKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.view = viewList
	}
	return m, nil
}

func (m *Model) applyFilter() {
	source := ""
	if m.activeTab < len(m.tabs) {
		source = m.tabs[m.activeTab].Source
	}
	query := m.filterInput.Value()

	// First filter by source tab
	var tabFiltered []model.Package
	for _, p := range m.allPkgs {
		if source != "" && string(p.Source) != source {
			continue
		}
		// ALL tab hides dep sources — they only show in their own tab
		if source == "" && depSources[p.Source] {
			continue
		}
		tabFiltered = append(tabFiltered, p)
	}

	// Then apply fuzzy search
	m.filteredPkgs = fuzzyFilter(tabFiltered, query)

	if m.cursor >= len(m.filteredPkgs) {
		m.cursor = max(0, len(m.filteredPkgs)-1)
	}
}

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	var b strings.Builder

	// Title bar
	title := StyleTitle.Render("GlazePKG")
	b.WriteString(title)
	b.WriteString("\n\n")

	switch m.view {
	case viewList:
		m.renderListView(&b)
	case viewDetail:
		b.WriteString(renderDetail(m.detailPkg))
	case viewDiff:
		b.WriteString(renderDiffView(m.currentDiff, m.diffSince))
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  " + strings.Repeat("─", min(m.width-4, 120))))
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	content := b.String()

	// Render overlays on top
	if m.showHelp {
		return content + "\n" + renderHelpOverlay(m.width, m.height)
	}
	if m.showExport {
		return content + "\n" + renderExportOverlay(m.exportCursor, m.width, m.height)
	}

	return content
}

func (m Model) renderListView(b *strings.Builder) {
	// Tabs
	if len(m.tabs) > 0 {
		b.WriteString("  ")
		b.WriteString(renderTabs(m.tabs, m.activeTab))
		b.WriteString("\n")
		b.WriteString(StyleDim.Render("  " + strings.Repeat("─", min(m.width-4, 120))))
		b.WriteString("\n\n")
	}

	// Filter
	if m.filtering {
		b.WriteString("  ")
		b.WriteString(m.filterInput.View())
		b.WriteString("\n\n")
	} else if m.filterInput.Value() != "" {
		b.WriteString("  ")
		b.WriteString(StyleFilterPrompt.Render("/ "))
		b.WriteString(StyleFilterText.Render(m.filterInput.Value()))
		b.WriteString("\n\n")
	}

	// Scanning spinner
	if m.scanning {
		b.WriteString("  ")
		b.WriteString(m.spinner.View())
		b.WriteString(" Scanning package managers...")
		b.WriteString("\n")
		return
	}

	// Package table
	listHeight := m.height - 12
	if listHeight < 5 {
		listHeight = 5
	}
	b.WriteString(renderPackageTable(m.filteredPkgs, m.cursor, listHeight, m.width))

	// Loading descriptions indicator
	if m.loadingDescs {
		b.WriteString("\n  ")
		b.WriteString(m.spinner.View())
		b.WriteString(StyleDim.Render(" Loading descriptions..."))
	}
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "unknown"
	}
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

func (m Model) renderStatusBar() string {
	if m.statusMsg != "" {
		return StyleStatusBar.Render(m.statusMsg)
	}

	switch m.view {
	case viewList:
		return StyleStatusBar.Render("/: search  tab: source  enter: detail  r: rescan  s: snap  d: diff  e: export  ?: help  q: quit")
	case viewDetail:
		return StyleStatusBar.Render("esc: back    q: quit")
	case viewDiff:
		return StyleStatusBar.Render("esc: back    q: quit")
	}
	return ""
}
