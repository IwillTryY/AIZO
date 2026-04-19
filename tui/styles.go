package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Primary   = lipgloss.Color("#7C3AED")
	Secondary = lipgloss.Color("#06B6D4")
	Success   = lipgloss.Color("#10B981")
	Warning   = lipgloss.Color("#F59E0B")
	Danger    = lipgloss.Color("#EF4444")
	Muted     = lipgloss.Color("#6B7280")
	BgDark    = lipgloss.Color("#1F2937")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			MarginBottom(1)

	TabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(Muted)

	ActiveTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(Primary).
			Bold(true).
			Underline(true)

	StatusHealthy = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	StatusDegraded = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	StatusUnhealthy = lipgloss.NewStyle().
			Foreground(Danger).
			Bold(true)

	StatusMuted = lipgloss.NewStyle().
			Foreground(Muted)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Muted).
			Padding(1, 2)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Secondary)

	HelpStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)
)

// StatusStyle returns the appropriate style for a status string
func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "healthy", "running", "online", "Healthy":
		return StatusHealthy
	case "degraded", "warning", "Degraded":
		return StatusDegraded
	case "unhealthy", "failed", "dead", "Critical", "critical":
		return StatusUnhealthy
	default:
		return StatusMuted
	}
}
