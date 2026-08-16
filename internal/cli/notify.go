package cli

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
)

// haveCmd reports whether a binary is on PATH.
func haveCmd(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// notifyCmd builds the command that raises a desktop notification, using
// whatever the OS already ships. gpk stays a run-and-exit binary: no daemon, no
// run loop, no cgo — the user owns the scheduling (a launchd agent, a systemd
// timer, cron) and gpk just reports once when asked (#29).
//
// Returns nil when the platform has no notifier available, so the caller can
// fall back to plain output instead of failing.
func notifyCmd(title, body string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		if !haveCmd("osascript") {
			return nil
		}
		script := fmt.Sprintf("display notification %s with title %s",
			appleScriptString(body), appleScriptString(title))
		return exec.Command("osascript", "-e", script)
	case "windows":
		if !haveCmd("powershell") {
			return nil
		}
		return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", windowsToast(title, body))
	default:
		if !haveCmd("notify-send") {
			return nil
		}
		return exec.Command("notify-send", "--app-name=gpk", "--icon=system-software-update", title, body)
	}
}

// notify sends the notification and reports why it couldn't when it fails, so a
// scheduled `gpk outdated --notify` that silently stops working is noticeable.
func notify(title, body string, stderr io.Writer) error {
	cmd := notifyCmd(title, body)
	if cmd == nil {
		return fmt.Errorf("no desktop notifier found (%s)", notifierName())
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("%s: %s", exitReason(err), msg)
		}
		return fmt.Errorf("%s", exitReason(err))
	}
	return nil
}

func notifierName() string {
	switch runtime.GOOS {
	case "darwin":
		return "expected osascript"
	case "windows":
		return "expected powershell"
	default:
		return "expected notify-send from libnotify"
	}
}

// appleScriptString quotes a Go string as an AppleScript literal. Package names
// come from package managers, so backslashes and quotes have to be escaped or a
// crafted name could inject script.
func appleScriptString(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\n':
			b.WriteString("\\n")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// windowsToast builds the PowerShell one-liner that shows a balloon tip. Uses
// WinForms rather than the WinRT toast APIs because it works on any Windows with
// .NET Framework present and needs no AppUserModelID registration.
func windowsToast(title, body string) string {
	return strings.Join([]string{
		"[void][reflection.assembly]::LoadWithPartialName('System.Windows.Forms');",
		"$n = New-Object System.Windows.Forms.NotifyIcon;",
		"$n.Icon = [System.Drawing.SystemIcons]::Information;",
		"$n.BalloonTipTitle = " + powerShellString(title) + ";",
		"$n.BalloonTipText = " + powerShellString(body) + ";",
		"$n.Visible = $true;",
		"$n.ShowBalloonTip(10000);",
		"Start-Sleep -Seconds 10;",
		"$n.Dispose()",
	}, " ")
}

// powerShellString quotes a Go string as a PowerShell single-quoted literal,
// where the only escape is a doubled quote.
func powerShellString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// notifyBody summarizes the outdated set for a notification: a count plus the
// first few names, because a notification bubble has no room for 60 packages.
func notifyBody(entries []outdatedEntry) string {
	const maxNames = 5
	names := make([]string, 0, maxNames)
	for _, e := range entries {
		if len(names) == maxNames {
			break
		}
		names = append(names, e.Name)
	}
	body := strings.Join(names, ", ")
	if len(entries) > len(names) {
		body += fmt.Sprintf(" and %d more", len(entries)-len(names))
	}
	return body
}
