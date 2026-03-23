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
func Render(w io.Writer, s *Summary) error {
	ew := &errWriter{w: w}
	if s.TotalPrompts == 0 {
		ew.println("no events found for this period")
		return ew.err
	}
	renderHeader(ew, s)
	renderProjects(ew, s)
	renderCategories(ew, s)
	renderAgents(ew, s)
	renderContextSwitches(ew, s)
	renderDailyActivity(ew, s)
	return ew.err
}

// errWriter captures the first write error so callers don't check every call.
type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) printf(format string, args ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintf(ew.w, format, args...)
}

func (ew *errWriter) println(args ...any) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintln(ew.w, args...)
}

func renderHeader(ew *errWriter, s *Summary) {
	var title string
	if isSameWeek(s.From, s.To) {
		title = fmt.Sprintf("Week of %s — %s", s.From.Format("Jan 2"), s.To.Format("Jan 2, 2006"))
	} else if isSameMonth(s.From, s.To) {
		title = s.From.Format("January 2006")
	} else {
		title = fmt.Sprintf("%s — %s", s.From.Format("Jan 2, 2006"), s.To.Format("Jan 2, 2006"))
	}
	width := len(title) + 4
	ew.printf("\n╭%s╮\n", strings.Repeat("─", width))
	ew.printf("│  %s  │\n", title)
	ew.printf("╰%s╯\n\n", strings.Repeat("─", width))
}

func renderProjects(ew *errWriter, s *Summary) {
	ew.println(" Projects                          Sessions   Prompts   Est. Time")
	ew.println(separator)
	for _, p := range s.Projects {
		name := p.Name
		if name == "" {
			name = "(unknown)"
		}
		ew.printf(" %-34s %-10d %-9d %s\n", name, p.Sessions, p.Prompts, fmtDuration(p.EstTime))
	}
	ew.println(separator)
	ew.printf(" %-34s %-10d %-9d %s\n", "Total", s.TotalSessions, s.TotalPrompts, fmtDuration(s.TotalEstTime))
	ew.println()
}

func renderCategories(ew *errWriter, s *Summary) {
	if len(s.Categories) == 0 {
		return
	}
	ew.println(" By Category")
	ew.println(separator)
	for _, c := range s.Categories {
		ew.printf(" %-16s %s  %.0f%%\n", c.Category, bar(c.Percent), c.Percent)
	}
	ew.println()
}

func renderAgents(ew *errWriter, s *Summary) {
	if len(s.Agents) == 0 {
		return
	}
	ew.println(" By Agent")
	ew.println(separator)
	for _, a := range s.Agents {
		ew.printf(" %-16s %s  %.0f%%\n", a.Agent, bar(a.Percent), a.Percent)
	}
	ew.println()
}

func renderContextSwitches(ew *errWriter, s *Summary) {
	if len(s.Daily) == 0 {
		return
	}
	ew.println(" Context Switches (project changes within a day)")
	ew.println(separator)

	var parts []string
	total := 0
	for _, d := range s.Daily {
		parts = append(parts, fmt.Sprintf("%s: %d", d.Date.Format("Mon"), d.ContextSwitches))
		total += d.ContextSwitches
	}
	avg := float64(total) / float64(len(s.Daily))
	ew.printf(" %s\n", strings.Join(parts, "  "))
	ew.printf(" Weekly avg: %.1f switches/day\n\n", avg)
}

func renderDailyActivity(ew *errWriter, s *Summary) {
	if len(s.Daily) == 0 {
		return
	}
	ew.println(" Daily Activity")
	ew.println(separator)

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
		ew.printf(" %-4s %s  %d prompts\n", d.Date.Format("Mon"), bar(pct), d.Prompts)
	}
	ew.println()
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
