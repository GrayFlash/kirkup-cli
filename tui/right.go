package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/GrayFlash/kirkup-cli/retro"
)

// renderRight renders the summary panel.
func renderRight(summary *retro.Summary, focused bool, width, height int) string {
	style := stylePanel
	if focused {
		style = stylePanelFocused
	}

	if summary == nil || summary.TotalPrompts == 0 {
		empty := styleMuted.Render("no data for this period")
		return style.Width(width).Height(height).Render(empty)
	}

	inner := style.GetHorizontalFrameSize()
	contentW := width - inner

	var sections []string

	// -- Stats row --
	stats := fmt.Sprintf("%s prompts  ·  %s sessions  ·  %s",
		styleStat.Render(fmt.Sprintf("%d", summary.TotalPrompts)),
		styleStat.Render(fmt.Sprintf("%d", summary.TotalSessions)),
		styleStat.Render(fmtDuration(summary.TotalEstTime)),
	)
	sections = append(sections, stats, "")

	barW := 20

	// -- By Category --
	if len(summary.Categories) > 0 {
		sections = append(sections, styleSectionTitle.Render("By Category"))
		sections = append(sections, styleMuted.Render(strings.Repeat("─", max(0, contentW-2))))
		labelW := maxLabelLen(categoryLabels(summary))
		for _, c := range summary.Categories {
			sections = append(sections, renderBar(c.Category, c.Percent, labelW, barW))
		}
		sections = append(sections, "")
	}

	// -- By Agent --
	if len(summary.Agents) > 0 {
		sections = append(sections, styleSectionTitle.Render("By Agent"))
		sections = append(sections, styleMuted.Render(strings.Repeat("─", max(0, contentW-2))))
		labelW := maxLabelLen(agentLabels(summary))
		for _, a := range summary.Agents {
			sections = append(sections, renderBar(a.Agent, a.Percent, labelW, barW))
		}
		sections = append(sections, "")
	}

	// -- Daily Activity --
	if len(summary.Daily) > 0 {
		sections = append(sections, styleSectionTitle.Render("Daily Activity"))
		sections = append(sections, styleMuted.Render(strings.Repeat("─", max(0, contentW-2))))
		maxP := 0
		for _, d := range summary.Daily {
			if d.Prompts > maxP {
				maxP = d.Prompts
			}
		}
		for _, d := range summary.Daily {
			pct := 0.0
			if maxP > 0 {
				pct = float64(d.Prompts) / float64(maxP) * 100
			}
			label := d.Date.Format("Mon Jan 2")
			count := styleMuted.Render(fmt.Sprintf("%d prompts", d.Prompts))
			
			filled := int(pct / 100 * float64(barW))
			if filled < 0 {
				filled = 0
			}
			if filled > barW {
				filled = barW
			}
			safeBarW := barW
			if safeBarW < 0 {
				safeBarW = 0
				filled = 0
			}

			bar := styleBar.Render(strings.Repeat("█", filled)) +
				styleBarEmpty.Render(strings.Repeat("░", safeBarW-filled))
			sections = append(sections, fmt.Sprintf(" %-12s %s  %s",
				lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render(label),
				bar,
				count,
			))
		}
		sections = append(sections, "")
	}

	// -- Projects breakdown --
	if len(summary.Projects) > 0 {
		sections = append(sections, styleSectionTitle.Render("Projects"))
		sections = append(sections, styleMuted.Render(strings.Repeat("─", max(0, contentW-2))))
		for _, p := range summary.Projects {
			name := p.Name
			if name == "" {
				name = "(unknown)"
			}
			sections = append(sections, fmt.Sprintf(" %-24s %s sessions  %s prompts  %s",
				lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render(name),
				styleStat.Render(fmt.Sprintf("%d", p.Sessions)),
				styleStat.Render(fmt.Sprintf("%d", p.Prompts)),
				styleMuted.Render(fmtDuration(p.EstTime)),
			))
			for _, b := range p.Branches {
				sections = append(sections, styleMuted.Render(fmt.Sprintf("   %-22s %d prompts", b.Name, b.Prompts)))
			}
		}
	}

	content := strings.Join(sections, "\n")
	return style.Width(width).Height(height).Render(content)
}

func categoryLabels(s *retro.Summary) []string {
	out := make([]string, len(s.Categories))
	for i, c := range s.Categories {
		out[i] = c.Category
	}
	return out
}

func agentLabels(s *retro.Summary) []string {
	out := make([]string, len(s.Agents))
	for i, a := range s.Agents {
		out[i] = a.Agent
	}
	return out
}

func maxLabelLen(labels []string) int {
	n := 0
	for _, l := range labels {
		if len(l) > n {
			n = len(l)
		}
	}
	return n
}
