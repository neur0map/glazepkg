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
	"github.com/neur0map/glazepkg/internal/updater"
)

type view int

const (
	viewList view = iota
	viewDetail
	viewDiff
)

// Size filter thresholds (in bytes).
var sizeFilters = []struct {
	Label    string
	MinBytes int64
	MaxBytes int64
}{
	{"All", 0, 0},                             // no filter
	{"< 1 MB", 0, 1 << 20},                   // 0 – 1 MB
	{"1–10 MB", 1 << 20, 10 << 20},           // 1 – 10 MB
	{"10–100 MB", 10 << 20, 100 << 20},       // 10 – 100 MB
	{"> 100 MB", 100 << 20, 0},               // 100 MB+
	{"Has updates", -1, -1},                   // special: only packages with updates
}

type updateAvailableMsg struct {
	latest string
}

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

type updatesDoneMsg struct {
	updates map[string]string // key → latest version
}

type depsDoneMsg struct {
	deps map[string][]string // key → dependency list
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
	detailPkg    model.Package
	editingDesc  bool
	descInput    textinput.Model
	userNotes    map[string]string

	// Diff
	currentDiff model.Diff
	diffSince   time.Time

	// Filter / Search
	filterInput textinput.Model
	filtering   bool
	sizeFilter  int // 0=all, cycles through sizeFilterLabels

	// Overlays
	showHelp     bool
	showExport   bool
	exportCursor int
	showDeps     bool
	depsCursor   int

	// Descriptions
	loadingDescs bool
	descCache    *manager.DescriptionCache

	// Updates
	loadingUpdates bool
	updateCache    *manager.UpdateCache

	// Dependencies
	loadingDeps bool
	depsCache   *manager.DepsCache

	// Update banner
	version      string
	updateBanner string

	// Spinner
	spinner spinner.Model
}

func NewModel(version string) Model {
	ti := textinput.New()
	ti.Placeholder = "fuzzy search..."
	ti.CharLimit = 64
	ti.Prompt = "/ "
	ti.PromptStyle = StyleFilterPrompt
	ti.TextStyle = StyleFilterText

	di := textinput.New()
	di.Placeholder = "enter description..."
	di.CharLimit = 200
	di.Prompt = "Description: "
	di.PromptStyle = StyleDetailKey
	di.TextStyle = StyleDetailVal

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(ColorBlue)

	return Model{
		spinner:     sp,
		filterInput: ti,
		descInput:   di,
		view:        viewList,
		scanning:    true,
		descCache:   manager.NewDescriptionCache(),
		updateCache: manager.NewUpdateCache(),
		depsCache:   manager.NewDepsCache(),
		userNotes:   snapshot.LoadNotes(),
		version:     version,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadOrScan, checkForUpdate(m.version))
}

func checkForUpdate(currentVersion string) tea.Cmd {
	return func() tea.Msg {
		if currentVersion == "dev" {
			return nil
		}
		latest, err := updater.LatestVersion()
		if err != nil || latest == currentVersion {
			return nil
		}
		return updateAvailableMsg{latest: latest}
	}
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

func fetchDescriptions(pkgs []model.Package, cache *manager.DescriptionCache, skipKeys map[string]string) tea.Cmd {
	return func() tea.Msg {
		// Filter out packages with user-edited descriptions
		var toFetch []model.Package
		for _, p := range pkgs {
			if _, skip := skipKeys[p.Key()]; !skip {
				toFetch = append(toFetch, p)
			}
		}
		mgrs := manager.All()
		descs := manager.FetchDescriptions(mgrs, toFetch, cache)
		return descriptionsDoneMsg{descs: descs}
	}
}

func fetchDependencies(pkgs []model.Package, cache *manager.DepsCache) tea.Cmd {
	return func() tea.Msg {
		// Filter out packages that already have deps (e.g., populated during scan)
		var toFetch []model.Package
		for _, p := range pkgs {
			if len(p.DependsOn) == 0 {
				toFetch = append(toFetch, p)
			}
		}
		mgrs := manager.All()
		deps := manager.FetchDependencies(mgrs, toFetch, cache)
		return depsDoneMsg{deps: deps}
	}
}

func fetchUpdates(pkgs []model.Package, cache *manager.UpdateCache) tea.Cmd {
	return func() tea.Msg {
		mgrs := manager.All()
		updates := manager.FetchUpdates(mgrs, pkgs, cache)
		return updatesDoneMsg{updates: updates}
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
		if m.scanning || m.loadingDescs || m.loadingUpdates || m.loadingDeps {
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
		// Apply user notes immediately so they're visible before descriptions load
		for i := range m.allPkgs {
			if note, ok := m.userNotes[m.allPkgs[i].Key()]; ok {
				m.allPkgs[i].Description = note
			}
		}
		m.tabs = buildTabs(m.allPkgs)
		m.applyFilter()
		if msg.fromCache {
			age := manager.ScanCacheAge()
			m.statusMsg = fmt.Sprintf("loaded from cache (%s old) — press r to rescan", formatDuration(age))
		}
		// Dispatch background description fetch (skip packages with user notes)
		m.loadingDescs = true
		return m, fetchDescriptions(m.allPkgs, m.descCache, m.userNotes)

	case descriptionsDoneMsg:
		m.loadingDescs = false
		// Merge fetched descriptions (user-noted packages were excluded from fetch)
		for i := range m.allPkgs {
			key := m.allPkgs[i].Key()
			if _, hasNote := m.userNotes[key]; hasNote {
				continue
			}
			if desc, ok := msg.descs[key]; ok {
				m.allPkgs[i].Description = desc
			}
		}
		m.applyFilter()
		// Dispatch background update check and dependency fetch in parallel
		m.loadingUpdates = true
		m.loadingDeps = true
		return m, tea.Batch(
			fetchUpdates(m.allPkgs, m.updateCache),
			fetchDependencies(m.allPkgs, m.depsCache),
		)

	case updatesDoneMsg:
		m.loadingUpdates = false
		for i := range m.allPkgs {
			if latest, ok := msg.updates[m.allPkgs[i].Key()]; ok {
				m.allPkgs[i].LatestVersion = latest
			}
		}
		m.applyFilter()
		return m, nil

	case depsDoneMsg:
		m.loadingDeps = false
		for i := range m.allPkgs {
			key := m.allPkgs[i].Key()
			if deps, ok := msg.deps[key]; ok && len(deps) > 0 {
				m.allPkgs[i].DependsOn = deps
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
		// Carry over LatestVersion and Source from the list entry,
		// since QueryDetail always returns Source=pacman even for AUR.
		if m.cursor < len(m.filteredPkgs) {
			listPkg := m.filteredPkgs[m.cursor]
			if listPkg.Name == msg.pkg.Name {
				msg.pkg.LatestVersion = listPkg.LatestVersion
				msg.pkg.Source = listPkg.Source
			}
		}
		m.detailPkg = msg.pkg
		m.view = viewDetail
		return m, nil

	case updateAvailableMsg:
		m.updateBanner = fmt.Sprintf("↑ %s → %s available — run `gpk update`", m.version, msg.latest)
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

	if m.editingDesc {
		var cmd tea.Cmd
		m.descInput, cmd = m.descInput.Update(msg)
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

	// Edit mode intercepts keys
	if m.editingDesc {
		switch key {
		case "esc":
			m.editingDesc = false
			m.descInput.Blur()
			return m, nil
		case "enter":
			m.editingDesc = false
			m.descInput.Blur()
			desc := strings.TrimSpace(m.descInput.Value())
			pkgKey := m.detailPkg.Key()
			if desc == "" {
				delete(m.userNotes, pkgKey)
			} else {
				m.userNotes[pkgKey] = desc
			}
			m.detailPkg.Description = desc
			// Update in allPkgs too
			for i := range m.allPkgs {
				if m.allPkgs[i].Key() == pkgKey {
					m.allPkgs[i].Description = desc
					break
				}
			}
			m.applyFilter()
			if err := snapshot.SaveNotes(m.userNotes); err != nil {
				m.statusMsg = "note save error: " + err.Error()
			} else {
				m.statusMsg = "description saved"
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.descInput, cmd = m.descInput.Update(msg)
			return m, cmd
		}
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
	case "esc":
		if m.filterInput.Value() != "" {
			m.filterInput.SetValue("")
			m.applyFilter()
		}
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
	case "f":
		m.sizeFilter = (m.sizeFilter + 1) % len(sizeFilters)
		m.applyFilter()
		if m.sizeFilter == 0 {
			m.statusMsg = ""
		} else {
			m.statusMsg = "filter: " + sizeFilters[m.sizeFilter].Label
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
	// Deps overlay intercepts keys
	if m.showDeps {
		switch key {
		case "esc", "q", "d":
			m.showDeps = false
		case "j", "down":
			total := len(m.detailPkg.DependsOn) + len(m.detailPkg.RequiredBy)
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
			total := len(m.detailPkg.DependsOn) + len(m.detailPkg.RequiredBy)
			if total > 0 {
				m.depsCursor = total - 1
			}
		}
		return m, nil
	}

	switch key {
	case "esc", "q":
		m.showDeps = false
		m.view = viewList
	case "e":
		m.editingDesc = true
		m.descInput.SetValue(m.detailPkg.Description)
		m.descInput.Focus()
		return m, textinput.Blink
	case "d":
		if len(m.detailPkg.DependsOn) > 0 || len(m.detailPkg.RequiredBy) > 0 {
			m.showDeps = true
			m.depsCursor = 0
		}
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

	// Apply size / update filter
	if m.sizeFilter > 0 {
		sf := sizeFilters[m.sizeFilter]
		var matched, unknown []model.Package
		for _, p := range tabFiltered {
			if sf.MinBytes == -1 {
				// Special "Has updates" filter
				if p.LatestVersion != "" && p.LatestVersion != p.Version {
					matched = append(matched, p)
				}
				continue
			}
			if p.SizeBytes == 0 {
				unknown = append(unknown, p)
				continue
			}
			if sf.MinBytes > 0 && p.SizeBytes < sf.MinBytes {
				continue
			}
			if sf.MaxBytes > 0 && p.SizeBytes >= sf.MaxBytes {
				continue
			}
			matched = append(matched, p)
		}
		// Sort matched by size descending, then append unknown-size packages
		sort.Slice(matched, func(i, j int) bool {
			return matched[i].SizeBytes > matched[j].SizeBytes
		})
		tabFiltered = append(matched, unknown...)
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
	if m.updateBanner != "" {
		b.WriteString("  " + StyleUpdateBanner.Render(m.updateBanner))
	}
	b.WriteString("\n\n")

	switch m.view {
	case viewList:
		m.renderListView(&b)
	case viewDetail:
		b.WriteString(renderDetail(m.detailPkg, m.editingDesc, m.descInput.View()))
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
	if m.showDeps {
		return content + "\n" + renderDepsOverlay(m.detailPkg, m.depsCursor, m.width, m.height)
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
	showSize := m.sizeFilter > 0 && sizeFilters[m.sizeFilter].MinBytes != -1
	b.WriteString(renderPackageTable(m.filteredPkgs, m.cursor, listHeight, m.width, showSize))

	// Loading indicators
	if m.loadingDescs {
		b.WriteString("\n  ")
		b.WriteString(m.spinner.View())
		b.WriteString(StyleDim.Render(" Loading descriptions..."))
	} else if m.loadingUpdates || m.loadingDeps {
		b.WriteString("\n  ")
		b.WriteString(m.spinner.View())
		b.WriteString(StyleDim.Render(" Loading details..."))
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

	keyStyle := lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	sepStyle := lipgloss.NewStyle().Foreground(ColorSubtext)
	descStyle := lipgloss.NewStyle().Foreground(ColorText)
	sep := sepStyle.Render("  ")

	formatBinds := func(binds []struct{ key, desc string }) string {
		var parts []string
		for _, b := range binds {
			parts = append(parts, keyStyle.Render(b.key)+descStyle.Render(" "+b.desc))
		}
		return strings.Join(parts, sep)
	}

	switch m.view {
	case viewList:
		binds := []struct{ key, desc string }{
			{"/", "search"}, {"tab", "source"}, {"f", "filter"},
			{"enter", "detail"}, {"r", "rescan"}, {"s", "snap"},
			{"d", "diff"}, {"e", "export"}, {"?", "help"}, {"q", "quit"},
		}
		bar := formatBinds(binds)
		if m.sizeFilter > 0 {
			filterStyle := lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
			bar = filterStyle.Render("["+sizeFilters[m.sizeFilter].Label+"]") + sep + bar
		}
		return " " + bar
	case viewDetail:
		if m.editingDesc {
			return " " + formatBinds([]struct{ key, desc string }{
				{"enter", "save"}, {"esc", "cancel"},
			})
		}
		if m.showDeps {
			return " " + formatBinds([]struct{ key, desc string }{
				{"j/k", "navigate"}, {"esc", "close"},
			})
		}
		binds := []struct{ key, desc string }{
			{"e", "edit description"},
		}
		if len(m.detailPkg.DependsOn) > 0 || len(m.detailPkg.RequiredBy) > 0 {
			binds = append(binds, struct{ key, desc string }{"d", "dependencies"})
		}
		binds = append(binds, struct{ key, desc string }{"esc", "back"}, struct{ key, desc string }{"q", "quit"})
		return " " + formatBinds(binds)
	case viewDiff:
		return " " + formatBinds([]struct{ key, desc string }{
			{"esc", "back"}, {"q", "quit"},
		})
	}
	return ""
}
