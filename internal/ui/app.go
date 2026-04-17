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
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"

	"github.com/neur0map/glazepkg/internal/config"
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
	viewSearch
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
	pkg     model.Package
	err     error
	opLabel string
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
	opLabel    string // "upgrade" or "install"
}

type upgradeNotifClearMsg struct{}

type removeResultMsg struct {
	pkg model.Package
	err error
}

type removeRequest struct {
	pkg        model.Package
	cmd        *exec.Cmd
	cmdStr     string
	deepCmd    *exec.Cmd
	deepCmdStr string
	privileged bool
	password   string
}

type removeNotifClearMsg struct{}

type searchResultMsg struct {
	source model.Source
	pkgs   []model.Package
	err    error
}

type searchDoneMsg struct{}

type installResultMsg struct {
	pkg model.Package
	err error
}

type installNotifClearMsg struct{}

type searchResultGroup struct {
	name     string
	entries  []model.Package
	expanded bool
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

	// Multi-select
	multiSelect     bool
	selections      map[string]bool
	batchFocus      int // 0 = password, 1 = Yes, 2 = No
	pendingBatch    *batchConfirmState
	batchLog        []batchProgressMsg
	batchCurrentPkg string
	batchOps        []batchOp
	batchPassword   string
	batchOpLabel    string
	batchCtx        context.Context

	// Modal subsystem — see internal/ui/modal.go.
	modal        ModalType
	modalAnim    float64
	modalAnimVel float64
	modalOpening bool
	modalSpring  harmonica.Spring

	// Overlays
	exportCursor     int
	depsCursor       int
	pkgHelpLines     []string
	pkgHelpScroll    int
	confirmFocus     int // 0 = password (privileged only), 1 = Yes, 2 = No
	pendingUpgrade   *upgradeRequest
	passwordInput    textinput.Model
	upgradeInFlight  bool
	upgradingPkgName string
	upgradeCancel    context.CancelFunc
	upgradeNotifMsg  string
	upgradeNotifErr  bool

	// Remove
	removeFocus     int // 0 = mode, 1 = password, 2 = Yes, 3 = No
	removeMode      int // 0 = package only, 1 = package + deps
	pendingRemove   *removeRequest
	removeInFlight  bool
	removingPkgName string
	removeCancel    context.CancelFunc
	removeNotifMsg  string
	removeNotifErr  bool

	// Search + Install
	searchInput     textinput.Model
	searchActive    bool
	searchPending   int
	searchResults   []searchResultGroup
	searchCursor    int
	showPreRelease  bool
	installInFlight bool
	installCancel   context.CancelFunc
	installNotifMsg string
	installNotifErr bool

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

	// Theme picker
	themeCursor int
	themeList   []config.Theme
	prevThemeID string // for reverting on Esc
	appConfig   config.Config

	// Spinner
	spinner spinner.Model
}

func NewModel(version string) Model {
	// Load config and apply theme before building styles
	cfg := config.Load()
	theme := config.ResolveTheme(cfg.Appearance.Theme)
	ApplyTheme(theme)

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

	si := textinput.New()
	si.Placeholder = "search packages..."
	si.CharLimit = 64
	si.Prompt = "  search: "
	si.PromptStyle = lipgloss.NewStyle().Foreground(ColorCyan)
	si.TextStyle = StyleNormal

	return Model{
		appConfig:     cfg,
		spinner:       sp,
		searchInput:   si,
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
		opLabel:    "upgrade",
	}

	m.pendingUpgrade = req
	m.passwordInput.SetValue("")
	if needsSudo {
		m.confirmFocus = 0 // password field
		m.passwordInput.Focus()
		return tea.Batch(m.openModal(ModalConfirmUpgrade), textinput.Blink)
	}
	m.confirmFocus = 1 // Yes button
	m.passwordInput.Blur()
	return m.openModal(ModalConfirmUpgrade)
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
		if m.scanning || m.loadingDescs || m.loadingUpdates || m.loadingDeps || m.upgradeInFlight || m.removeInFlight || m.searchActive {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case modalAnimTickMsg:
		if m.modal == ModalNone {
			return m, nil
		}
		target := 1.0
		if !m.modalOpening {
			target = 0.0
		}
		m.modalAnim, m.modalAnimVel = m.modalSpring.Update(m.modalAnim, m.modalAnimVel, target)

		// Short-circuit close: as soon as the modal overshoots past 0, it's
		// visually gone. Stop the tick chain immediately — any further spring
		// oscillation around 0 is invisible and wastes CPU.
		if !m.modalOpening && m.modalAnim <= 0 {
			m.modal = ModalNone
			m.modalAnim = 0
			m.modalAnimVel = 0
			m.resetTransientModalState()
			return m, nil
		}

		// Short-circuit open: once the modal reaches full size, stop ticking.
		// clipModalByAnim clamps anim>=1 to the full box, so any continued
		// oscillation would be visually invisible but CPU-costly.
		if m.modalOpening && m.modalAnim >= 1.0 {
			m.modalAnim = 1.0
			return m, nil
		}

		return m, modalAnimTick()
	case upgradeResultMsg:
		m.upgradeInFlight = false
		m.upgradingPkgName = ""
		m.upgradeCancel = nil
		op := msg.opLabel
		if op == "" {
			op = "upgrade"
		}
		if msg.err != nil {
			errMsg := msg.err.Error()
			if len(errMsg) > 120 {
				errMsg = errMsg[:120] + "..."
			}
			m.upgradeNotifMsg = fmt.Sprintf("%s failed: %s", op, errMsg)
			m.upgradeNotifErr = true
			return m, tea.Tick(8*time.Second, func(time.Time) tea.Msg {
				return upgradeNotifClearMsg{}
			})
		}
		m.upgradeNotifMsg = fmt.Sprintf("%s %s successfully", msg.pkg.Name, pastTense(op))
		m.upgradeNotifErr = false
		// Seed the description cache so the new package shows its description after rescan
		if msg.pkg.Description != "" {
			m.descCache.Set(msg.pkg.Key(), msg.pkg.Description)
		}
		return m, tea.Batch(
			m.rescanManager(msg.pkg.Source),
			tea.Tick(5*time.Second, func(time.Time) tea.Msg {
				return upgradeNotifClearMsg{}
			}),
		)

	case upgradeNotifClearMsg:
		m.upgradeNotifMsg = ""
		m.batchLog = nil
		return m, nil

	case removeResultMsg:
		m.removeInFlight = false
		m.removingPkgName = ""
		m.removeCancel = nil
		if msg.err != nil {
			errMsg := msg.err.Error()
			if len(errMsg) > 120 {
				errMsg = errMsg[:120] + "..."
			}
			m.removeNotifMsg = fmt.Sprintf("remove failed: %s", errMsg)
			m.removeNotifErr = true
			return m, tea.Tick(8*time.Second, func(time.Time) tea.Msg {
				return removeNotifClearMsg{}
			})
		}
		m.removeNotifMsg = fmt.Sprintf("%s removed successfully", msg.pkg.Name)
		m.removeNotifErr = false
		// Go back to list view since the package no longer exists
		m.view = viewList
		return m, tea.Batch(
			m.rescanManager(msg.pkg.Source),
			tea.Tick(5*time.Second, func(time.Time) tea.Msg {
				return removeNotifClearMsg{}
			}),
		)

	case removeNotifClearMsg:
		m.removeNotifMsg = ""
		return m, nil

	case searchResultMsg:
		m.handleSearchResult(msg)
		return m, nil

	case batchProgressMsg:
		return m.handleBatchProgress(msg)

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
			if p.Description == "" {
				if desc, ok := m.descCache.Get(p.Key()); ok && desc != "" {
					p.Description = desc
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
		m.statusMsg = "" // clear "loading help..." set at trigger
		return m, m.openModal(ModalPkgHelp)

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
		if msg.err != nil {
			m.statusMsg = "export error: " + msg.err.Error()
		} else {
			m.statusMsg = "exported: " + msg.path
		}
		return m, nil
	}

	if m.view == viewSearch && m.searchInput.Focused() {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
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
	key := normalizeHotkey(msg.String())

	// Quit — cancel any in-flight upgrade first
	if key == "ctrl+c" {
		if m.upgradeCancel != nil {
			m.upgradeCancel()
			m.upgradeCancel = nil
		}
		if m.removeCancel != nil {
			m.removeCancel()
			m.removeCancel = nil
		}
		return m, tea.Quit
	}

	// Modal guard: a modal absorbs all input while open.
	if m.modal != ModalNone {
		return m.handleModalKey(msg)
	}

	// Clear status message on any non-modal keypress.
	m.statusMsg = ""

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
	case viewSearch:
		return m.handleSearchKey(msg)
	}

	return m, nil
}

func (m *Model) handleListKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		if m.upgradeInFlight || m.removeInFlight {
			m.statusMsg = "operation in progress — press ctrl+c to force quit"
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
	case "/", "ctrl+f":
		m.filtering = true
		m.filterInput.Focus()
		return m, textinput.Blink
	case "?", "h":
		return m, m.openModal(ModalHelp)
	case "tab":
		if len(m.tabs) == 0 {
			return m, nil
		}
		m.activeTab = (m.activeTab + 1) % len(m.tabs)
		m.cursor = 0
		m.applyFilter()
	case "shift+tab":
		if len(m.tabs) == 0 {
			return m, nil
		}
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
		m.exportCursor = 0
		return m, m.openModal(ModalExport)
	case "i":
		return m, m.enterSearchView()
	case "m":
		m.toggleMultiSelect()
	case "t":
		return m, m.openThemePicker()
	case " ":
		if m.multiSelect {
			m.toggleSelection()
		}
	case "u":
		if m.multiSelect && m.selectionCount() > 0 {
			return m, m.batchUpgradeSelected()
		}
	case "x":
		if m.multiSelect && m.selectionCount() > 0 {
			return m, m.batchRemoveSelected()
		}
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
	switch key {
	case "esc", "q":
		m.view = viewList
	case "e":
		m.editingDesc = true
		m.descInput.SetValue(m.detailPkg.Description)
		m.descInput.Focus()
		return m, textinput.Blink
	case "d":
		if len(m.detailPkg.DependsOn) > 0 || len(m.detailPkg.RequiredBy) > 0 {
			m.depsCursor = 0
			return m, m.openModal(ModalDeps)
		}
	case "h":
		m.statusMsg = "loading help..."
		return m, fetchPkgHelp(m.detailPkg.Name)
	case "u":
		return m, m.upgradeDetailPackage()
	case "x":
		return m, m.removeDetailPackage()
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

// openThemePicker initializes theme list and prevThemeID, then opens ModalTheme.
// Returns the animation-tick Cmd from openModal.
func (m *Model) openThemePicker() tea.Cmd {
	// Build theme list: System first, then all named themes
	systemTheme := config.Theme{
		ID:      "system",
		Name:    "System (uses terminal colors)",
		Palette: config.SystemPalette(),
	}
	all := config.AllThemes()
	m.themeList = append([]config.Theme{systemTheme}, all...)
	m.prevThemeID = m.appConfig.Appearance.Theme
	m.themeCursor = 0
	for i, t := range m.themeList {
		if t.ID == m.prevThemeID {
			m.themeCursor = i
			break
		}
	}
	return m.openModal(ModalTheme)
}

// refreshInputStyles updates text input and spinner styles after a theme change.
func (m *Model) refreshInputStyles() {
	m.filterInput.PromptStyle = StyleFilterPrompt
	m.filterInput.TextStyle = StyleFilterText
	m.descInput.PromptStyle = StyleDetailKey
	m.descInput.TextStyle = StyleDetailVal
	m.passwordInput.PromptStyle = StyleDim
	m.passwordInput.TextStyle = StyleNormal
	m.searchInput.PromptStyle = lipgloss.NewStyle().Foreground(ColorCyan)
	m.searchInput.TextStyle = StyleNormal
	m.spinner.Style = lipgloss.NewStyle().Foreground(ColorBlue)
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

	// Then apply ranked search (name prefix > name contains > description, with fuzzy fallback)
	m.filteredPkgs = rankPackages(tabFiltered, query)

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
	case viewSearch:
		m.renderSearchView(&b)
	}

	// Batch progress log
	if len(m.batchLog) > 0 {
		b.WriteString("\n")
		maxShow := 4
		if !m.upgradeInFlight {
			maxShow = 8 // show more after completion
		}
		start := 0
		if len(m.batchLog) > maxShow {
			start = len(m.batchLog) - maxShow
		}
		for _, entry := range m.batchLog[start:] {
			icon := lipgloss.NewStyle().Foreground(ColorGreen).Render("  ✓ ")
			if entry.status == "failed" {
				icon = lipgloss.NewStyle().Foreground(ColorRed).Render("  ✗ ")
			}
			nameStyle := StyleDim
			line := icon + nameStyle.Render(entry.name)
			if entry.status == "failed" && entry.err != "" {
				errTrunc := entry.err
				if len(errTrunc) > 60 {
					errTrunc = errTrunc[:60] + "..."
				}
				line += StyleDim.Render(" — " + errTrunc)
			}
			b.WriteString(line + "\n")
		}
	}

	// Operation notifications (above the status bar)
	if m.upgradeNotifMsg != "" {
		opLabel := "UPGRADE"
		if m.batchOpLabel != "" {
			opLabel = strings.ToUpper(m.batchOpLabel)
		}
		b.WriteString("\n  " + renderOpNotification(m.upgradeNotifMsg, m.upgradeNotifErr, m.upgradeInFlight, opLabel, m.spinner.View()))
	}
	if m.removeNotifMsg != "" {
		b.WriteString("\n  " + renderOpNotification(m.removeNotifMsg, m.removeNotifErr, m.removeInFlight, "REMOVE", m.spinner.View()))
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  " + strings.Repeat("─", min(m.width-4, 120))))
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	content := b.String()

	return m.renderModal(content)
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
	b.WriteString(renderPackageTable(m.filteredPkgs, m.cursor, listHeight, m.width, showSize, m.upgradingPkgName, m.removingPkgName, m.selections))

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
		if m.width <= 0 {
			return strings.Join(parts, sep)
		}
		// Wrap into multiple lines when too wide
		maxW := m.width - 2
		var lines []string
		var line []string
		w := 0
		sepW := lipgloss.Width(sep)
		for _, p := range parts {
			pw := lipgloss.Width(p)
			needed := pw
			if len(line) > 0 {
				needed += sepW
			}
			if w+needed > maxW && len(line) > 0 {
				lines = append(lines, strings.Join(line, sep))
				line = nil
				w = 0
			}
			line = append(line, p)
			w += needed
		}
		if len(line) > 0 {
			lines = append(lines, strings.Join(line, sep))
		}
		return strings.Join(lines, "\n ")
	}

	switch m.view {
	case viewList:
		var binds []struct{ key, desc string }
		if m.multiSelect {
			count := m.selectionCount()
			selectStyle := lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
			prefix := selectStyle.Render(fmt.Sprintf("[%d selected]", count))
			binds = []struct{ key, desc string }{
				{"space", "toggle"}, {"u", "upgrade"}, {"x", "remove"},
				{"/", "search"}, {"m", "exit select"}, {"q", "quit"},
			}
			return " " + prefix + "  " + formatBinds(binds)
		}
		binds = []struct{ key, desc string }{
			{"/", "search"}, {"tab", "source"}, {"f", "filter"},
			{"enter", "detail"}, {"r", "rescan"}, {"s", "snap"},
			{"m", "select"}, {"i", "search/install"}, {"d", "diff"}, {"e", "export"}, {"t", "theme"}, {"?", "help"}, {"q", "quit"},
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
		if m.modal == ModalPkgHelp {
			return " " + formatBinds([]struct{ key, desc string }{
				{"j/k", "scroll"}, {"pgdn/pgup", "page"}, {"esc", "close"},
			})
		}
		if m.modal == ModalDeps {
			return " " + formatBinds([]struct{ key, desc string }{
				{"j/k", "navigate"}, {"esc", "close"},
			})
		}
		var binds []struct{ key, desc string }
		if mgr := manager.BySource(m.detailPkg.Source); mgr != nil {
			if _, ok := mgr.(manager.Upgrader); ok {
				binds = append(binds, struct{ key, desc string }{"u", "upgrade"})
			}
			if _, ok := mgr.(manager.Remover); ok {
				binds = append(binds, struct{ key, desc string }{"x", "remove"})
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
	case viewSearch:
		if m.searchInput.Focused() {
			return " " + formatBinds([]struct{ key, desc string }{
				{"enter", "search"}, {"esc", "back"},
			})
		}
		binds := []struct{ key, desc string }{
			{"j/k", "navigate"}, {"enter", "expand"}, {"i", "install"},
			{"p", "pre-release"}, {"/", "new search"}, {"q", "back"},
		}
		if m.showPreRelease {
			preStyle := lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
			return " " + preStyle.Render("[pre-release]") + "  " + formatBinds(binds)
		}
		return " " + formatBinds(binds)
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
			msg := extractErrorLines(string(out))
			if msg != "" {
				err = fmt.Errorf("%w: %s", err, msg)
			}
		}
		return upgradeResultMsg{pkg: req.pkg, err: err, opLabel: req.opLabel}
	}
}

// extractErrorLines pulls the meaningful error from command output.
// Looks for lines starting with "E:", "error:", "Error:", "fatal:", or
// "Sorry," (sudo). Falls back to the last non-empty line if nothing
// matches. Strips sudo password prompts.
func extractErrorLines(raw string) string {
	lines := strings.Split(raw, "\n")
	var errLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip sudo password prompts
		if strings.Contains(line, "password") && strings.Contains(line, ":") && len(line) < 80 {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "e:") ||
			strings.HasPrefix(lower, "error:") ||
			strings.HasPrefix(lower, "error -") ||
			strings.HasPrefix(lower, "fatal:") ||
			strings.HasPrefix(lower, "sorry,") {
			errLines = append(errLines, line)
		}
	}
	if len(errLines) > 0 {
		msg := strings.Join(errLines, "; ")
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		return msg
	}
	// Fallback: last non-empty line
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" && !strings.Contains(line, "password") {
			if len(line) > 200 {
				line = line[:200] + "..."
			}
			return line
		}
	}
	return ""
}

func (m *Model) executePendingUpgrade() tea.Cmd {
	if m.pendingUpgrade == nil {
		return nil
	}
	req := *m.pendingUpgrade
	if len(req.cmd.Args) > 0 && req.cmd.Args[0] == "sudo" {
		req.password = m.passwordInput.Value()
	}
	m.pendingUpgrade = nil
	m.passwordInput.SetValue("")
	m.passwordInput.Blur()
	m.upgradeInFlight = true
	m.upgradingPkgName = req.pkg.Name
	op := gerund(req.opLabel)
	m.upgradeNotifMsg = fmt.Sprintf("%s %s...", op, req.pkg.Name)
	m.upgradeNotifErr = false
	return tea.Batch(m.spinner.Tick, m.runUpgradeRequest(req))
}

func (m *Model) needsSudoPassword() bool {
	return m.pendingUpgrade != nil && len(m.pendingUpgrade.cmd.Args) > 0 && m.pendingUpgrade.cmd.Args[0] == "sudo"
}

func (m *Model) cancelUpgradeConfirm() {
	m.pendingUpgrade = nil
	m.passwordInput.SetValue("")
	m.passwordInput.Blur()
	m.statusMsg = "upgrade cancelled"
}

// upgradeConfirmBody renders the confirm-upgrade modal body: action summary,
// optional password field, Yes/No buttons with focus highlight on m.confirmFocus.
func upgradeConfirmBody(m *Model) string {
	req := m.pendingUpgrade
	if req == nil {
		return ""
	}
	var b strings.Builder
	title := "Upgrade"
	if req.opLabel == "install" {
		title = "Install"
	}
	b.WriteString(StyleNormal.Render(fmt.Sprintf("%s %s (%s)?", title, req.pkg.Name, req.pkg.Source)))
	b.WriteString("\n\n")
	b.WriteString(StyleDim.Render("command:"))
	b.WriteString("\n")
	cmdStyle := lipgloss.NewStyle().Foreground(ColorCyan)
	b.WriteString(cmdStyle.Render(req.cmdStr))
	b.WriteString("\n")

	needsSudo := len(req.cmd.Args) > 0 && req.cmd.Args[0] == "sudo"
	if req.privileged {
		warnStyle := lipgloss.NewStyle().Foreground(ColorYellow)
		if needsSudo {
			b.WriteString("\n" + warnStyle.Render("requires elevated privileges"))
			b.WriteString("\n\n")
			b.WriteString(m.passwordInput.View())
			b.WriteString("\n")
		} else {
			b.WriteString("\n" + warnStyle.Render("requires an elevated terminal"))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	yesStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	noStyle := lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
	switch m.confirmFocus {
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
	b.WriteString("    " + yesStyle.Render("  Yes  ") + "   " + noStyle.Render("  No  "))
	return b.String()
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

// --- Remove flow ---

func (m *Model) removeDetailPackage() tea.Cmd {
	if m.removeInFlight || m.upgradeInFlight {
		m.statusMsg = "operation already in progress"
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

	remover, ok := mgr.(manager.Remover)
	if !ok {
		m.statusMsg = "this package manager does not support removing packages"
		return nil
	}

	cmd := remover.RemoveCmd(pkg.Name)
	req := &removeRequest{
		pkg:        pkg,
		cmd:        cmd,
		cmdStr:     strings.Join(cmd.Args, " "),
		privileged: isPrivilegedSource(pkg.Source),
	}

	if deep, ok := mgr.(manager.DeepRemover); ok {
		deepCmd := deep.RemoveCmdWithDeps(pkg.Name)
		req.deepCmd = deepCmd
		req.deepCmdStr = strings.Join(deepCmd.Args, " ")
	}

	m.pendingRemove = req
	m.removeMode = 0
	m.passwordInput.SetValue("")

	needsSudo := len(cmd.Args) > 0 && cmd.Args[0] == "sudo"
	hasDeep := req.deepCmd != nil

	if hasDeep {
		m.removeFocus = 0
		m.passwordInput.Blur()
		return m.openModal(ModalConfirmRemove)
	}
	if needsSudo {
		m.removeFocus = 1
		m.passwordInput.Focus()
		return tea.Batch(m.openModal(ModalConfirmRemove), textinput.Blink)
	}
	m.removeFocus = 2
	m.passwordInput.Blur()
	return m.openModal(ModalConfirmRemove)
}

func (m *Model) executeRemove() tea.Cmd {
	if m.pendingRemove == nil {
		return nil
	}
	req := *m.pendingRemove

	// Use deep remove command if that mode was selected
	if m.removeMode == 1 && req.deepCmd != nil {
		req.cmd = req.deepCmd
		req.cmdStr = req.deepCmdStr
	}

	if len(req.cmd.Args) > 0 && req.cmd.Args[0] == "sudo" {
		req.password = m.passwordInput.Value()
	}

	m.pendingRemove = nil
	m.passwordInput.SetValue("")
	m.passwordInput.Blur()
	m.removeInFlight = true
	m.removingPkgName = req.pkg.Name
	m.removeNotifMsg = fmt.Sprintf("removing %s...", req.pkg.Name)
	m.removeNotifErr = false
	return tea.Batch(m.spinner.Tick, m.runRemoveRequest(req))
}

func (m *Model) runRemoveRequest(req removeRequest) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.removeCancel = cancel
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
			msg := extractErrorLines(string(out))
			if msg != "" {
				err = fmt.Errorf("%w: %s", err, msg)
			}
		}
		return removeResultMsg{pkg: req.pkg, err: err}
	}
}

func (m *Model) cancelRemoveConfirm() {
	m.pendingRemove = nil
	m.removeMode = 0
	m.passwordInput.SetValue("")
	m.passwordInput.Blur()
	m.statusMsg = "remove cancelled"
}

func (m *Model) removeNeedsSudo() bool {
	return m.pendingRemove != nil && len(m.pendingRemove.cmd.Args) > 0 && m.pendingRemove.cmd.Args[0] == "sudo"
}

// removeConfirmBody renders the confirm-remove modal body: optional
// DeepRemover mode selector, orphaned-deps preview + conflicts, command
// preview, optional password field, and Yes/No buttons with focus highlight
// driven by m.removeFocus / m.removeMode.
func removeConfirmBody(m *Model) string {
	req := m.pendingRemove
	if req == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(StyleNormal.Render(fmt.Sprintf("Remove %s (%s)?", req.pkg.Name, req.pkg.Source)))
	b.WriteString("\n")

	// RequiredBy warning
	if len(req.pkg.RequiredBy) > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(ColorYellow)
		reqList := strings.Join(req.pkg.RequiredBy, ", ")
		if len(reqList) > 60 {
			reqList = reqList[:60] + "..."
		}
		b.WriteString("\n" + warnStyle.Render("⚠ required by: "+reqList))
		b.WriteString("\n")
	}

	// Mode selector (DeepRemover only)
	if req.deepCmd != nil {
		b.WriteString("\n")
		b.WriteString(StyleDim.Render("mode:"))
		b.WriteString("\n")

		modeStyle0 := StyleNormal
		modeStyle1 := StyleNormal
		if m.removeFocus == 0 && m.removeMode == 0 {
			modeStyle0 = lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
		} else if m.removeFocus == 0 && m.removeMode == 1 {
			modeStyle1 = lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
		}

		prefix0 := "  "
		prefix1 := "  "
		if m.removeMode == 0 {
			prefix0 = "› "
		} else {
			prefix1 = "› "
		}

		b.WriteString(modeStyle0.Render(prefix0 + "Remove package only"))
		b.WriteString("\n")
		b.WriteString(modeStyle1.Render(prefix1 + "Remove package + orphaned deps"))
		b.WriteString("\n")

		// Show orphaned deps when deep remove selected
		if m.removeMode == 1 && len(req.pkg.DependsOn) > 0 {
			b.WriteString("\n")
			b.WriteString(StyleDim.Render("orphaned deps to remove:"))
			b.WriteString("\n")
			depStyle := lipgloss.NewStyle().Foreground(ColorSubtext)
			depList := strings.Join(req.pkg.DependsOn, ", ")
			if len(depList) > 60 {
				depList = depList[:60] + "..."
			}
			b.WriteString(depStyle.Render("  " + depList))
			b.WriteString("\n")

			// Flag deps still required by other packages
			var conflicts []string
			for _, dep := range req.pkg.DependsOn {
				for _, p := range m.allPkgs {
					if p.Name == dep && len(p.RequiredBy) > 0 {
						others := filterOut(p.RequiredBy, req.pkg.Name)
						if len(others) > 0 {
							conflicts = append(conflicts, fmt.Sprintf("%s is required by: %s", dep, strings.Join(others, ", ")))
						}
					}
				}
			}
			if len(conflicts) > 0 {
				warnStyle := lipgloss.NewStyle().Foreground(ColorYellow)
				for _, c := range conflicts {
					if len(c) > 60 {
						c = c[:60] + "..."
					}
					b.WriteString("\n" + warnStyle.Render("⚠ "+c))
				}
				b.WriteString("\n")
			}
		}
	}

	// Command preview
	cmdStr := req.cmdStr
	if m.removeMode == 1 && req.deepCmdStr != "" {
		cmdStr = req.deepCmdStr
	}
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("command:"))
	b.WriteString("\n")
	cmdStyle := lipgloss.NewStyle().Foreground(ColorCyan)
	b.WriteString(cmdStyle.Render(cmdStr))
	b.WriteString("\n")

	needsSudo := len(req.cmd.Args) > 0 && req.cmd.Args[0] == "sudo"

	if req.privileged {
		warnStyle := lipgloss.NewStyle().Foreground(ColorYellow)
		if needsSudo {
			b.WriteString("\n" + warnStyle.Render("requires elevated privileges"))
			b.WriteString("\n\n")
			b.WriteString(m.passwordInput.View())
			b.WriteString("\n")
		} else {
			b.WriteString("\n" + warnStyle.Render("requires an elevated terminal"))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	yesStyle := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	noStyle := lipgloss.NewStyle().Foreground(ColorRed).Bold(true)

	switch m.removeFocus {
	case 2:
		yesStyle = yesStyle.Background(ColorGreen).Foreground(ColorBase)
		noStyle = noStyle.Foreground(ColorSubtext)
	case 3:
		yesStyle = yesStyle.Foreground(ColorSubtext)
		noStyle = noStyle.Background(ColorRed).Foreground(ColorBase)
	default:
		yesStyle = yesStyle.Foreground(ColorSubtext)
		noStyle = noStyle.Foreground(ColorSubtext)
	}

	b.WriteString("    " + yesStyle.Render("  Yes  ") + "   " + noStyle.Render("  No  "))

	return b.String()
}

func renderOpNotification(msg string, isErr, inFlight bool, opLabel, spinnerView string) string {
	icon := " ✓ "
	color := ColorGreen
	label := "DONE"
	if isErr {
		icon = " ✗ "
		color = ColorRed
		label = "FAIL"
	} else if inFlight {
		icon = " " + spinnerView + " "
		color = ColorCyan
		label = opLabel
	}
	badge := lipgloss.NewStyle().
		Background(color).
		Foreground(ColorBase).
		Bold(true).
		Render(" " + label + " ")
	msgStyle := lipgloss.NewStyle().Foreground(color)
	return badge + icon + msgStyle.Render(msg)
}

func gerund(op string) string {
	if op == "" {
		return "upgrading"
	}
	if strings.HasSuffix(op, "e") {
		return op[:len(op)-1] + "ing"
	}
	return op + "ing"
}

func pastTense(op string) string {
	if op == "" {
		return "upgraded"
	}
	if strings.HasSuffix(op, "e") {
		return op + "d"
	}
	return op + "ed"
}

func filterOut(items []string, exclude string) []string {
	var result []string
	for _, s := range items {
		if s != exclude {
			result = append(result, s)
		}
	}
	return result
}
