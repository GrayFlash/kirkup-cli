package retro

import (
	"fmt"
	"io"
	"math"
	"strings"
	"time"
)

const (
	barWidth  = 20
	separator = " ─────────────────────────────────────────────────────────────────"
)

// Render writes the full retrospective summary to w.
func Render(w io.Writer, s *Summary) {
	if s.TotalPrompts == 0 {
		fmt.Fprintln(w, "no events found for this period")
		return
	}

	renderHeader(w, s)
	renderProjects(w, s)
	renderCategories(w, s)
	renderAgents(w, s)
	renderContextSwitches(w, s)
	renderDailyActivity(w, s)
}

func renderHeader(w io.Writer, s *Summary) {
	var title string
	if isSameWeek(s.From, s.To) {
		title = fmt.Sprintf("Week of %s — %s", s.From.Format("Jan 2"), s.To.Format("Jan 2, 2006"))
	} else if isSameMonth(s.From, s.To) {
		title = s.From.Format("January 2006")
	} else {
		title = fmt.Sprintf("%s — %s", s.From.Format("Jan 2, 2006"), s.To.Format("Jan 2, 2006"))
	}
	width := len(title) + 4
	fmt.Fprintf(w, "\n╭%s╮\n", strings.Repeat("─", width))
	fmt.Fprintf(w, "│  %s  │\n", title)
	fmt.Fprintf(w, "╰%s╯\n\n", strings.Repeat("─", width))
}

func renderProjects(w io.Writer, s *Summary) {
	fmt.Fprintln(w, " Projects                          Sessions   Prompts   Est. Time")
	fmt.Fprintln(w, separator)
	for _, p := range s.Projects {
		name := p.Name
		if name == "" {
			name = "(unknown)"
		}
		fmt.Fprintf(w, " %-34s %-10d %-9d %s\n",
			name, p.Sessions, p.Prompts, fmtDuration(p.EstTime))
	}
	fmt.Fprintln(w, separator)
	fmt.Fprintf(w, " %-34s %-10d %-9d %s\n",
		"Total", s.TotalSessions, s.TotalPrompts, fmtDuration(s.TotalEstTime))
	fmt.Fprintln(w)
}

func renderCategories(w io.Writer, s *Summary) {
	if len(s.Categories) == 0 {
		return
	}
	fmt.Fprintln(w, " By Category")
	fmt.Fprintln(w, separator)
	for _, c := range s.Categories {
		fmt.Fprintf(w, " %-16s %s  %.0f%%\n",
			c.Category, bar(c.Percent), c.Percent)
	}
	fmt.Fprintln(w)
}

func renderAgents(w io.Writer, s *Summary) {
	if len(s.Agents) == 0 {
		return
	}
	fmt.Fprintln(w, " By Agent")
	fmt.Fprintln(w, separator)
	for _, a := range s.Agents {
		fmt.Fprintf(w, " %-16s %s  %.0f%%\n",
			a.Agent, bar(a.Percent), a.Percent)
	}
	fmt.Fprintln(w)
}

func renderContextSwitches(w io.Writer, s *Summary) {
	if len(s.Daily) == 0 {
		return
	}
	fmt.Fprintln(w, " Context Switches (project changes within a day)")
	fmt.Fprintln(w, separator)

	var parts []string
	total := 0
	for _, d := range s.Daily {
		parts = append(parts, fmt.Sprintf("%s: %d", d.Date.Format("Mon"), d.ContextSwitches))
		total += d.ContextSwitches
	}
	avg := 0.0
	if len(s.Daily) > 0 {
		avg = float64(total) / float64(len(s.Daily))
	}
	fmt.Fprintf(w, " %s\n", strings.Join(parts, "  "))
	fmt.Fprintf(w, " Weekly avg: %.1f switches/day\n\n", avg)
}

func renderDailyActivity(w io.Writer, s *Summary) {
	if len(s.Daily) == 0 {
		return
	}
	fmt.Fprintln(w, " Daily Activity")
	fmt.Fprintln(w, separator)

	maxPrompts := 0
	for _, d := range s.Daily {
		if d.Prompts > maxPrompts {
			maxPrompts = d.Prompts
		}
	}

	for _, d := range s.Daily {
		pct := 0.0
		if maxPrompts > 0 {
			pct = float64(d.Prompts) / float64(maxPrompts) * 100
		}
		fmt.Fprintf(w, " %-4s %s  %d prompts\n",
			d.Date.Format("Mon"), bar(pct), d.Prompts)
	}
	fmt.Fprintln(w)
}

// bar renders a fixed-width bar chart segment.
func bar(percent float64) string {
	filled := int(math.Round(percent / 100 * barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
}

// fmtDuration formats a duration as "~Xh" or "~Xm".
func fmtDuration(d time.Duration) string {
	h := d.Hours()
	if h >= 1 {
		return fmt.Sprintf("~%.1fh", h)
	}
	return fmt.Sprintf("~%.0fm", d.Minutes())
}

func isSameWeek(a, b time.Time) bool {
	ay, aw := a.ISOWeek()
	by, bw := b.ISOWeek()
	return ay == by && aw == bw
}

func isSameMonth(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month()
}
