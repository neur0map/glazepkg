package cli

// Exit codes form the stable contract for callers (scripts, QML, CI gates).
// See docs/superpowers/specs/2026-05-14-headless-cli-phase1-design.md.
const (
	ExitOK       = 0 // success / yes / clean
	ExitErr      = 1 // generic error (bad flag, scan failed, IO/network)
	ExitNegative = 2 // meaningful "no" (not installed, has updates, not found)
	// 3 and 4 are reserved for Phase 2 (ambiguity, cache+network failure).
)
