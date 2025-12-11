package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LogsModel handles the logs view
type LogsModel struct {
	logs            []LogEntry
	agentFilter     string // Filter by agent ID, empty means all
	levelFilter     string // Filter by log level, empty means all
	availableLevels []string
	agents          []Agent // Available agents for filtering
	width           int
	height          int
	scrollOffset    int
	filterCursor    int // 0 = agent, 1 = level
	showFilterMenu  bool
	screenIndex     int
}

type LogMessagesBatch struct {
	Messages []LogEntry
}

// NewLogsModel creates a new logs model
func NewLogsModel() *LogsModel {
	return &LogsModel{
		logs:            []LogEntry{},
		availableLevels: []string{"all", "debug", "info", "warn", "error"},
		agents:          []Agent{},
		screenIndex:     0,
	}
}

func (m *LogsModel) SetScreenIndex(index int) {
	m.screenIndex = index
}

func (m *LogsModel) Init() tea.Cmd {
	return nil
}

func (m *LogsModel) GetTitle() string {
	return "📋 Agent Logs"
}

// SetSize updates the dimensions
func (m *LogsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetAgents sets the available agents for filtering
func (m *LogsModel) SetAgents(agents []Agent) {
	m.agents = agents
}

// AddLog adds a log entry
func (m *LogsModel) AddLog(entry LogEntry) {
	m.logs = append(m.logs, entry)
	// Keep only last 1000 logs
	if len(m.logs) > 1000 {
		m.logs = m.logs[len(m.logs)-1000:]
	}
}

// GetFilteredLogs returns logs matching current filters
func (m LogsModel) GetFilteredLogs() []LogEntry {
	var filtered []LogEntry
	for _, log := range m.logs {
		if m.agentFilter != "" && log.AgentID != m.agentFilter {
			continue
		}
		if m.levelFilter != "" && m.levelFilter != "all" && log.Level != m.levelFilter {
			continue
		}
		filtered = append(filtered, log)
	}
	return filtered
}

// Update handles input for the logs view
func (m *LogsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ChatMessage:
		m.AddLog(LogEntry{
			Timestamp: time.Now().Format("15:04:05"),
			Level:     msg.Role,
			AgentID:   msg.AgentID,
			Message:   msg.Content,
		})
		return m, nil
	case tea.WindowSizeMsg:
		m.SetSize(int(msg.Width), int(msg.Height))
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.scrollOffset++
		case "down", "j":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "f":
			m.showFilterMenu = !m.showFilterMenu
		case "left", "h":
			if m.showFilterMenu && m.filterCursor > 0 {
				m.filterCursor--
			}
		case "right", "l":
			if m.showFilterMenu && m.filterCursor < 1 {
				m.filterCursor++
			}
		case "1", "2", "3", "4", "5":
			if m.showFilterMenu && m.filterCursor == 1 {
				// Level filter
				idx := int(msg.String()[0] - '1')
				if idx >= 0 && idx < len(m.availableLevels) {
					m.levelFilter = m.availableLevels[idx]
					if m.levelFilter == "all" {
						m.levelFilter = ""
					}
				}
			} else if m.showFilterMenu && m.filterCursor == 0 {
				// Agent filter
				idx := int(msg.String()[0] - '1')
				if idx == 0 {
					m.agentFilter = ""
				} else if idx > 0 && idx <= len(m.agents) {
					m.agentFilter = m.agents[idx-1].ID
				}
			}
		case "c":
			// Clear filters
			m.agentFilter = ""
			m.levelFilter = ""
		case "home", "g":
			m.scrollOffset = 0
		case "end", "G":
			logs := m.GetFilteredLogs()
			if len(logs) > m.height-6 {
				m.scrollOffset = len(logs) - (m.height - 6)
			}
		}
	}
	return m, nil
}

// View renders the logs view
func (m *LogsModel) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("📋 Agent Logs"))
	b.WriteString("\n\n")

	// Filter bar
	filterBar := m.renderFilterBar()
	b.WriteString(filterBar)
	b.WriteString("\n\n")

	// Filter menu if shown
	if m.showFilterMenu {
		b.WriteString(m.renderFilterMenu())
		b.WriteString("\n\n")
	}

	logs := m.GetFilteredLogs()

	if len(logs) == 0 {
		b.WriteString(HelpStyle.Render("No logs to display."))
		b.WriteString("\n")
	} else {
		// Calculate visible logs
		visibleHeight := m.height - 12
		if visibleHeight < 5 {
			visibleHeight = 5
		}

		endIdx := len(logs) - m.scrollOffset
		if endIdx > len(logs) {
			endIdx = len(logs)
		}
		if endIdx < 0 {
			endIdx = 0
		}

		startIdx := endIdx - visibleHeight
		if startIdx < 0 {
			startIdx = 0
		}

		for i := startIdx; i < endIdx; i++ {
			log := logs[i]
			line := m.renderLogEntry(log)
			b.WriteString(line)
			b.WriteString("\n")
		}

		// Scroll indicator
		if len(logs) > visibleHeight {
			scrollInfo := fmt.Sprintf("Showing %d-%d of %d", startIdx+1, endIdx, len(logs))
			b.WriteString("\n")
			b.WriteString(HelpStyle.Render(scrollInfo))
		}
	}

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("↑/↓ scroll • f filter • c clear filters • tab switch screen • q quit"))

	return b.String()
}

func (m LogsModel) renderFilterBar() string {
	var parts []string

	// Agent filter
	agentText := "All Agents"
	if m.agentFilter != "" {
		for _, a := range m.agents {
			if a.ID == m.agentFilter {
				agentText = a.Name
				break
			}
		}
	}
	parts = append(parts, fmt.Sprintf("Agent: %s", agentText))

	// Level filter
	levelText := "All Levels"
	if m.levelFilter != "" {
		levelText = strings.ToUpper(m.levelFilter)
	}
	parts = append(parts, fmt.Sprintf("Level: %s", levelText))

	return HelpStyle.Render(strings.Join(parts, " | "))
}

func (m LogsModel) renderFilterMenu() string {
	var b strings.Builder

	boxStyle := BoxStyle.Width(m.width - 4)

	// Agent filter options
	var agentOptions []string
	agentLabel := "Agent: "
	if m.filterCursor == 0 {
		agentLabel = SelectedItemStyle.Render(agentLabel)
	}
	agentOptions = append(agentOptions, agentLabel)

	allAgentsOpt := "1:All"
	if m.agentFilter == "" {
		allAgentsOpt = ActiveTabStyle.Render(allAgentsOpt)
	}
	agentOptions = append(agentOptions, allAgentsOpt)

	for i, a := range m.agents {
		opt := fmt.Sprintf("%d:%s", i+2, a.Name)
		if m.agentFilter == a.ID {
			opt = ActiveTabStyle.Render(opt)
		}
		agentOptions = append(agentOptions, opt)
	}

	// Level filter options
	var levelOptions []string
	levelLabel := "Level: "
	if m.filterCursor == 1 {
		levelLabel = SelectedItemStyle.Render(levelLabel)
	}
	levelOptions = append(levelOptions, levelLabel)

	for i, lvl := range m.availableLevels {
		opt := fmt.Sprintf("%d:%s", i+1, lvl)
		if (m.levelFilter == "" && lvl == "all") || m.levelFilter == lvl {
			opt = ActiveTabStyle.Render(opt)
		}
		levelOptions = append(levelOptions, opt)
	}

	b.WriteString(strings.Join(agentOptions, " "))
	b.WriteString("\n")
	b.WriteString(strings.Join(levelOptions, " "))

	return boxStyle.Render(b.String())
}

func (m LogsModel) renderLogEntry(log LogEntry) string {
	// Format: [TIMESTAMP] [LEVEL] [AGENT] message
	timestamp := LogTimestampStyle.Render(log.Timestamp)
	level := LogLevelStyle(log.Level).Render(fmt.Sprintf("[%-5s]", strings.ToUpper(log.Level)))

	agentName := log.AgentID
	for _, a := range m.agents {
		if a.ID == log.AgentID {
			agentName = a.Name
			break
		}
	}
	agent := lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Render(fmt.Sprintf("[%s]", agentName))

	return fmt.Sprintf("%s %s %s %s", timestamp, level, agent, log.Message)
}
