package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

func renderDiffView(diff model.Diff, since time.Time) string {
	var b strings.Builder

	ago := formatTimeAgo(since)
	b.WriteString(StyleNormal.Bold(true).Render(fmt.Sprintf("  Changes since last snapshot (%s)", since.Format("2006-01-02"))))
	b.WriteString(StyleDim.Render(fmt.Sprintf("  %s", ago)))
	b.WriteString(StyleNormal.Render("\n"))
	b.WriteString(StyleDim.Render("  " + strings.Repeat("─", 75)))
	b.WriteString(StyleNormal.Render("\n\n"))

	if len(diff.Added) == 0 && len(diff.Removed) == 0 && len(diff.Upgraded) == 0 {
		b.WriteString(StyleDim.Render("  No changes detected."))
		b.WriteString(StyleNormal.Render("\n"))
		return b.String()
	}

	for _, p := range diff.Added {
		b.WriteString(StyleAdded.Render(fmt.Sprintf("  + %-30s %-15s %s", p.Name, p.Version, p.Source)))
		b.WriteString(StyleNormal.Render("\n"))
	}

	for _, e := range diff.Upgraded {
		b.WriteString(StyleUpgrade.Render(fmt.Sprintf("  ↑ %-30s %-7s → %-7s %s", e.New.Name, e.Old.Version, e.New.Version, e.New.Source)))
		b.WriteString(StyleNormal.Render("\n"))
	}

	for _, p := range diff.Removed {
		b.WriteString(StyleRemoved.Render(fmt.Sprintf("  - %-30s %-15s %s", p.Name, p.Version, p.Source)))
		b.WriteString(StyleNormal.Render("\n"))
	}

	b.WriteString(StyleNormal.Render("\n"))
	summary := fmt.Sprintf("  +%d added    %d upgraded    %d removed",
		len(diff.Added), len(diff.Upgraded), len(diff.Removed))
	b.WriteString(StyleNormal.Render(summary))
	b.WriteString(StyleNormal.Render("\n"))

	return b.String()
}
