package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/realityos/aizo/layer5"
)

// DashboardModel is the dashboard tab
type DashboardModel struct {
	containerCount int
	status         string
}

func NewDashboardModel() DashboardModel {
	return DashboardModel{status: "Loading..."}
}

func (m DashboardModel) Init() tea.Cmd { return nil }

func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "r" {
			m.status = "Refreshed"
			// Try to count containers
			runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
			if err == nil {
				containers, err := runtime.ListContainers(context.Background())
				if err == nil {
					m.containerCount = len(containers)
				}
			}
		}
	}
	return m, nil
}

func (m DashboardModel) View() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("System Overview"))
	b.WriteString("\n\n")

	b.WriteString(BoxStyle.Render(fmt.Sprintf(
		"%s\n  Containers: %d\n  Adapters: HTTP, SSH, gRPC, MQTT\n  Storage: SQLite (~/.aizo/aizo.db)",
		HeaderStyle.Render("Infrastructure"), m.containerCount,
	)))
	b.WriteString("\n\n")
	b.WriteString(BoxStyle.Render(fmt.Sprintf(
		"%s\n  CPU: --%%  Memory: --%%  Disk: --%%\n  Network Errors: 0\n  Recent Incidents: 0",
		HeaderStyle.Render("Resources"),
	)))
	b.WriteString("\n\n")
	b.WriteString(BoxStyle.Render(fmt.Sprintf(
		"%s\n  Pending Proposals: 0\n  Total Incidents: 0\n  Auto-resolved: 0",
		HeaderStyle.Render("AI Intelligence"),
	)))
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("  r: refresh"))
	return b.String()
}

// ContainerModel is the containers tab
type ContainerModel struct {
	containers  []*layer5.WSL2Container
	selected    int
	message     string
	inputMode   string // "", "create", "shell"
	input       string
	shellOutput []string
	shellContainer *layer5.WSL2Container
}

func NewContainerModel() ContainerModel {
	return ContainerModel{containers: make([]*layer5.WSL2Container, 0)}
}

func (m ContainerModel) Update(msg tea.Msg) (ContainerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Shell mode — interactive exec
		if m.inputMode == "shell" {
			switch msg.Type {
			case tea.KeyEnter:
				if m.input != "" {
					cmd := m.input
					m.shellOutput = append(m.shellOutput, "$ "+cmd)
					m.input = ""

					runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
					if err != nil {
						m.shellOutput = append(m.shellOutput, "Error: "+err.Error())
					} else {
						out, err := runtime.ExecInContainer(context.Background(), m.shellContainer, []string{"sh", "-c", cmd})
						if err != nil {
							m.shellOutput = append(m.shellOutput, "Error: "+err.Error())
						} else {
							for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
								m.shellOutput = append(m.shellOutput, line)
							}
						}
					}
				}
			case tea.KeyBackspace:
				if len(m.input) > 0 {
					m.input = m.input[:len(m.input)-1]
				}
			case tea.KeySpace:
				m.input += " "
			case tea.KeyRunes:
				m.input += string(msg.Runes)
			}
			return m, nil
		}

		// Create mode
		if m.inputMode == "create" {
			switch msg.Type {
			case tea.KeyEnter:
				if m.input != "" {
					m.message = m.createContainer(m.input)
					m.input = ""
					m.inputMode = ""
					m.refresh()
				}
			case tea.KeyEsc:
				m.inputMode = ""
				m.input = ""
			case tea.KeyBackspace:
				if len(m.input) > 0 {
					m.input = m.input[:len(m.input)-1]
				}
			case tea.KeySpace:
				m.input += " "
			case tea.KeyRunes:
				m.input += string(msg.Runes)
			}
			return m, nil
		}

		switch msg.String() {
		case "r":
			m.refresh()
			m.message = "Refreshed"
		case "c":
			m.inputMode = "create"
			m.input = ""
			m.message = ""
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.containers)-1 {
				m.selected++
			}
		case "enter":
			// Open shell into selected container
			if len(m.containers) > 0 {
				c := m.containers[m.selected]
				if c.Running {
					m.inputMode = "shell"
					m.shellContainer = c
					m.shellOutput = []string{
						fmt.Sprintf("Connected to %s (%s)", c.Name, c.ID),
						"Type commands below. esc to exit.",
						strings.Repeat("─", 50),
					}
					m.input = ""
				} else {
					m.message = fmt.Sprintf("Container %s is not running. Press 's' to start it first.", c.Name)
				}
			}
		case "s":
			if len(m.containers) > 0 {
				m.message = m.startContainer(m.containers[m.selected])
				m.refresh()
			}
		case "x":
			if len(m.containers) > 0 {
				m.message = m.stopContainer(m.containers[m.selected])
				m.refresh()
			}
		case "d":
			if len(m.containers) > 0 {
				m.message = m.removeContainer(m.containers[m.selected])
				m.refresh()
			}
		}
	}
	return m, nil
}

func (m *ContainerModel) refresh() {
	runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
	if err != nil {
		return
	}
	containers, err := runtime.ListContainers(context.Background())
	if err != nil {
		return
	}
	m.containers = containers
	if m.selected >= len(m.containers) {
		m.selected = max(0, len(m.containers)-1)
	}
}

func (m *ContainerModel) createContainer(name string) string {
	runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
	if err != nil {
		return "Error: " + err.Error()
	}
	c, err := runtime.CreateContainer(context.Background(), name, "", []string{"/bin/sh"}, nil)
	if err != nil {
		return "Error: " + err.Error()
	}
	return fmt.Sprintf("✓ Created %s (%s)", c.Name, c.ID)
}

func (m *ContainerModel) startContainer(c *layer5.WSL2Container) string {
	runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
	if err != nil {
		return "Error: " + err.Error()
	}
	err = runtime.StartContainer(context.Background(), c, []string{"/bin/sh"}, nil)
	if err != nil {
		return "Error: " + err.Error()
	}
	return fmt.Sprintf("✓ Started %s", c.Name)
}

func (m *ContainerModel) stopContainer(c *layer5.WSL2Container) string {
	runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
	if err != nil {
		return "Error: " + err.Error()
	}
	err = runtime.StopContainer(context.Background(), c, 5)
	if err != nil {
		return "Error: " + err.Error()
	}
	return fmt.Sprintf("✓ Stopped %s", c.Name)
}

func (m *ContainerModel) removeContainer(c *layer5.WSL2Container) string {
	runtime, err := layer5.NewWSL2Runtime("Ubuntu", "~/realityos")
	if err != nil {
		return "Error: " + err.Error()
	}
	err = runtime.RemoveContainer(context.Background(), c)
	if err != nil {
		return "Error: " + err.Error()
	}
	return fmt.Sprintf("✓ Removed %s", c.Name)
}

func (m ContainerModel) View() string {
	var b strings.Builder

	// Shell mode
	if m.inputMode == "shell" {
		b.WriteString(HeaderStyle.Render(fmt.Sprintf("Shell: %s", m.shellContainer.Name)))
		b.WriteString("\n\n")

		// Show last 20 lines of output
		start := 0
		if len(m.shellOutput) > 20 {
			start = len(m.shellOutput) - 20
		}
		for _, line := range m.shellOutput[start:] {
			b.WriteString("  " + line + "\n")
		}

		b.WriteString("\n")
		b.WriteString("  $ " + m.input + "█\n")
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("  enter: run command | esc: exit shell"))
		return b.String()
	}

	// Create mode
	if m.inputMode == "create" {
		b.WriteString(HeaderStyle.Render("Containers"))
		b.WriteString("\n\n")
		b.WriteString("  Container name: " + m.input + "█\n\n")
		b.WriteString(HelpStyle.Render("  enter: create | esc: cancel"))
		return b.String()
	}

	b.WriteString(HeaderStyle.Render("Containers"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  %-14s %-20s %-10s %-8s\n", "ID", "NAME", "STATUS", "PID"))
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n")

	if len(m.containers) == 0 {
		b.WriteString(StatusMuted.Render("  No containers. Press 'c' to create one."))
	} else {
		for i, c := range m.containers {
			cursor := "  "
			if i == m.selected {
				cursor = "▸ "
			}
			status := StatusStyle(string(c.Status)).Render(string(c.Status))
			b.WriteString(fmt.Sprintf("%s%-14s %-20s %-10s %-8d\n", cursor, c.ID, c.Name, status, c.PID))
		}
	}

	if m.message != "" {
		b.WriteString("\n  " + m.message)
	}

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("  c: create | enter: shell | s: start | x: stop | d: delete | r: refresh | j/k: navigate"))
	return b.String()
}

// EntityModel is the entities tab
type EntityModel struct {
	message string
}

func NewEntityModel() EntityModel { return EntityModel{} }
func (m EntityModel) Update(msg tea.Msg) (EntityModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "r" {
			m.message = "Refreshed"
		}
	}
	return m, nil
}
func (m EntityModel) View() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Entities"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  %-20s %-15s %-12s %-30s\n", "ID", "TYPE", "STATE", "ENDPOINT"))
	b.WriteString(strings.Repeat("─", 80))
	b.WriteString("\n")
	b.WriteString(StatusMuted.Render("  Use CLI: aizo entity register --id <id> --name <name>"))
	if m.message != "" {
		b.WriteString("\n  " + m.message)
	}
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("  r: refresh"))
	return b.String()
}

// MetricsModel is the metrics tab
type MetricsModel struct{}

func NewMetricsModel() MetricsModel { return MetricsModel{} }
func (m MetricsModel) Update(msg tea.Msg) (MetricsModel, tea.Cmd) { return m, nil }
func (m MetricsModel) View() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Metrics"))
	b.WriteString("\n\n")
	b.WriteString(StatusMuted.Render("  Use CLI: aizo metrics query --entity <id> --last 1h"))
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("  r: refresh"))
	return b.String()
}

// LogsModel is the logs tab
type LogsModel struct {
	searchMode  bool
	searchInput string
	message     string
}

func NewLogsModel() LogsModel { return LogsModel{} }
func (m LogsModel) Update(msg tea.Msg) (LogsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searchMode {
			switch msg.Type {
			case tea.KeyEnter:
				if m.searchInput != "" {
					m.message = fmt.Sprintf("Searched for: %s (use CLI: aizo logs search \"%s\")", m.searchInput, m.searchInput)
					m.searchInput = ""
					m.searchMode = false
				}
			case tea.KeyBackspace:
				if len(m.searchInput) > 0 {
					m.searchInput = m.searchInput[:len(m.searchInput)-1]
				}
			case tea.KeySpace:
				m.searchInput += " "
			case tea.KeyRunes:
				m.searchInput += string(msg.Runes)
			}
			return m, nil
		}

		switch msg.String() {
		case "/":
			m.searchMode = true
			m.searchInput = ""
		case "r":
			m.message = "Refreshed"
		}
	}
	return m, nil
}
func (m LogsModel) View() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Logs"))
	b.WriteString("\n\n")

	if m.searchMode {
		b.WriteString("  Search: " + m.searchInput + "█\n\n")
		b.WriteString(HelpStyle.Render("  enter: search | esc: cancel"))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("  %-10s %-8s %-15s %s\n", "TIME", "LEVEL", "ENTITY", "MESSAGE"))
	b.WriteString(strings.Repeat("─", 80))
	b.WriteString("\n")
	b.WriteString(StatusMuted.Render("  No logs. Use CLI: aizo logs search <query>"))

	if m.message != "" {
		b.WriteString("\n\n  " + m.message)
	}

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("  /: search | r: refresh"))
	return b.String()
}

// ChatModel is the AI chat tab
type ChatModel struct {
	messages    []string
	input       string
	inputActive bool
}

func NewChatModel() ChatModel {
	return ChatModel{
		messages: []string{"AI: Hello! I'm the AIZO intelligence layer. How can I help?"},
	}
}

func (m ChatModel) Update(msg tea.Msg) (ChatModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.inputActive {
			if msg.String() == "i" || msg.String() == "enter" {
				m.inputActive = true
				return m, nil
			}
			return m, nil
		}

		switch msg.Type {
		case tea.KeyEnter:
			if m.input != "" {
				m.messages = append(m.messages, "You: "+m.input)
				m.messages = append(m.messages, "AI: [Use 'aizo ai chat' for full AI interaction]")
				m.input = ""
			}
		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		case tea.KeySpace:
			m.input += " "
		case tea.KeyRunes:
			m.input += string(msg.Runes)
		}
	}
	return m, nil
}

func (m ChatModel) View() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("AI Chat"))
	b.WriteString("\n\n")

	start := 0
	if len(m.messages) > 10 {
		start = len(m.messages) - 10
	}
	for _, msg := range m.messages[start:] {
		b.WriteString("  " + msg + "\n")
	}

	b.WriteString("\n")
	if m.inputActive {
		b.WriteString(fmt.Sprintf("  > %s█\n", m.input))
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render("  enter: send | esc: stop typing | For full AI: aizo ai chat"))
	} else {
		b.WriteString(StatusMuted.Render("  Press 'i' or Enter to start typing"))
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("  i/enter: type | For full AI: aizo ai chat"))
	}
	return b.String()
}

// AuditModel is the audit trail tab
type AuditModel struct{}

func NewAuditModel() AuditModel { return AuditModel{} }
func (m AuditModel) Update(msg tea.Msg) (AuditModel, tea.Cmd) { return m, nil }
func (m AuditModel) View() string {
	var b strings.Builder
	b.WriteString(HeaderStyle.Render("Audit Trail"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  %-10s %-8s %-15s %-20s %s\n", "TIME", "ACTOR", "ACTION", "RESOURCE", "DETAIL"))
	b.WriteString(strings.Repeat("─", 80))
	b.WriteString("\n")
	b.WriteString(StatusMuted.Render("  Use CLI: aizo audit list --last 24h"))
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("  r: refresh"))
	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
