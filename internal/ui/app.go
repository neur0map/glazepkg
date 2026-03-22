package ui

import (
	"context"
	"fmt"
	"os/exec"
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
	{"All", 0, 0},                      // no filter
	{"< 1 MB", 0, 1 << 20},             // 0 – 1 MB
	{"1–10 MB", 1 << 20, 10 << 20},     // 1 – 10 MB
	{"10–100 MB", 10 << 20, 100 << 20}, // 10 – 100 MB
	{"> 100 MB", 100 << 20, 0},         // 100 MB+
	{"Has updates", -1, -1},            // special: only packages with updates
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

type upgradeResultMsg struct {
	pkg model.Package
	err error
}

type managerRescanMsg struct {
	source  model.Source
	pkgs    []model.Package
	updates map[string]string
	err     error
}

type pkgHelpMsg struct {
	lines []string
}

type upgradeRequest struct {
	pkg        model.Package
	cmd        *exec.Cmd
	cmdStr     string
	privileged bool
	password   string
}

type upgradeNotifClearMsg struct{}

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
	detailPkg   model.Package
	editingDesc bool
	descInput   textinput.Model
	userNotes   map[string]string

	// Diff
	currentDiff model.Diff
	diffSince   time.Time

	// Filter / Search
	filterInput textinput.Model
	filtering   bool
	sizeFilter  int // 0=all, cycles through sizeFilterLabels

	// Overlays
	showHelp          bool
	showExport        bool
	exportCursor      int
	showDeps          bool
	depsCursor        int
	showPkgHelp       bool
	pkgHelpLines      []string
	pkgHelpScroll     int
	confirmingUpgrade bool
	confirmFocus      int // 0 = password (privileged only), 1 = Yes, 2 = No
	pendingUpgrade    *upgradeRequest
	passwordInput     textinput.Model
	upgradeInFlight   bool
	upgradeCancel     context.CancelFunc
	upgradeNotifMsg   string
	upgradeNotifErr   bool

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

	pi := textinput.New()
	pi.Placeholder = "password"
	pi.CharLimit = 128
	pi.Prompt = "  Password: "
	pi.PromptStyle = StyleDim
	pi.TextStyle = StyleNormal
	pi.EchoMode = textinput.EchoPassword
	pi.EchoCharacter = '•'

	return Model{
		spinner:       sp,
		filterInput:   ti,
		descInput:     di,
		passwordInput: pi,
		view:          viewList,
		scanning:      true,
		descCache:     manager.NewDescriptionCache(),
		updateCache:   manager.NewUpdateCache(),
		depsCache:     manager.NewDepsCache(),
		userNotes:     snapshot.LoadNotes(),
		version:       version,
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

func fetchPkgHelp(name string) tea.Cmd {
	return func() tea.Msg {
		lines := tryPkgHelp(name)
		return pkgHelpMsg{lines: lines}
	}
}

func tryPkgHelp(name string) []string {
	// Try common help flags in order
	flags := [][]string{
		{name, "--help"},
		{name, "-h"},
		{name, "help"},
	}
	for _, args := range flags {
		cmd := exec.Command(args[0], args[1:]...)
		// Many tools write help to stderr
		out, err := cmd.CombinedOutput()
		if len(out) > 0 {
			return parseHelpOutput(string(out))
		}
		_ = err
	}
	return []string{"No help available for " + name}
}

func parseHelpOutput(raw string) []string {
	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		// Replace tabs with spaces for consistent rendering
		line = strings.ReplaceAll(line, "\t", "    ")
		lines = append(lines, line)
	}
	// Trim trailing empty lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	// Cap at 500 lines
	if len(lines) > 500 {
		lines = lines[:500]
		lines = append(lines, "... (truncated)")
	}
	return lines
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

func (m *Model) upgradeDetailPackage() tea.Cmd {
	if m.upgradeInFlight {
		m.statusMsg = "upgrade already in progress"
		return nil
	}

	pkg := m.detailPkg

	mgr := manager.BySource(pkg.Source)
	if mgr == nil {
		m.statusMsg = fmt.Sprintf("manager not found for %s", pkg.Source)
		return nil
	}
	if !mgr.Available() {
		m.statusMsg = fmt.Sprintf("%s is not available", pkg.Source)
		return nil
	}

	upgrader, ok := mgr.(manager.Upgrader)
	if !ok {
		m.statusMsg = manager.ErrUpgradeNotSupported.Error()
		return nil
	}

	cmd := upgrader.UpgradeCmd(pkg.Name)
	cmdStr := strings.Join(cmd.Args, " ")
	needsSudo := len(cmd.Args) > 0 && cmd.Args[0] == "sudo"
	req := &upgradeRequest{
		pkg:        pkg,
		cmd:        cmd,
		cmdStr:     cmdStr,
		privileged: isPrivilegedSource(pkg.Source),
	}

	m.pendingUpgrade = req
	m.confirmingUpgrade = true
	m.passwordInput.SetValue("")
	if needsSudo {
		m.confirmFocus = 0 // password field
		m.passwordInput.Focus()
		return textinput.Blink
	}
	m.confirmFocus = 1 // Yes button
	m.passwordInput.Blur()
	return nil
}

func (m *Model) rescanManager(source model.Source) tea.Cmd {
	cache := m.updateCache
	var keys []string
	for _, p := range m.allPkgs {
		if p.Source == source {
			keys = append(keys, p.Key())
		}
	}
	return func() tea.Msg {
		mgr := manager.BySource(source)
		if mgr == nil {
			return managerRescanMsg{source: source, err: fmt.Errorf("manager not found for %s", source)}
		}
		if !mgr.Available() {
			return managerRescanMsg{source: source, err: fmt.Errorf("%s is not available", source)}
		}
		pkgs, err := mgr.Scan()
		if err != nil {
			return managerRescanMsg{source: source, err: err}
		}
		if cache != nil {
			cache.Invalidate(keys)
		}
		updates := manager.FetchUpdates([]manager.Manager{mgr}, pkgs, cache)
		return managerRescanMsg{source: source, pkgs: pkgs, updates: updates}
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
		if m.scanning || m.loadingDescs || m.loadingUpdates || m.loadingDeps || m.upgradeInFlight {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case upgradeResultMsg:
		m.upgradeInFlight = false
		m.upgradeCancel = nil
		if msg.err != nil {
			errMsg := msg.err.Error()
			if len(errMsg) > 120 {
				errMsg = errMsg[:120] + "..."
			}
			m.upgradeNotifMsg = fmt.Sprintf("upgrade failed: %s", errMsg)
			m.upgradeNotifErr = true
			return m, tea.Tick(8*time.Second, func(time.Time) tea.Msg {
				return upgradeNotifClearMsg{}
			})
		}
		m.upgradeNotifMsg = fmt.Sprintf("%s upgraded successfully", msg.pkg.Name)
		m.upgradeNotifErr = false
		return m, tea.Batch(
			m.rescanManager(msg.pkg.Source),
			tea.Tick(5*time.Second, func(time.Time) tea.Msg {
				return upgradeNotifClearMsg{}
			}),
		)

	case upgradeNotifClearMsg:
		m.upgradeNotifMsg = ""
		return m, nil
	case managerRescanMsg:
		if msg.err != nil {
			m.statusMsg = "refresh error: " + msg.err.Error()
			return m, nil
		}
		// Index previous entries so we can preserve cached metadata.
		prev := make(map[string]model.Package)
		var next []model.Package
		for _, p := range m.allPkgs {
			if p.Source == msg.source {
				prev[p.Key()] = p
			} else {
				next = append(next, p)
			}
		}
		for _, p := range msg.pkgs {
			if old, ok := prev[p.Key()]; ok {
				if p.Description == "" {
					p.Description = old.Description
				}
				if len(p.DependsOn) == 0 {
					p.DependsOn = old.DependsOn
				}
				if len(p.RequiredBy) == 0 {
					p.RequiredBy = old.RequiredBy
				}
				if p.SizeBytes == 0 {
					p.SizeBytes = old.SizeBytes
				}
			}
			if note, ok := m.userNotes[p.Key()]; ok {
				p.Description = note
			}
			if latest, ok := msg.updates[p.Key()]; ok {
				p.LatestVersion = latest
			}
			next = append(next, p)
		}
		sort.Slice(next, func(i, j int) bool {
			return next[i].Name < next[j].Name
		})
		m.allPkgs = next
		manager.SaveScanCache(m.allPkgs)
		m.tabs = buildTabs(m.allPkgs)
		m.applyFilter()
		m.statusMsg = fmt.Sprintf("refreshed %s packages", msg.source)
		return m, nil

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

	case pkgHelpMsg:
		m.pkgHelpLines = msg.lines
		m.pkgHelpScroll = 0
		m.showPkgHelp = true
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

	if m.confirmingUpgrade && m.confirmFocus == 0 {
		var cmd tea.Cmd
		m.passwordInput, cmd = m.passwordInput.Update(msg)
		return m, cmd
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

	// Quit — cancel any in-flight upgrade first
	if key == "ctrl+c" {
		if m.upgradeCancel != nil {
			m.upgradeCancel()
			m.upgradeCancel = nil
		}
		return m, tea.Quit
	}

	if m.confirmingUpgrade {
		return m.handleUpgradeConfirmKey(msg)
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
		if m.upgradeInFlight {
			m.statusMsg = "upgrade in progress — press ctrl+c to force quit"
			return m, nil
		}
		return m, tea.Quit
	case "esc":
		if m.sizeFilter > 0 {
			m.sizeFilter = 0
			m.statusMsg = ""
			m.applyFilter()
		} else if m.filterInput.Value() != "" {
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

func (m Model) selectedPackage() (model.Package, bool) {
	if m.cursor < 0 || len(m.filteredPkgs) == 0 || m.cursor >= len(m.filteredPkgs) {
		return model.Package{}, false
	}
	return m.filteredPkgs[m.cursor], true
}

func (m *Model) handleDetailKey(key string) (tea.Model, tea.Cmd) {
	// Help overlay intercepts keys
	if m.showPkgHelp {
		maxScroll := len(m.pkgHelpLines) - (m.height - 8)
		if maxScroll < 0 {
			maxScroll = 0
		}
		switch key {
		case "esc", "q", "h":
			m.showPkgHelp = false
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
		m.showPkgHelp = false
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
	case "h":
		m.statusMsg = "loading help..."
		return m, fetchPkgHelp(m.detailPkg.Name)
	case "u":
		return m, m.upgradeDetailPackage()
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

	// Upgrade notification (above the status bar)
	if m.upgradeNotifMsg != "" {
		icon := " ✓ "
		color := ColorGreen
		label := "DONE"
		if m.upgradeNotifErr {
			icon = " ✗ "
			color = ColorRed
			label = "FAIL"
		} else if m.upgradeInFlight {
			icon = " " + m.spinner.View() + " "
			color = ColorCyan
			label = "UPGRADE"
		}
		badge := lipgloss.NewStyle().
			Background(color).
			Foreground(lipgloss.Color("#1a1b26")).
			Bold(true).
			Render(" " + label + " ")
		msgStyle := lipgloss.NewStyle().Foreground(color)
		b.WriteString("\n  " + badge + icon + msgStyle.Render(m.upgradeNotifMsg))
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  " + strings.Repeat("─", min(m.width-4, 120))))
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	content := b.String()

	// Render overlays on top
	if m.confirmingUpgrade {
		return content + "\n" + m.renderUpgradeConfirmOverlay()
	}
	if m.showHelp {
		return content + "\n" + renderHelpOverlay(m.width, m.height)
	}
	if m.showExport {
		return content + "\n" + renderExportOverlay(m.exportCursor, m.width, m.height)
	}
	if m.showDeps {
		return content + "\n" + renderDepsOverlay(m.detailPkg, m.depsCursor, m.width, m.height)
	}
	if m.showPkgHelp {
		return content + "\n" + renderPkgHelpOverlay(m.detailPkg.Name, m.pkgHelpLines, m.pkgHelpScroll, m.width, m.height)
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
		if m.showPkgHelp {
			return " " + formatBinds([]struct{ key, desc string }{
				{"j/k", "scroll"}, {"pgdn/pgup", "page"}, {"esc", "close"},
			})
		}
		if m.showDeps {
			return " " + formatBinds([]struct{ key, desc string }{
				{"j/k", "navigate"}, {"esc", "close"},
			})
		}
		var binds []struct{ key, desc string }
		if mgr := manager.BySource(m.detailPkg.Source); mgr != nil {
			if _, ok := mgr.(manager.Upgrader); ok {
				binds = append(binds, struct{ key, desc string }{"u", "upgrade"})
			}
		}
		binds = append(binds, struct{ key, desc string }{"e", "edit description"})
		if len(m.detailPkg.DependsOn) > 0 || len(m.detailPkg.RequiredBy) > 0 {
			binds = append(binds, struct{ key, desc string }{"d", "dependencies"})
		}
		binds = append(binds, struct{ key, desc string }{"h", "help/usage"})
		binds = append(binds, struct{ key, desc string }{"esc", "back"}, struct{ key, desc string }{"q", "quit"})
		return " " + formatBinds(binds)
	case viewDiff:
		return " " + formatBinds([]struct{ key, desc string }{
			{"esc", "back"}, {"q", "quit"},
		})
	}
	return ""
}

func (m *Model) runUpgradeRequest(req upgradeRequest) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.upgradeCancel = cancel
	// Rebuild the command with context so it can be cancelled on quit
	ctxCmd := exec.CommandContext(ctx, req.cmd.Args[0], req.cmd.Args[1:]...)
	ctxCmd.Dir = req.cmd.Dir
	ctxCmd.Env = req.cmd.Env
	return func() tea.Msg {
		defer cancel()
		if req.password != "" {
			ctxCmd.Stdin = strings.NewReader(req.password + "\n")
			req.password = ""
		}
		out, err := ctxCmd.CombinedOutput()
		if err != nil {
			msg := strings.TrimSpace(string(out))
			// Strip the sudo password prompt from the error output
			if idx := strings.Index(msg, "\n"); idx > 0 && strings.Contains(msg[:idx], "password") {
				msg = strings.TrimSpace(msg[idx+1:])
			}
			if msg != "" {
				if len(msg) > 800 {
					msg = msg[:800] + "..."
				}
				err = fmt.Errorf("%w: %s", err, msg)
			}
		}
		return upgradeResultMsg{pkg: req.pkg, err: err}
	}
}

func (m *Model) executePendingUpgrade() tea.Cmd {
	if m.pendingUpgrade == nil {
		m.confirmingUpgrade = false
		return nil
	}
	req := *m.pendingUpgrade
	if len(req.cmd.Args) > 0 && req.cmd.Args[0] == "sudo" {
		req.password = m.passwordInput.Value()
	}
	m.pendingUpgrade = nil
	m.confirmingUpgrade = false
	m.passwordInput.SetValue("")
	m.passwordInput.Blur()
	m.upgradeInFlight = true
	m.upgradeNotifMsg = fmt.Sprintf("upgrading %s...", req.pkg.Name)
	m.upgradeNotifErr = false
	return tea.Batch(m.spinner.Tick, m.runUpgradeRequest(req))
}

func (m *Model) needsSudoPassword() bool {
	return m.pendingUpgrade != nil && len(m.pendingUpgrade.cmd.Args) > 0 && m.pendingUpgrade.cmd.Args[0] == "sudo"
}

func (m *Model) handleUpgradeConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	hasPwField := m.needsSudoPassword()

	// Password field is focused — let textinput handle typing
	if hasPwField && m.confirmFocus == 0 {
		switch key {
		case "esc":
			m.cancelUpgradeConfirm()
			return m, nil
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
			return m, m.executePendingUpgrade()
		}
		// No
		m.cancelUpgradeConfirm()
	case "esc":
		m.cancelUpgradeConfirm()
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

func (m *Model) cancelUpgradeConfirm() {
	m.confirmingUpgrade = false
	m.pendingUpgrade = nil
	m.passwordInput.SetValue("")
	m.passwordInput.Blur()
	m.statusMsg = "upgrade cancelled"
}

func (m Model) renderUpgradeConfirmOverlay() string {
	req := m.pendingUpgrade
	if req == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(StyleOverlayTitle.Render("  Confirm Upgrade"))
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  " + strings.Repeat("─", 40)))
	b.WriteString("\n\n")
	b.WriteString(StyleNormal.Render(fmt.Sprintf("  Upgrade %s (%s)?", req.pkg.Name, req.pkg.Source)))
	b.WriteString("\n\n")
	b.WriteString(StyleDim.Render("  command:"))
	b.WriteString("\n")

	cmdStyle := lipgloss.NewStyle().Foreground(ColorCyan)
	b.WriteString("  " + cmdStyle.Render(req.cmdStr))
	b.WriteString("\n")

	overlayHeight := 11
	needsSudo := len(req.cmd.Args) > 0 && req.cmd.Args[0] == "sudo"

	if req.privileged {
		warnStyle := lipgloss.NewStyle().Foreground(ColorYellow)
		if needsSudo {
			b.WriteString("\n  " + warnStyle.Render("requires elevated privileges"))
			b.WriteString("\n\n")
			b.WriteString(m.passwordInput.View())
			b.WriteString("\n")
			overlayHeight = 16
		} else {
			b.WriteString("\n  " + warnStyle.Render("requires an elevated terminal"))
			b.WriteString("\n")
			overlayHeight = 13
		}
	}

	b.WriteString("\n")

	yesStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	noStyle := lipgloss.NewStyle().Foreground(ColorRed).Bold(true)

	if m.confirmFocus == 1 {
		yesStyle = yesStyle.Background(ColorGreen).Foreground(lipgloss.Color("#1a1b26"))
		noStyle = noStyle.Foreground(ColorSubtext)
	} else if m.confirmFocus == 2 {
		yesStyle = yesStyle.Foreground(ColorSubtext)
		noStyle = noStyle.Background(ColorRed).Foreground(lipgloss.Color("#1a1b26"))
	} else {
		yesStyle = yesStyle.Foreground(ColorSubtext)
		noStyle = noStyle.Foreground(ColorSubtext)
	}

	b.WriteString("      " + yesStyle.Render("  Yes  ") + "   " + noStyle.Render("  No  "))

	content := b.String()

	cmdLen := len(req.cmdStr) + 8
	overlayWidth := 48
	if cmdLen > overlayWidth {
		overlayWidth = cmdLen
	}
	if overlayWidth > m.width-4 {
		overlayWidth = m.width - 4
	}

	overlay := StyleOverlay.
		Width(overlayWidth).
		Height(overlayHeight).
		Render(content)

	return placeOverlay(m.width, m.height, overlay)
}

func isPrivilegedSource(source model.Source) bool {
	switch source {
	case model.SourceApt, model.SourceDnf, model.SourcePacman, model.SourceSnap,
		model.SourceApk, model.SourceXbps, model.SourceChocolatey:
		return true
	default:
		return false
	}
}
