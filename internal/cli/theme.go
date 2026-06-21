package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/neur0map/glazepkg/internal/config"
	"github.com/neur0map/glazepkg/internal/manager"
)

func init() {
	subcommands["theme"] = runTheme
}

// runTheme lists the available color themes (with a live palette swatch) or
// sets the active one. The choice is shared with the TUI via the config file.
func runTheme(args []string, mgrs []manager.Manager, version string, stdout, stderr io.Writer, stdin io.Reader) int {
	fs := flag.NewFlagSet("theme", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonFlag := fs.Bool("json", false, "emit JSON envelope")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return ExitOK
		}
		return ExitErr
	}

	cfg := config.Load()
	themes := config.AllThemes()
	rest := fs.Args()

	if len(rest) == 0 {
		if *jsonFlag {
			ids := make([]string, 0, len(themes)+1)
			for _, t := range themes {
				ids = append(ids, t.ID)
			}
			ids = append(ids, "system")
			data := struct {
				Active string   `json:"active"`
				Themes []string `json:"themes"`
			}{Active: cfg.Appearance.Theme, Themes: ids}
			if err := writeEnvelope(stdout, version, data); err != nil {
				fmt.Fprintf(stderr, "error: encoding JSON: %v\n", err)
				return ExitErr
			}
			return ExitOK
		}
		st := newStyler()
		fmt.Fprintln(stdout, st.title("Themes"))
		for _, t := range themes {
			marker := " "
			suffix := ""
			if t.ID == cfg.Appearance.Theme {
				marker = st.ok("●")
				suffix = "  " + st.dim("(active)")
			}
			fmt.Fprintf(stdout, "  %s %s  %s%s\n", marker, padRight(t.ID, 18), themeSwatch(st, t), suffix)
		}
		return ExitOK
	}

	id := rest[0]
	if !validTheme(id, themes) {
		fmt.Fprintf(stderr, "error: unknown theme %q\n", id)
		ids := make([]string, len(themes))
		for i, t := range themes {
			ids[i] = t.ID
		}
		if sug := suggestNames(id, ids, 4, 3); len(sug) > 0 {
			fmt.Fprintf(stderr, "did you mean: %s\n", strings.Join(sug, ", "))
		}
		return ExitErr
	}

	cfg.Appearance.Theme = id
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(stderr, "error: saving theme: %v\n", err)
		return ExitErr
	}
	fmt.Fprintf(stdout, "%s theme set to %s\n", newStyler().ok("✓"), id)
	return ExitOK
}

func validTheme(id string, themes []config.Theme) bool {
	if id == "system" {
		return true
	}
	for _, t := range themes {
		if t.ID == id {
			return true
		}
	}
	return false
}

func themeSwatch(st *styler, t config.Theme) string {
	if !st.on {
		return ""
	}
	var b strings.Builder
	for _, hex := range []string{t.Palette.Blue, t.Palette.Cyan, t.Palette.Green, t.Palette.Yellow, t.Palette.Orange, t.Palette.Red, t.Palette.Purple} {
		b.WriteString(st.paint("██", hex, false))
	}
	return b.String()
}
