package cli

// Exit codes form the stable contract for callers (scripts, QML, CI gates).
// See docs/superpowers/specs/2026-05-14-headless-cli-phase1-design.md.
const (
	ExitOK        = 0 // success / yes / clean
	ExitErr       = 1 // generic error (bad flag, scan failed, IO/network)
	ExitNegative  = 2 // meaningful "no" (not installed, has updates, not found)
	ExitAmbiguous = 3 // install candidate available in multiple managers
	// 4 reserved for Phase 2 cache+network failure.
)
