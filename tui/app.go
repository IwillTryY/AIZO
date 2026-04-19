package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Tab represents a TUI tab
type Tab int

const (
	TabDashboard Tab = iota
	TabContainers
	TabEntities
	TabMetrics
	TabLogs
	TabAI
	TabAudit
)

var tabNames = []string{"Dashboard", "Containers", "Entities", "Metrics", "Logs", "AI Chat", "Audit"}

// Model is the main TUI model
type Model struct {
	activeTab  Tab
	width      int
	height     int
	dashboard  DashboardModel
	containers ContainerModel
	entities   EntityModel
	metrics    MetricsModel
	logs       LogsModel
	chat       ChatModel
	audit      AuditModel
	quitting   bool
}

// NewModel creates a new TUI model
func NewModel() Model {
	return Model{
		activeTab:  TabDashboard,
		dashboard:  NewDashboardModel(),
		containers: NewContainerModel(),
		entities:   NewEntityModel(),
		metrics:    NewMetricsModel(),
		logs:       NewLogsModel(),
		chat:       NewChatModel(),
		audit:      NewAuditModel(),
	}
}

// isInputActive returns true if any tab is in text input mode
func (m Model) isInputActive() bool {
	switch m.activeTab {
	case TabAI:
		return m.chat.inputActive
	case TabContainers:
		return m.containers.inputMode != ""
	case TabLogs:
		return m.logs.searchMode
	}
	return false
}
func (m Model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If any tab has input active, delegate everything to it
		if m.isInputActive() {
			if msg.String() == "esc" {
				// Esc always exits input mode
				switch m.activeTab {
				case TabAI:
					m.chat.inputActive = false
				case TabContainers:
					m.containers.inputMode = ""
					m.containers.input = ""
				case TabLogs:
					m.logs.searchMode = false
					m.logs.searchInput = ""
				}
				return m, nil
			}
			var cmd tea.Cmd
			switch m.activeTab {
			case TabAI:
				m.chat, cmd = m.chat.Update(msg)
			case TabContainers:
				m.containers, cmd = m.containers.Update(msg)
			case TabLogs:
				m.logs, cmd = m.logs.Update(msg)
			}
			return m, cmd
		}

		// Global keys (only when not in input mode)
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			m.quitting = true
			return m, tea.Quit
		case "ctrl+n", "right":
			m.activeTab = Tab((int(m.activeTab) + 1) % len(tabNames))
			return m, nil
		case "ctrl+p", "left":
			m.activeTab = Tab((int(m.activeTab) - 1 + len(tabNames)) % len(tabNames))
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Delegate to active tab
	var cmd tea.Cmd
	switch m.activeTab {
	case TabDashboard:
		m.dashboard, cmd = m.dashboard.Update(msg)
	case TabContainers:
		m.containers, cmd = m.containers.Update(msg)
	case TabEntities:
		m.entities, cmd = m.entities.Update(msg)
	case TabMetrics:
		m.metrics, cmd = m.metrics.Update(msg)
	case TabLogs:
		m.logs, cmd = m.logs.Update(msg)
	case TabAI:
		m.chat, cmd = m.chat.Update(msg)
	case TabAudit:
		m.audit, cmd = m.audit.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	w := m.width
	if w < 40 {
		w = 80
	}
	h := m.height
	if h < 10 {
		h = 24
	}

	// Header: tab indicator
	tabIndicator := lipgloss.NewStyle().
		Foreground(Muted).
		Render(fmt.Sprintf("  [%d/%d] %s", int(m.activeTab)+1, len(tabNames), tabNames[m.activeTab]))

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary).
		Render("AIZO")

	topLine := header + "  " + tabIndicator
	topBar := lipgloss.NewStyle().
		Width(w).
		Render(topLine)

	// Content
	var content string
	switch m.activeTab {
	case TabDashboard:
		content = m.dashboard.View()
	case TabContainers:
		content = m.containers.View()
	case TabEntities:
		content = m.entities.View()
	case TabMetrics:
		content = m.metrics.View()
	case TabLogs:
		content = m.logs.View()
	case TabAI:
		content = m.chat.View()
	case TabAudit:
		content = m.audit.View()
	}

	// Pad content to fill screen
	contentLines := strings.Count(content, "\n")
	availableHeight := h - 4 // header + footer + padding
	if contentLines < availableHeight {
		content += strings.Repeat("\n", availableHeight-contentLines)
	}

	// Footer
	var footer string
	if m.isInputActive() {
		footer = lipgloss.NewStyle().Foreground(Warning).Render("  esc: exit input")
	} else {
		footer = lipgloss.NewStyle().Foreground(Muted).Render("  ctrl+n/→: next tab  ctrl+p/←: prev tab  q: quit")
	}

	return topBar + "\n" + strings.Repeat("─", w) + "\n" + content + "\n" + footer
}

// Run starts the TUI
func Run() error {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
