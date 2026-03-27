package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type projectEntry struct {
	name    string
	prompts int
}

// renderLeft renders the project list panel.
func renderLeft(projects []projectEntry, selected int, focused bool, width, height int) string {
	style := stylePanel
	if focused {
		style = stylePanelFocused
	}

	inner := style.GetHorizontalFrameSize()
	contentW := width - inner

	title := styleSectionTitle.Render("Projects")
	lines := []string{title, styleMuted.Render(strings.Repeat("─", max(0, contentW)))}

	for i, p := range projects {
		name := p.name
		if name == "" {
			name = "(unknown)"
		}
		if len(name) > contentW-6 {
			name = name[:contentW-7] + "…"
		}

		label := fmt.Sprintf("%-*s %3dp", contentW-6, name, p.prompts)
		if i == selected {
			lines = append(lines, styleProjectSelected.Render(label))
		} else {
			lines = append(lines, styleProjectItem.Render(label))
		}
	}

	// Fill remaining height so the panel border stretches.
	panelInnerH := height - style.GetVerticalFrameSize()
	for len(lines) < panelInnerH {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return style.Width(width).Height(height).Render(content)
}

// renderBar renders a lipgloss-styled bar chart row.
func renderBar(label string, pct float64, labelW, barW int) string {
	filled := min(int(pct/100*float64(barW)), barW)
	bar := styleBar.Render(strings.Repeat("█", filled)) +
		styleBarEmpty.Render(strings.Repeat("░", barW-filled))

	return fmt.Sprintf(" %-*s %s %s",
		labelW,
		lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render(label),
		bar,
		styleMuted.Render(fmt.Sprintf("%.0f%%", pct)),
	)
}
