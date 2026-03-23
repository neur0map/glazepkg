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
	"github.com/neur0map/glazepkg/internal/theme"
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
	{"All", 0, 0},
	{"< 1 MB", 0, 1 << 20},
	{"1–10 MB", 1 << 20, 10 << 20},
	{"10–100 MB", 10 << 20, 100 << 20},
	{"> 100 MB", 100 << 20, 0},
	{"Has updates", -1, -1},
}

// ─── Message types ────────────────────────────────────────────────────────────

type updateAvailableMsg struct{ latest string }

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

type descriptionsDoneMsg struct{ descs map[string]string }
type updatesDoneMsg struct{ updates map[string]string }
type depsDoneMsg struct{ deps map[string][]string }
type exportDoneMsg struct {
	path string
	err  error
}

type upgradeResultMsg struct {
	pkg model.Package
	err error
}

type upgradeNotifClearMsg struct{}

type managerRescanMsg struct {
	source  model.Source
	pkgs    []model.Package
	updates map[string]string
	err     error
}

type pkgHelpMsg struct{ lines []string }

// themeChangedMsg is sent after ApplyTheme() so the bubbletea loop forces a
// full redraw with the new palette.
type themeChangedMsg struct{ t theme.Theme }

// SetVersionMsg is sent by the main process when the async version check completes.
type SetVersionMsg string

type upgradeRequest struct {
	pkg        model.Package
	cmd        *exec.Cmd
	cmdStr     string
	privileged bool
	password   string
}

// ─── Model ────────────────────────────────────────────────────────────────────

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
	sizeFilter  int

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
	confirmFocus      int // 0=password, 1=Yes, 2=No
	pendingUpgrade    *upgradeRequest
	passwordInput     textinput.Model
	upgradeInFlight   bool
	upgradingPkgName  string
	upgradeCancel     context.CancelFunc
	upgradeNotifMsg   string
	upgradeNotifErr   bool

	// Theme picker overlay
	showThemeMenu bool     // whether the theme overlay is open
	themeCursor   int      // cursor position within the theme list
	themeNames    []string // ordered list from theme.ListThemes()

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
	spinner    spinner.Model
	titleFrame int // incremented on every spinner tick; drives rainbow title animation
}

func NewModel(version string) Model {
	// Initialise the theme system; apply active theme to the UI palette.
	// Errors are non-fatal — the compile-time default (Tokyo Night) remains.
	if err := theme.Load(); err != nil {
		_ = err // logged inside theme.Load
	}
	ApplyTheme(theme.Active())

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

	sp := spinner.New()
	sp.Spinner = spinner.Points

	pi := textinput.New()
	pi.Placeholder = "password"
	pi.CharLimit = 128
	pi.Prompt = "  Password: "
	pi.EchoMode = textinput.EchoPassword
	pi.EchoCharacter = '•'

	m := Model{
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
		themeNames:    theme.ListThemes(),
		themeCursor:   theme.ActiveIndex(),
	}
	m.syncThemeStyles()
	return m
}

func (m *Model) syncThemeStyles() {
	m.filterInput.PromptStyle = StyleFilterPrompt.Copy().Background(ColorBase)
	m.filterInput.TextStyle = StyleFilterText.Copy().Background(ColorBase)
	m.filterInput.PlaceholderStyle = StyleDim.Copy().Background(ColorBase)

	m.descInput.PromptStyle = StyleDetailKey.Copy().Background(ColorBase)
	m.descInput.TextStyle = StyleDetailVal.Copy().Background(ColorBase)
	m.descInput.PlaceholderStyle = StyleDim.Copy().Background(ColorBase)

	m.passwordInput.PromptStyle = StyleDim.Copy().Background(ColorOverlay)
	m.passwordInput.TextStyle = StyleNormal.Copy().Background(ColorOverlay)
	m.passwordInput.PlaceholderStyle = StyleDim.Copy().Background(ColorOverlay)

	m.spinner.Style = lipgloss.NewStyle().Foreground(ColorAccent)
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
		if err != nil {
			return nil
		}
		// Normalize: strip leading 'v' from both sides before comparing.
		normalLatest := strings.TrimPrefix(latest, "v")
		normalCurrent := strings.TrimPrefix(currentVersion, "v")
		if normalLatest == normalCurrent {
			return nil
		}
		return updateAvailableMsg{latest: normalLatest}
	}
}

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
	sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })
	manager.SaveScanCache(all)
	return scanDoneMsg{pkgs: all}
}

func forceRescan() tea.Msg { return freshScan() }

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
		return pkgHelpMsg{lines: tryPkgHelp(name)}
	}
}

func tryPkgHelp(name string) []string {
	flags := [][]string{{name, "--help"}, {name, "-h"}, {name, "help"}}
	for _, args := range flags {
		cmd := exec.Command(args[0], args[1:]...)
		out, _ := cmd.CombinedOutput()
		if len(out) > 0 {
			return parseHelpOutput(string(out))
		}
	}
	return []string{"No help available for " + name}
}

func parseHelpOutput(raw string) []string {
	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		lines = append(lines, strings.ReplaceAll(line, "\t", "    "))
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > 500 {
		lines = append(lines[:500], "... (truncated)")
	}
	return lines
}

func fetchDependencies(pkgs []model.Package, cache *manager.DepsCache) tea.Cmd {
	return func() tea.Msg {
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
		m.confirmFocus = 0
		m.passwordInput.Focus()
		return textinput.Blink
	}
	m.confirmFocus = 1
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

// ─── Update ───────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case spinner.TickMsg:
		// Always update the spinner model so the rainbow title animation runs
		// continuously.  The spinner glyph itself is only rendered in the UI
		// when there is active work (scanning / loading / upgradeInFlight).
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.titleFrame++
		return m, cmd

	// themeChangedMsg is dispatched after ApplyTheme() to force a redraw.
	// The actual palette swap already happened synchronously in handleKey; this
	// message just ensures bubbletea triggers a View() call.
	case themeChangedMsg:
		ApplyTheme(msg.t)
		m.syncThemeStyles()
		return m, nil

	case upgradeResultMsg:
		m.upgradeInFlight = false
		m.upgradingPkgName = ""
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
			tea.Tick(5*time.Second, func(time.Time) tea.Msg { return upgradeNotifClearMsg{} }),
		)

	case upgradeNotifClearMsg:
		m.upgradeNotifMsg = ""
		return m, nil

	case managerRescanMsg:
		if msg.err != nil {
			m.statusMsg = "refresh error: " + msg.err.Error()
			return m, nil
		}
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
		sort.Slice(next, func(i, j int) bool { return next[i].Name < next[j].Name })
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
		m.loadingDescs = true
		return m, fetchDescriptions(m.allPkgs, m.descCache, m.userNotes)

	case descriptionsDoneMsg:
		m.loadingDescs = false
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
		current := strings.TrimPrefix(m.version, "v")
		m.updateBanner = fmt.Sprintf("↑ v%s → v%s available — run `gpk update`", current, msg.latest)
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

// ─── Key handling ─────────────────────────────────────────────────────────────

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	m.statusMsg = ""

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

	// Theme overlay intercepts all keys.
	if m.showThemeMenu {
		return m.handleThemeMenuKey(key)
	}

	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

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

// handleThemeMenuKey handles input while the theme picker overlay is open.
//
// Navigation:  j/↓ and k/↑ move the cursor.
// Selection:   Enter applies the highlighted theme immediately.
// Cycling:     t advances to the next theme without closing the overlay, so the
//
//	user can tap t repeatedly to live-preview themes.
//
// Dismiss:     Esc / q closes the overlay without changing theme.
func (m *Model) handleThemeMenuKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.showThemeMenu = false

	case "j", "down":
		if m.themeCursor < len(m.themeNames)-1 {
			m.themeCursor++
		}

	case "k", "up":
		if m.themeCursor > 0 {
			m.themeCursor--
		}

	case "g", "home":
		m.themeCursor = 0

	case "G", "end":
		m.themeCursor = len(m.themeNames) - 1

	case "ctrl+d", "pgdown":
		m.themeCursor += 5
		if m.themeCursor >= len(m.themeNames) {
			m.themeCursor = len(m.themeNames) - 1
		}

	case "ctrl+u", "pgup":
		m.themeCursor -= 5
		if m.themeCursor < 0 {
			m.themeCursor = 0
		}

	case "enter":
		// Apply and persist the highlighted theme.
		return m.applyThemeByIndex(m.themeCursor, true)

	case "t":
		// Advance to the next theme (live-preview cycle) without closing.
		m.themeCursor = (m.themeCursor + 1) % len(m.themeNames)
		return m.applyThemeByIndex(m.themeCursor, false)
	}
	return m, nil
}

// applyThemeByIndex applies the theme at idx, optionally closing the menu.
func (m *Model) applyThemeByIndex(idx int, closeMenu bool) (tea.Model, tea.Cmd) {
	if idx < 0 || idx >= len(m.themeNames) {
		return m, nil
	}
	name := m.themeNames[idx]
	t, err := theme.SetActive(name)
	if err != nil {
		m.statusMsg = "theme error: " + err.Error()
		return m, nil
	}
	// Apply synchronously — rebuildStyles() runs immediately so that the very
	// next View() call uses the new colors.
	ApplyTheme(t)
	m.syncThemeStyles()
	m.statusMsg = fmt.Sprintf("theme: %s", t.Name)
	if closeMenu {
		m.showThemeMenu = false
	}
	// Dispatch a themeChangedMsg to guarantee bubbletea triggers View().
	return m, func() tea.Msg { return themeChangedMsg{t: t} }
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
	case "t":
		// Open the theme picker overlay, positioning cursor at active theme.
		m.showThemeMenu = true
		m.themeNames = theme.ListThemes()
		m.themeCursor = theme.ActiveIndex()
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
	case "u":
		if _, ok := m.selectedPackage(); ok {
			// Upgrade from list view is unsupported (no detail loaded).
			// Direct user to Enter → u in detail view for safety.
			m.statusMsg = "press Enter for detail view, then u to upgrade"
		}
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
	case "t":
		// Theme picker is also available from detail view.
		m.showThemeMenu = true
		m.themeNames = theme.ListThemes()
		m.themeCursor = theme.ActiveIndex()
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
	case "t":
		m.showThemeMenu = true
		m.themeNames = theme.ListThemes()
		m.themeCursor = theme.ActiveIndex()
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

	var tabFiltered []model.Package
	for _, p := range m.allPkgs {
		if source != "" && string(p.Source) != source {
			continue
		}
		if source == "" && depSources[p.Source] {
			continue
		}
		tabFiltered = append(tabFiltered, p)
	}

	if m.sizeFilter > 0 {
		sf := sizeFilters[m.sizeFilter]
		var matched, unknown []model.Package
		for _, p := range tabFiltered {
			if sf.MinBytes == -1 {
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
		sort.Slice(matched, func(i, j int) bool {
			return matched[i].SizeBytes > matched[j].SizeBytes
		})
		tabFiltered = append(matched, unknown...)
	}

	m.filteredPkgs = fuzzyFilter(tabFiltered, query)
	if m.cursor >= len(m.filteredPkgs) {
		m.cursor = max(0, len(m.filteredPkgs)-1)
	}
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	// Overlay priority: upgrade confirm > theme menu > help > export > deps > pkg-help.
	if m.confirmingUpgrade {
		return m.renderBase(m.renderUpgradeConfirmOverlay())
	}
	if m.showThemeMenu {
		return m.renderBase(m.renderThemeOverlay())
	}
	if m.showHelp {
		return m.renderBase(renderHelpOverlay(m.width, m.height))
	}
	if m.showExport {
		return m.renderBase(renderExportOverlay(m.exportCursor, m.width, m.height))
	}
	if m.showDeps {
		return m.renderBase(renderDepsOverlay(m.detailPkg, m.depsCursor, m.width, m.height))
	}
	if m.showPkgHelp {
		return m.renderBase(renderPkgHelpOverlay(m.detailPkg.Name, m.pkgHelpLines, m.pkgHelpScroll, m.width, m.height))
	}

	var b strings.Builder

	// ── Centered rainbow title ───────────────────────────────────────────────
	b.WriteString(m.renderTitleLine())
	b.WriteString(StyleNormal.Render("\n\n"))

	switch m.view {
	case viewList:
		m.renderListView(&b)
	case viewDetail:
		b.WriteString(renderDetail(m.detailPkg, m.editingDesc, m.descInput.View()))
	case viewDiff:
		b.WriteString(renderDiffView(m.currentDiff, m.diffSince))
	}

	if m.upgradeNotifMsg != "" {
		icon := StyleNormal.Render(" ✓ ")
		color := ColorGreen
		label := "DONE"
		if m.upgradeNotifErr {
			icon = StyleNormal.Render(" ✗ ")
			color = ColorRed
			label = "FAIL"
		} else if m.upgradeInFlight {
			icon = StyleNormal.Render(" " + m.spinner.View() + " ")
			color = ColorCyan
			label = "UPGRADE"
		}
		badge := lipgloss.NewStyle().
			Background(color).
			Foreground(badgeForeground(color)).
			Bold(true).
			Render(" " + label + " ")
		msgStyle := lipgloss.NewStyle().Foreground(color).Background(ColorBase)
		b.WriteString(StyleNormal.Render("\n  ") + badge + icon + msgStyle.Render(m.upgradeNotifMsg))
	}

	b.WriteString(StyleNormal.Render("\n"))
	b.WriteString(StyleDim.Render("  " + strings.Repeat("─", min(m.width-4, 120))))
	b.WriteString(StyleNormal.Render("\n"))
	b.WriteString(m.renderStatusBar())

	return m.renderBase(b.String())
}

// renderThemeOverlay draws the theme picker as a centred modal.
// The list is capped at maxVisible rows and scrolls to keep the cursor in view.
func (m Model) renderThemeOverlay() string {
	const maxVisible = 12

	names := m.themeNames
	if len(names) == 0 {
		return ""
	}

	activeThemeName := theme.Active().Name

	// Compute scroll window.
	start := 0
	if m.themeCursor >= maxVisible {
		start = m.themeCursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(names) {
		end = len(names)
	}

	var b strings.Builder

	// Header
	b.WriteString(StyleOverlayTitle.Render("  Themes"))
	b.WriteString(StyleOverlayBase.Render("\n"))
	b.WriteString(StyleOverlayBase.Copy().Foreground(ColorSubtext).Render("  " + strings.Repeat("─", 34)))
	b.WriteString(StyleOverlayBase.Render("\n\n"))

	// Scroll indicator (top)
	if start > 0 {
		b.WriteString(StyleOverlayBase.Copy().Foreground(ColorSubtext).Render(fmt.Sprintf("  ↑ %d more above", start)))
		b.WriteString(StyleOverlayBase.Render("\n"))
	}

	for i := start; i < end; i++ {
		name := names[i]
		t, _ := theme.GetTheme(name)

		// Build the type badge ("dark" / "light" / "").
		typeBadge := ""
		if t.Type != "" {
			badgeColor := ColorSubtext
			if t.Type == "light" {
				badgeColor = ColorYellow
			}
			typeBadge = StyleOverlayBase.Render(" ") + lipgloss.NewStyle().
				Foreground(badgeColor).
				Background(ColorOverlay).
				Render("["+t.Type+"]")
		}

		isActive := name == activeThemeName
		isCursor := i == m.themeCursor

		var row string
		switch {
		case isCursor && isActive:
			row = StyleThemeActive.Render("▶ "+name) + typeBadge
		case isCursor:
			row = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Background(ColorOverlay).
				Padding(0, 1).
				Bold(true).
				Render("▶ "+name) + typeBadge
		case isActive:
			row = lipgloss.NewStyle().
				Foreground(ColorGreen).
				Background(ColorOverlay).
				Padding(0, 1).
				Render("✓ "+name) + typeBadge
		default:
			row = StyleThemeItem.Render("  "+name) + typeBadge
		}
		b.WriteString(row + StyleOverlayBase.Render("\n"))
	}

	// Scroll indicator (bottom)
	remaining := len(names) - end
	if remaining > 0 {
		b.WriteString(StyleOverlayBase.Copy().Foreground(ColorSubtext).Render(fmt.Sprintf("  ↓ %d more below", remaining)))
		b.WriteString(StyleOverlayBase.Render("\n"))
	}

	b.WriteString(StyleOverlayBase.Render("\n"))
	b.WriteString(StyleOverlayBase.Copy().Foreground(ColorSubtext).Render("  j/k navigate · Enter select · t cycle · Esc close"))

	content := b.String()

	// Width: wide enough for the longest theme name plus decoration.
	overlayWidth := 42
	for _, name := range names {
		if w := len(name) + 8; w > overlayWidth {
			overlayWidth = w
		}
	}
	if overlayWidth > m.width-4 {
		overlayWidth = m.width - 4
	}

	overlayHeight := min(end-start, maxVisible) + 8

	overlay := StyleOverlay.
		Width(overlayWidth).
		Height(overlayHeight).
		Render(content)

	return placeOverlay(m.width, m.height, overlay)
}

// ─── Title rendering ──────────────────────────────────────────────────────────

// rainbowPalette is the ordered color sequence used for the title gradient.
// Cool blues and teals lead into warm amber and rose, cycling back — a palette
// that reads as "premium developer tool" rather than a toy.
var rainbowPalette = []lipgloss.Color{
	"#7aa2f7", // blue
	"#7dcfff", // sky
	"#73daca", // teal
	"#9ece6a", // green
	"#e0af68", // amber
	"#ff9e64", // orange
	"#f7768e", // rose
	"#bb9af7", // purple
}

// blendColors interpolates between two colors based on the ratio (0.0 to 1.0).
func blendColors(c1, c2 lipgloss.Color, ratio float64) lipgloss.Color {
	r1, g1, b1, _ := parseHexColor(string(c1))
	r2, g2, b2, _ := parseHexColor(string(c2))
	r := r1 + (r2-r1)*ratio
	g := g1 + (g2-g1)*ratio
	b := b1 + (b2-b1)*ratio
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", int(r*255), int(g*255), int(b*255)))
}

// renderRainbowTitle renders each rune in `text` using a smooth cycling gradient.
// The colors are blended frame-by-frame for a fluid animation.
func renderRainbowTitle(text string, frame int) string {
	n := len(rainbowPalette)
	// Speed factor: how fast the colors cycle (0.25 is approx 1 cycle per 32 frames)
	const speed = 0.25
	// Spread factor: how fast colors change across the text (1.0 = 1 palette step per char)
	const spread = 1.0

	var sb strings.Builder
	for i, ch := range text {
		// Calculate position in the palette
		t := float64(frame)*speed + float64(i)*spread

		// Ensure positive index
		val := t
		if val < 0 {
			val = -val
		}

		idx := int(val) % n
		nextIdx := (idx + 1) % n
		ratio := val - float64(int(val))

		c := blendColors(rainbowPalette[idx], rainbowPalette[nextIdx], ratio)
		sb.WriteString(
			lipgloss.NewStyle().
				Foreground(c).
				Background(ColorBase). // Blend with theme background
				Bold(true).
				Render(string(ch)),
		)
	}
	return sb.String()
}

// renderTitleLine returns a single terminal-width line containing the centered
// rainbow "GlazePKG" heading and version.
func (m Model) renderTitleLine() string {
	const leftDeco = "✦ "
	const rightDeco = " ✦"
	const baseText = "GlazePKG"

	// Stars match the accent color and theme background
	starStyle := lipgloss.NewStyle().Foreground(ColorAccent).Background(ColorBase).Bold(true)
	left := starStyle.Render(leftDeco)
	right := starStyle.Render(rightDeco)

	// Rainbow title
	rainbow := renderRainbowTitle(baseText, m.titleFrame)

	// Version string (dimmed, theme background)
	verText := " v" + m.version
	version := StyleDim.Copy().Background(ColorBase).Render(verText)

	// Combine: ✦ [Rainbow] v1.2.0 ✦
	content := left + rainbow + version + right

	// Calculate centering based on visual width (stripping ANSI)
	visualWidth := lipgloss.Width(content)

	var line string
	if m.width > visualWidth {
		pad := (m.width - visualWidth) / 2
		leftPad := StyleNormal.Render(strings.Repeat(" ", pad))
		line = leftPad + content
	} else {
		line = content
	}

	if m.updateBanner == "" {
		return line
	}

	// Update banner: centered on the line below the title.
	bannerRendered := StyleUpdateBanner.Render(m.updateBanner)
	bannerVisualWidth := len([]rune(m.updateBanner)) // banner is ASCII-safe
	var bannerLine string
	if m.width > bannerVisualWidth {
		pad := (m.width - bannerVisualWidth) / 2
		bannerLine = StyleNormal.Render(strings.Repeat(" ", pad)) + bannerRendered
	} else {
		bannerLine = bannerRendered
	}
	return line + StyleNormal.Render("\n") + bannerLine
}

func (m Model) renderListView(b *strings.Builder) {
	if len(m.tabs) > 0 {
		b.WriteString(StyleNormal.Render("  "))
		b.WriteString(renderTabs(m.tabs, m.activeTab))
		b.WriteString(StyleNormal.Render("\n"))
		b.WriteString(StyleDim.Render("  " + strings.Repeat("─", min(m.width-4, 120))))
		b.WriteString(StyleNormal.Render("\n\n"))
	}

	if m.filtering {
		b.WriteString(StyleNormal.Render("  "))
		b.WriteString(m.filterInput.View())
		b.WriteString(StyleNormal.Render("\n\n"))
	} else if m.filterInput.Value() != "" {
		b.WriteString(StyleNormal.Render("  "))
		b.WriteString(StyleFilterPrompt.Render("/ "))
		b.WriteString(StyleFilterText.Render(m.filterInput.Value()))
		b.WriteString(StyleNormal.Render("\n\n"))
	}

	if m.scanning {
		// Rotate through contextual messages to communicate progress.
		scanMsgs := []string{
			"Scanning package managers",
			"Discovering installed packages",
			"Collecting version info",
			"Resolving package sources",
		}
		scanMsg := scanMsgs[(m.titleFrame/10)%len(scanMsgs)]
		b.WriteString(StyleNormal.Render("  "))
		b.WriteString(lipgloss.NewStyle().Foreground(ColorAccent).Render(m.spinner.View()))
		b.WriteString(StyleDim.Render("  " + scanMsg + "..."))
		b.WriteString(StyleNormal.Render("\n"))
		return
	}

	listHeight := m.height - 12
	if listHeight < 5 {
		listHeight = 5
	}
	showSize := m.sizeFilter > 0 && sizeFilters[m.sizeFilter].MinBytes != -1
	b.WriteString(renderPackageTable(m.filteredPkgs, m.cursor, listHeight, m.width, showSize, m.upgradingPkgName))

	if m.loadingDescs {
		loadDescMsgs := []string{
			"Fetching package descriptions",
			"Querying package databases",
			"Pulling metadata",
			"Enriching package data",
		}
		msg := loadDescMsgs[(m.titleFrame/8)%len(loadDescMsgs)]
		b.WriteString(StyleNormal.Render("\n  "))
		b.WriteString(lipgloss.NewStyle().Foreground(ColorCyan).Render(m.spinner.View()))
		b.WriteString(StyleDim.Render("  " + msg + "..."))
	} else if m.loadingUpdates || m.loadingDeps {
		loadMsgs := []string{
			"Checking for updates",
			"Resolving dependency graph",
			"Querying upstream registries",
			"Comparing version vectors",
		}
		msg := loadMsgs[(m.titleFrame/8)%len(loadMsgs)]
		b.WriteString(StyleNormal.Render("\n  "))
		b.WriteString(lipgloss.NewStyle().Foreground(ColorPurple).Render(m.spinner.View()))
		b.WriteString(StyleDim.Render("  " + msg + "..."))
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

	keyStyle := lipgloss.NewStyle().Foreground(ColorCyan).Background(ColorBase).Bold(true)
	sepStyle := lipgloss.NewStyle().Foreground(ColorSubtext).Background(ColorBase)
	descStyle := lipgloss.NewStyle().Foreground(ColorText).Background(ColorBase)
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
			{"d", "diff"}, {"e", "export"}, {"t", "theme"}, {"?", "help"}, {"q", "quit"},
		}
		bar := formatBinds(binds)
		if m.sizeFilter > 0 {
			filterStyle := lipgloss.NewStyle().Foreground(ColorYellow).Background(ColorBase).Bold(true)
			bar = filterStyle.Render("["+sizeFilters[m.sizeFilter].Label+"]") + sep + bar
		}
		return StyleNormal.Render(" ") + bar
	case viewDetail:
		if m.editingDesc {
			return StyleNormal.Render(" ") + formatBinds([]struct{ key, desc string }{
				{"enter", "save"}, {"esc", "cancel"},
			})
		}
		if m.showPkgHelp {
			return StyleNormal.Render(" ") + formatBinds([]struct{ key, desc string }{
				{"j/k", "scroll"}, {"pgdn/pgup", "page"}, {"esc", "close"},
			})
		}
		if m.showDeps {
			return StyleNormal.Render(" ") + formatBinds([]struct{ key, desc string }{
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
		binds = append(binds,
			struct{ key, desc string }{"h", "help/usage"},
			struct{ key, desc string }{"t", "theme"},
			struct{ key, desc string }{"esc", "back"},
			struct{ key, desc string }{"q", "quit"},
		)
		return StyleNormal.Render(" ") + formatBinds(binds)
	case viewDiff:
		return StyleNormal.Render(" ") + formatBinds([]struct{ key, desc string }{
			{"t", "theme"}, {"esc", "back"}, {"q", "quit"},
		})
	}
	return ""
}

func (m Model) renderBase(content string) string {
	// Style the entire content block by lines to ensure the background color
	// is correctly applied even if sub-components have internal unstyled spaces.
	lines := strings.Split(content, "\n")
	var styledLines []string
	for _, line := range lines {
		styledLines = append(styledLines, StyleNormal.Copy().Width(m.width).Render(line))
	}
	styledContent := strings.Join(styledLines, "\n")

	return lipgloss.Place(m.width, m.height,
		lipgloss.Left, lipgloss.Top,
		styledContent,
		lipgloss.WithWhitespaceBackground(ColorBase),
	)
}

// ─── Upgrade confirm overlay ──────────────────────────────────────────────────

func (m *Model) runUpgradeRequest(req upgradeRequest) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.upgradeCancel = cancel
	ctxCmd := exec.CommandContext(ctx, req.cmd.Args[0], req.cmd.Args[1:]...)
	ctxCmd.Dir = req.cmd.Dir
	ctxCmd.Env = req.cmd.Env
	return func() tea.Msg {
		defer cancel() //nolint:errcheck
		// Detect commands that were tagged as needing elevation but cannot be
		// elevated automatically (process is not admin, gsudo is not installed).
		// Fail immediately with a clear, actionable message rather than letting
		// the child process crash deep inside its own file-system operations
		// (e.g. the choco .chocolateyPending "Access is denied" error).
		for _, e := range ctxCmd.Env {
			if e == "GLAZEPKG_NEEDS_ELEVATION=1" {
				return upgradeResultMsg{
					pkg: req.pkg,
					err: fmt.Errorf("administrator privileges required — re-run GlazePKG from an elevated terminal, or install gsudo: choco install gsudo"),
				}
			}
		}
		// Run any manager-specific pre-upgrade preparation (e.g. removing
		// stale Chocolatey .chocolateyPending markers).  This is called inside
		// the goroutine so it executes after elevation is confirmed but before
		// the command starts, guaranteeing a clean environment for every run.
		if mgr := manager.BySource(req.pkg.Source); mgr != nil {
			if prep, ok := mgr.(manager.PreUpgrader); ok {
				if err := prep.PrepareUpgrade(req.pkg.Name); err != nil {
					return upgradeResultMsg{pkg: req.pkg, err: err}
				}
			}
		}
		if req.password != "" {
			ctxCmd.Stdin = strings.NewReader(req.password + "\n")
			req.password = ""
		}
		out, err := ctxCmd.CombinedOutput()
		if err != nil {
			if msg := extractErrorLines(string(out)); msg != "" {
				err = fmt.Errorf("%w: %s", err, msg)
			}
		}
		return upgradeResultMsg{pkg: req.pkg, err: err}
	}
}

func extractErrorLines(raw string) string {
	lines := strings.Split(raw, "\n")
	var errLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
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
	m.upgradingPkgName = req.pkg.Name
	m.upgradeNotifMsg = fmt.Sprintf("upgrading %s...", req.pkg.Name)
	m.upgradeNotifErr = false
	return tea.Batch(m.spinner.Tick, m.runUpgradeRequest(req))
}

func (m *Model) needsSudoPassword() bool {
	return m.pendingUpgrade != nil &&
		len(m.pendingUpgrade.cmd.Args) > 0 &&
		m.pendingUpgrade.cmd.Args[0] == "sudo"
}

func (m *Model) handleUpgradeConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	hasPwField := m.needsSudoPassword()

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
		if m.confirmFocus == 1 {
			if hasPwField && m.passwordInput.Value() == "" {
				m.confirmFocus = 0
				m.passwordInput.Focus()
				return m, textinput.Blink
			}
			return m, m.executePendingUpgrade()
		}
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
	b.WriteString(StyleOverlayBase.Render("\n"))
	b.WriteString(StyleOverlayBase.Copy().Foreground(ColorSubtext).Render("  " + strings.Repeat("─", 40)))
	b.WriteString(StyleOverlayBase.Render("\n\n"))
	b.WriteString(StyleOverlayBase.Render(fmt.Sprintf("  Upgrade %s (%s)?", req.pkg.Name, req.pkg.Source)))
	b.WriteString(StyleOverlayBase.Render("\n\n"))
	b.WriteString(StyleOverlayBase.Copy().Foreground(ColorSubtext).Render("  command:"))
	b.WriteString(StyleOverlayBase.Render("\n"))
	cmdStyle := lipgloss.NewStyle().Foreground(ColorCyan).Background(ColorOverlay)
	b.WriteString(StyleOverlayBase.Render("  ") + cmdStyle.Render(req.cmdStr))
	b.WriteString(StyleOverlayBase.Render("\n"))

	overlayHeight := 11
	needsSudo := len(req.cmd.Args) > 0 && req.cmd.Args[0] == "sudo"

	if req.privileged {
		warnStyle := lipgloss.NewStyle().Foreground(ColorYellow).Background(ColorOverlay)
		if needsSudo {
			b.WriteString(StyleOverlayBase.Render("\n") + StyleOverlayBase.Render("  ") + warnStyle.Render("requires elevated privileges"))
			b.WriteString(StyleOverlayBase.Render("\n\n"))
			b.WriteString(m.passwordInput.View())
			b.WriteString(StyleOverlayBase.Render("\n"))
			overlayHeight = 16
		} else {
			b.WriteString(StyleOverlayBase.Render("\n") + StyleOverlayBase.Render("  ") + warnStyle.Render("requires an elevated terminal"))
			b.WriteString(StyleOverlayBase.Render("\n"))
			overlayHeight = 13
		}
	}

	b.WriteString(StyleOverlayBase.Render("\n"))

	yesStyle := lipgloss.NewStyle().Foreground(ColorGreen).Background(ColorOverlay).Bold(true)
	noStyle := lipgloss.NewStyle().Foreground(ColorRed).Background(ColorOverlay).Bold(true)

	switch m.confirmFocus {
	case 1:
		yesStyle = yesStyle.Background(ColorGreen).Foreground(badgeForeground(ColorGreen))
		noStyle = noStyle.Foreground(ColorSubtext)
	case 2:
		yesStyle = yesStyle.Foreground(ColorSubtext)
		noStyle = noStyle.Background(ColorRed).Foreground(badgeForeground(ColorRed))
	default:
		yesStyle = yesStyle.Foreground(ColorSubtext)
		noStyle = noStyle.Foreground(ColorSubtext)
	}

	b.WriteString(StyleOverlayBase.Render("      ") + yesStyle.Render("  Yes  ") + StyleOverlayBase.Render("   ") + noStyle.Render("  No  "))

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
