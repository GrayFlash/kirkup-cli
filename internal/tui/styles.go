package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorAccent   = lipgloss.Color("12")  // bright blue
	colorMuted    = lipgloss.Color("240") // dark gray
	colorSelected = lipgloss.Color("4")   // blue
	colorBorder   = lipgloss.Color("238") // subtle border

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("0")).
			Padding(0, 1)

	stylePeriod = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	stylePeriodNav = lipgloss.NewStyle().
			Foreground(colorMuted)

	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	stylePanelFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent)

	styleProjectItem = lipgloss.NewStyle().
				Padding(0, 1)

	styleProjectSelected = lipgloss.NewStyle().
				Padding(0, 1).
				Background(colorSelected).
				Foreground(lipgloss.Color("15")).
				Bold(true)

	styleSectionTitle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	styleMuted = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleKeyHelp = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("4"))

	styleBarEmpty = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleStat = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)
)
