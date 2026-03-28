package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/GrayFlash/kirkup-cli/retro"
	"github.com/GrayFlash/kirkup-cli/store"
)

type panel int

const (
	panelLeft panel = iota
	panelRight
)

type periodMode int

const (
	periodWeek periodMode = iota
	periodMonth
)

// Model is the root bubbletea model for the TUI.
type Model struct {
	store      store.Store
	gapMinutes int

	// layout
	width  int
	height int
	focus  panel

	// period state
	mode   periodMode
	offset int // weeks/months back from now (0 = current)

	// project list
	projects []projectEntry
	selected int // index into projects; 0 = "All"

	// loaded summary
	summary *retro.Summary
	loading bool
	err     error
}

type summaryMsg struct {
	summary  *retro.Summary
	projects []projectEntry
	err      error
}

// New creates a new TUI model.
func New(s store.Store, gapMinutes int) Model {
	return Model{
		store:      s,
		gapMinutes: gapMinutes,
		focus:      panelLeft,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadSummary()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case summaryMsg:
		m.loading = false
		m.err = msg.err
		m.summary = msg.summary
		if msg.projects != nil {
			m.projects = msg.projects
			if m.selected >= len(m.projects) {
				m.selected = 0
			}
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "tab":
			if m.focus == panelLeft {
				m.focus = panelRight
			} else {
				m.focus = panelLeft
			}

		case "up", "k":
			if m.focus == panelLeft && m.selected > 0 {
				m.selected--
				m.loading = true
				return m, m.loadSummary()
			}

		case "down", "j":
			if m.focus == panelLeft && m.selected < len(m.projects)-1 {
				m.selected++
				m.loading = true
				return m, m.loadSummary()
			}

		case "left", "h":
			m.offset++
			m.loading = true
			return m, m.loadSummary()

		case "right", "l":
			if m.offset > 0 {
				m.offset--
				m.loading = true
				return m, m.loadSummary()
			}

		case "m":
			if m.mode == periodWeek {
				m.mode = periodMonth
			} else {
				m.mode = periodWeek
			}
			m.offset = 0
			m.loading = true
			return m, m.loadSummary()

		case "r":
			m.loading = true
			return m, m.loadSummary()
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return ""
	}

	header := m.viewHeader()
	footer := m.viewFooter()
	help := m.viewHelp()

	headerH := lipgloss.Height(header)
	footerH := lipgloss.Height(footer)
	helpH := lipgloss.Height(help)
	bodyH := m.height - headerH - footerH - helpH

	leftW := 28
	if leftW > m.width {
		leftW = m.width
	}
	rightW := m.width - leftW - 1
	if rightW < 0 {
		rightW = 0
	}

	left := renderLeft(m.projects, m.selected, m.focus == panelLeft, leftW, bodyH)
	right := renderRight(m.summary, m.focus == panelRight, rightW, bodyH)

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer, help)
}

func (m Model) viewHeader() string {
	period := m.periodLabel()

	nav := ""
	if m.offset > 0 {
		nav = stylePeriodNav.Render("← ") + stylePeriod.Render(period) + stylePeriodNav.Render(" →")
	} else {
		nav = stylePeriodNav.Render("← ") + stylePeriod.Render(period)
	}

	title := styleHeader.Render("kirkup")
	left := title + styleMuted.Render("  ·  ") + nav

	modeLabel := "week"
	if m.mode == periodMonth {
		modeLabel = "month"
	}
	right := styleMuted.Render("[" + modeLabel + "]")
	if m.loading {
		right = styleMuted.Render("loading…")
	}
	if m.err != nil {
		right = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("error: " + m.err.Error())
	}

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	gap = max(gap, 0)
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) viewFooter() string {
	return styleMuted.Render(strings.Repeat("─", m.width))
}

func (m Model) viewHelp() string {
	keys := []string{
		"↑/↓ select project",
		"←/→ period",
		"m week/month",
		"tab switch panel",
		"r refresh",
		"q quit",
	}
	return styleKeyHelp.Render(" " + strings.Join(keys, "  ·  "))
}

func (m Model) periodLabel() string {
	from, to := m.periodRange()
	switch m.mode {
	case periodMonth:
		return from.Format("January 2006")
	default:
		return fmt.Sprintf("%s — %s", from.Format("Jan 2"), to.Format("Jan 2, 2006"))
	}
}

func (m Model) periodRange() (time.Time, time.Time) {
	now := time.Now()
	switch m.mode {
	case periodMonth:
		base := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		base = base.AddDate(0, -m.offset, 0)
		from := base
		to := from.AddDate(0, 1, 0).Add(-time.Second)
		return from, to
	default:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		monday := now.AddDate(0, 0, -(weekday-1)-(m.offset*7))
		from := truncDay(monday)
		to := from.AddDate(0, 0, 6).Add(24*time.Hour - time.Second)
		return from, to
	}
}

func (m Model) selectedProject() string {
	if len(m.projects) == 0 || m.selected == 0 {
		return ""
	}
	return m.projects[m.selected].name
}

func (m Model) loadSummary() tea.Cmd {
	return func() tea.Msg {
		from, to := m.periodRange()
		project := m.selectedProject()

		summary, err := retro.Aggregate(
			context.Background(), m.store,
			from, to, project,
			m.gapMinutes,
		)
		if err != nil {
			return summaryMsg{err: err}
		}

		// Fetch all events without project filter to build the list.
		// We rebuild this if we are selecting 'All' (index 0) or if projects is empty,
		// but really we want to rebuild it whenever period changes.
		// A cleaner way is just to always rebuild the project list when loading summary.
		// To avoid losing selection, we'll keep the current selection index.
		var projects []projectEntry
		allSummary, err := retro.Aggregate(
			context.Background(), m.store,
			from, to, "",
			m.gapMinutes,
		)
		if err == nil {
			projects = append(projects, projectEntry{name: "", prompts: allSummary.TotalPrompts})
			for _, p := range allSummary.Projects {
				projects = append(projects, projectEntry{name: p.Name, prompts: p.Prompts})
			}
		}

		return summaryMsg{summary: summary, projects: projects}
	}
}

func truncDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

