package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorAccent  = lipgloss.Color("#7aa2f7")
	colorMuted   = lipgloss.Color("#565f89")
	colorOK      = lipgloss.Color("#9ece6a")
	colorWarn    = lipgloss.Color("#e0af68")
	colorErr     = lipgloss.Color("#f7768e")
	colorBorder  = lipgloss.Color("#3b4261")
	colorPrimary = lipgloss.Color("#c0caf5")
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	tabActiveStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Padding(0, 1)

	tabStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	mutedStyle = lipgloss.NewStyle().Foreground(colorMuted)

	repoHeaderStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	frameStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2)

	editingFrame = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(0, 1)

	statusOK   = lipgloss.NewStyle().Foreground(colorOK)
	statusWarn = lipgloss.NewStyle().Foreground(colorWarn)
	statusErr  = lipgloss.NewStyle().Foreground(colorErr)
	primary    = lipgloss.NewStyle().Foreground(colorPrimary)
)

func ciStyle(state string) lipgloss.Style {
	switch state {
	case "pass":
		return statusOK
	case "fail":
		return statusErr
	default:
		return statusWarn
	}
}

func reviewStyle(state string) lipgloss.Style {
	switch state {
	case "approved":
		return statusOK
	case "changes_requested":
		return statusErr
	default:
		return statusWarn
	}
}
