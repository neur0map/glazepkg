package cli

import (
	"runtime"
	"strings"
	"testing"
)

func TestNotifyBodySummarizes(t *testing.T) {
	var entries []outdatedEntry
	for _, n := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		entries = append(entries, outdatedEntry{Name: n})
	}
	got := notifyBody(entries)
	if got != "a, b, c, d, e and 2 more" {
		t.Errorf("body = %q", got)
	}
	if got := notifyBody(entries[:2]); got != "a, b" {
		t.Errorf("short body = %q, want no overflow suffix", got)
	}
}

// Package names come from package managers, so a name containing a quote must
// not be able to close the AppleScript literal and inject script.
func TestAppleScriptStringEscapes(t *testing.T) {
	cases := map[string]string{
		`plain`:            `"plain"`,
		`say "hi"`:         `"say \"hi\""`,
		`back\slash`:       `"back\\slash"`,
		"two\nlines":       `"two\nlines"`,
		`" & do shell scr`: `"\" & do shell scr"`,
	}
	for in, want := range cases {
		if got := appleScriptString(in); got != want {
			t.Errorf("appleScriptString(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestPowerShellStringEscapes(t *testing.T) {
	if got := powerShellString("it's"); got != "'it''s'" {
		t.Errorf("powerShellString = %s", got)
	}
}

// The per-OS command must carry the title and body through unmangled. Only the
// host platform's branch is reachable, so assert on that one.
func TestNotifyCmdCarriesMessage(t *testing.T) {
	cmd := notifyCmd("3 package updates available", "git, vim, curl")
	if cmd == nil {
		t.Skipf("no notifier on %s", runtime.GOOS)
	}
	joined := strings.Join(cmd.Args, " ")
	for _, want := range []string{"3 package updates available", "git, vim, curl"} {
		if !strings.Contains(joined, want) {
			t.Errorf("command %v missing %q", cmd.Args, want)
		}
	}
	switch runtime.GOOS {
	case "darwin":
		if cmd.Args[0] != "osascript" || !strings.Contains(joined, "display notification") {
			t.Errorf("darwin command = %v", cmd.Args)
		}
	case "windows":
		if cmd.Args[0] != "powershell" || !strings.Contains(joined, "ShowBalloonTip") {
			t.Errorf("windows command = %v", cmd.Args)
		}
	default:
		if cmd.Args[0] != "notify-send" {
			t.Errorf("unix command = %v", cmd.Args)
		}
	}
}

func TestWindowsToastQuotesBothFields(t *testing.T) {
	script := windowsToast("Title's", "Body's")
	if !strings.Contains(script, "'Title''s'") || !strings.Contains(script, "'Body''s'") {
		t.Errorf("toast script did not quote fields: %s", script)
	}
}
