package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shuttl-ai/cli/ipc"
	"github.com/shuttl-ai/cli/log"
)

// AgentPickerModel handles agent selection
type AgentPickerModel struct {
	agents      []Agent
	cursor      int
	selected    map[string]bool // Track which agents have active sessions
	width       int
	height      int
	loading     bool
	err         error
	screenIndex int
	ipcClient   *ipc.Client
}

// NewAgentPickerModel creates a new agent picker
func NewAgentPickerModel(agents []Agent, ipcClient *ipc.Client) *AgentPickerModel {
	return &AgentPickerModel{
		agents:      agents,
		cursor:      0,
		selected:    make(map[string]bool),
		loading:     false,
		err:         nil,
		screenIndex: 0,
		ipcClient:   ipcClient,
	}
}

func (m *AgentPickerModel) Init() tea.Cmd {
	return func() tea.Msg {
		m.loading = true
		if m.ipcClient == nil || !m.ipcClient.IsRunning() {
			log.IPC("Cannot request agents: client not running")
			return AgentsLoadedMsg{Err: fmt.Errorf("IPC client not running")}
		}

		// Send the list_agents request
		log.IPC("Sending list_agents request")

		ctx, cancel := context.WithTimeout(m.ipcClient.Context(), 10*time.Second)
		defer cancel()

		agents, err := m.ipcClient.GetAgents(ctx)
		if err != nil {
			log.IPC("Failed to get agents: %v", err)
			switch err.(type) {
			case *ipc.BadApiStatusError, *ipc.InvalidAgentModelsError:
				return err
			default:
				return AgentsLoadedMsg{Err: m.err}
			}
		}

		log.IPC("Received %d agents", len(agents))
		agents_return := make([]Agent, len(agents))
		for i, agent := range agents {
			agents_return[i] = Agent{
				ID:          agent.Name,
				Name:        agent.Name,
				Description: agent.SystemPrompt,
				Status:      "online",
			}
		}

		// Parse agent list
		return AgentsLoadedMsg{Agents: agents_return}
	}
}

func (m *AgentPickerModel) GetTitle() string {
	return "🤖 Select an Agent"
}

// SetLoading sets the loading state
func (m *AgentPickerModel) SetLoading(loading bool) {
	m.loading = loading
	if loading {
		m.err = nil
	}
}

// SetError sets the error state
func (m *AgentPickerModel) SetError(err error) {
	m.err = err
	m.loading = false
}

// SetSize updates the dimensions
func (m *AgentPickerModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetAgents updates the list of agents
func (m *AgentPickerModel) SetAgents(agents []Agent) {
	m.agents = agents
	if m.cursor >= len(agents) && len(agents) > 0 {
		m.cursor = len(agents) - 1
	}
}

// MarkSelected marks an agent as having an active session
func (m *AgentPickerModel) MarkSelected(agentID string) {
	m.selected[agentID] = true
}

// GetSelectedAgent returns the currently highlighted agent
func (m *AgentPickerModel) GetSelectedAgent() *Agent {
	if len(m.agents) == 0 {
		return nil
	}
	return &m.agents[m.cursor]
}

type selectAgentMsg struct {
	agent *Agent
}

func selectAgent(agent *Agent) tea.Cmd {
	return func() tea.Msg {
		return selectAgentMsg{agent: agent}
	}
}

// Update handles input for the agent picker
func (m *AgentPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(int(msg.Width), int(msg.Height))
		return m, nil
	case AgentsLoadedMsg:
		m.agents = msg.Agents
		m.loading = false
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.selected[m.agents[m.cursor].ID] = true
			return m, tea.Batch(activateScreenObj(&ChatModel{}), selectAgent(&m.agents[m.cursor]))
		case "ctrl+r":
			m.loading = true
			return m, requestAgentsCmd(m.ipcClient)
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.agents)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m *AgentPickerModel) SetScreenIndex(index int) {
	m.screenIndex = index
}

// View renders the agent picker
func (m *AgentPickerModel) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("🤖 Select an Agent"))
	b.WriteString("\n\n")

	// Show loading state
	if m.loading {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA")).
			Bold(true)
		b.WriteString(loadingStyle.Render("⏳ Loading agents..."))
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("Please wait while we connect to the application."))
		return b.String()
	}

	// Show error state
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true)
		b.WriteString(errorStyle.Render("❌ Error loading agents:"))
		b.WriteString("\n\n")
		errorMsgStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F87171"))
		b.WriteString(errorMsgStyle.Render(m.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("Check the Debug tab for more details. • tab switch screen • esc quit"))
		return b.String()
	}

	if len(m.agents) == 0 {
		b.WriteString(HelpStyle.Render("No agents found. Make sure your app is configured correctly."))
		return b.String()
	}

	for i, agent := range m.agents {
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}

		// Status indicator
		statusIcon := "○"
		switch agent.Status {
		case "online":
			statusIcon = "●"
		case "busy":
			statusIcon = "◐"
		}

		// Session indicator
		sessionIndicator := ""
		if m.selected[agent.ID] {
			sessionIndicator = " [active]"
		}

		name := agent.Name
		if i == m.cursor {
			name = SelectedItemStyle.Render(name)
		} else {
			name = NormalItemStyle.Render(name)
		}

		statusText := StatusStyle(agent.Status).Render(statusIcon)
		line := fmt.Sprintf("%s%s %s%s", cursor, statusText, name, sessionIndicator)

		b.WriteString(line)
		b.WriteString("\n")

		// Show description for selected item
		if i == m.cursor && agent.Description != "" {
			desc := HelpStyle.Render("   " + agent.Description)
			b.WriteString(desc)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("↑/↓ navigate • enter select • tab switch screen • q quit"))

	return b.String()
}
