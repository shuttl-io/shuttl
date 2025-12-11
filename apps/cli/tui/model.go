package tui

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shuttl-ai/cli/ipc"
	"github.com/shuttl-ai/cli/log"
)

// TickMsg is sent every second for the countdown
type TickMsg time.Time

// AgentsLoadedMsg is sent when agents are loaded from the IPC client
type AgentsLoadedMsg struct {
	Agents []Agent
	Err    error
}

// IPCOutputMsg is sent when output is received from the IPC client
type IPCOutputMsg struct {
	Output ipc.OutputLine
}

// IPCErrorMsg is sent when an error occurs in the IPC client
type IPCErrorMsg struct {
	Err error
}

// Model is the main TUI model
type Model struct {
	currentScreen      ScreenNumber
	screens            []Screen
	activeScreenIndex  int
	width              int
	height             int
	ipcClient          *ipc.Client
	quitting           bool
	showingWarning     bool
	warningSecondsLeft int
	demoMode           bool
	rootCtx            context.Context
	cancel             context.CancelFunc
}

// NewModel creates a new TUI model
func NewModel(client *ipc.Client) Model {
	agentPicker := NewAgentPickerModel([]Agent{}, client)
	chat := NewChatModel()
	logs := NewLogsModel()
	debug := NewDebugModel()
	rootCtx, cancel := context.WithCancel(context.Background())

	screens := []Screen{
		agentPicker,
		chat,
		logs,
		debug,
	}

	// If no IPC client, show warning first
	showWarning := client == nil

	return Model{
		currentScreen:      ScreenAgentPicker,
		screens:            screens,
		activeScreenIndex:  0,
		ipcClient:          client,
		showingWarning:     showWarning,
		warningSecondsLeft: 10,
		demoMode:           false,
		rootCtx:            rootCtx,
		cancel:             cancel,
	}
}

// loadDemoData loads mock agents and logs for demo mode
func (m *Model) loadDemoData() tea.Cmd {
	agents := []Agent{
		{ID: "agent-1", Name: "General Assistant", Description: "A helpful general-purpose AI assistant", Status: "online"},
		{ID: "agent-2", Name: "Code Helper", Description: "Specialized in code review and programming help", Status: "online"},
		{ID: "agent-3", Name: "Data Analyst", Description: "Helps analyze and visualize data", Status: "busy"},
	}

	// m.agentPicker.SetAgents(agents)
	logs := []LogEntry{}
	// m.logs.SetAgents(agents)

	// // Add some demo logs
	now := time.Now()
	logs = append(logs, LogEntry{
		Timestamp: now.Add(-5 * time.Minute).Format("15:04:05"),
		Level:     "info",
		AgentID:   "agent-1",
		Message:   "Agent started successfully",
	})
	logs = append(logs, LogEntry{
		Timestamp: now.Add(-4 * time.Minute).Format("15:04:05"),
		Level:     "debug",
		AgentID:   "agent-1",
		Message:   "Loading model configuration",
	})
	logs = append(logs, LogEntry{
		Timestamp: now.Add(-3 * time.Minute).Format("15:04:05"),
		Level:     "info",
		AgentID:   "agent-2",
		Message:   "Code Helper agent initialized",
	})
	logs = append(logs, LogEntry{
		Timestamp: now.Add(-2 * time.Minute).Format("15:04:05"),
		Level:     "warn",
		AgentID:   "agent-3",
		Message:   "High memory usage detected",
	})
	logs = append(logs, LogEntry{
		Timestamp: now.Add(-1 * time.Minute).Format("15:04:05"),
		Level:     "error",
		AgentID:   "agent-3",
		Message:   "Failed to connect to external API",
	})
	var commands []tea.Cmd = []tea.Cmd{
		func() tea.Msg { return AgentsLoadedMsg{Agents: agents} },
		func() tea.Msg {
			return LogMessagesBatch{Messages: logs}
		},
	}

	m.demoMode = true
	return tea.Batch(commands...)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// requestAgentsCmd sends a list_agents request to the IPC client and waits for response
func requestAgentsCmd(client *ipc.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil || !client.IsRunning() {
			log.IPC("Cannot request agents: client not running")
			return AgentsLoadedMsg{Err: fmt.Errorf("IPC client not running")}
		}

		// Send the list_agents request
		log.IPC("Sending list_agents request")

		ctx, cancel := context.WithTimeout(client.Context(), 10*time.Second)
		defer cancel()

		agents, err := client.GetAgents(ctx)
		if err != nil {
			log.IPC("Failed to get agents: %v", err)
			return AgentsLoadedMsg{Err: fmt.Errorf("failed to get agents: %w", err)}
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

// Init initializes the model
func (m Model) Init() tea.Cmd {
	if m.showingWarning {
		return tickCmd()
	}

	var cmds []tea.Cmd = []tea.Cmd{
		activateScreen(0),
	}

	for i, screen := range m.screens {
		screen.SetScreenIndex(i)
		cmd := screen.Init()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) == 0 {
		return nil
	}

	return tea.Batch(cmds...)
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case activateScreenMsg:
		m.activeScreenIndex = int(msg)
		return m, nil

	case activateScreenObjMsg:
		obj := msg.obj
		for i, screen := range m.screens {
			if reflect.TypeOf(screen) == reflect.TypeOf(obj) {
				m.activeScreenIndex = i
				return m, activateScreen(i)
			}
		}
		return m, nil

	case TickMsg:
		if m.showingWarning {
			m.warningSecondsLeft--
			if m.warningSecondsLeft <= 0 {
				m.showingWarning = false
				// If we have an IPC client, request agents
				if m.ipcClient != nil && m.ipcClient.IsRunning() {
					return m, requestAgentsCmd(m.ipcClient)
				}
				return m, m.loadDemoData()
			}
		}
		if m.ipcClient != nil && m.ipcClient.IsRunning() {
			log.Debug("IPC client is running, sending tick")
		} else if m.ipcClient != nil && !m.ipcClient.IsRunning() {
			log.Error("Unexpectedly lost IPC client connection, quitting")
			m.quitting = true
			return m, tea.Quit
		}
		return m, tickCmd()

	case tea.KeyMsg:
		// Handle warning screen
		if m.showingWarning {
			switch msg.String() {
			case "ctrl+q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "enter", " ":
				// Skip warning and go to demo mode immediately
				m.showingWarning = false
				m.loadDemoData()
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			m.quitting = true
			m.cancel()
			return m, tea.Quit
		case "esc":
			// Use ESC to quit from any screen
			m.quitting = true
			m.cancel()
			return m, tea.Quit
		case "tab":
			// Cycle through screens
			// m.currentScreen = Screen((int(m.currentScreen) + 1) % 4)
			nextScreenIndex := (m.activeScreenIndex + 1) % len(m.screens)
			return m, activateScreen(nextScreenIndex)
		case "shift+tab":
			// Cycle backwards through screens
			// m.currentScreen = Screen((int(m.currentScreen) + 3) % 4)
			prevScreenIndex := (m.activeScreenIndex - 1 + len(m.screens)) % len(m.screens)
			return m, activateScreen(prevScreenIndex)
		}
	}

	// Don't process screen updates while showing warning
	if m.showingWarning {
		return m, nil
	}

	var cmds []tea.Cmd
	for i, screen := range m.screens {
		_, ok := msg.(tea.KeyMsg)
		if ok && m.activeScreenIndex != i {
			continue
		}
		updatedScreen, cmd := screen.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		m.screens[i] = updatedScreen.(Screen)
	}

	if len(cmds) == 0 {
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m Model) View() string {
	if m.quitting {
		return "👋 Goodbye!\n"
	}

	// Show warning screen if no config
	if m.showingWarning {
		return m.renderWarningScreen()
	}

	var b strings.Builder

	// Header with tabs
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	currentScreen := m.screens[m.activeScreenIndex]
	b.WriteString(currentScreen.View())

	return b.String()
}

func (m Model) renderWarningScreen() string {
	var b strings.Builder

	// Calculate center positioning
	warningBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#F59E0B")).
		Padding(2, 4).
		Align(lipgloss.Center)

	warningIcon := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")).
		Bold(true).
		Render("⚠️  WARNING  ⚠️")

	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")).
		Bold(true).
		MarginTop(1).
		Render("No shuttl.json configuration found!")

	description := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		MarginTop(1).
		Width(50).
		Align(lipgloss.Center).
		Render("The CLI could not find a shuttl.json file in the current directory or any parent directory.")

	configExample := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		MarginTop(1).
		Render("Expected format:\n\n  {\n    \"app\": \"./path/to/your/app\"\n  }")

	countdown := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444")).
		Bold(true).
		MarginTop(2).
		Render(fmt.Sprintf("Entering DEMO MODE in %d seconds...", m.warningSecondsLeft))

	skipHint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		MarginTop(1).
		Render("Press ENTER or SPACE to skip • Press Q to quit")

	demoNote := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
		MarginTop(2).
		Width(50).
		Align(lipgloss.Center).
		Render("Demo mode will load mock agents so you can explore the interface.")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		warningIcon,
		title,
		description,
		configExample,
		countdown,
		skipHint,
		demoNote,
	)

	box := warningBox.Render(content)

	// Center the box on screen
	if m.width > 0 && m.height > 0 {
		boxWidth := lipgloss.Width(box)
		boxHeight := lipgloss.Height(box)

		paddingLeft := (m.width - boxWidth) / 2
		paddingTop := (m.height - boxHeight) / 2

		if paddingLeft < 0 {
			paddingLeft = 0
		}
		if paddingTop < 0 {
			paddingTop = 0
		}

		// Add vertical padding
		for i := 0; i < paddingTop; i++ {
			b.WriteString("\n")
		}

		// Add horizontal padding to each line
		lines := strings.Split(box, "\n")
		for _, line := range lines {
			b.WriteString(strings.Repeat(" ", paddingLeft))
			b.WriteString(line)
			b.WriteString("\n")
		}
	} else {
		b.WriteString(box)
	}

	return b.String()
}

func (m Model) renderHeader() string {
	tabs := []string{}
	for _, screen := range m.screens {
		tabs = append(tabs, screen.GetTitle())
	}
	var renderedTabs []string

	for i, tab := range tabs {
		if i == m.activeScreenIndex {
			renderedTabs = append(renderedTabs, ActiveTabStyle.Render(tab))
		} else {
			renderedTabs = append(renderedTabs, InactiveTabStyle.Render(tab))
		}
	}

	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F9FAFB")).
		Render("⚡ Shuttl Dev")

	appInfo := ""
	if m.ipcClient != nil && m.ipcClient.IsRunning() {
		appInfo = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Render(fmt.Sprintf(" [PID: %d]", m.ipcClient.ProcessID()))
	} else if m.demoMode {
		appInfo = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Render(" [DEMO MODE]")
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center, title, appInfo, "  ", tabBar)

	return header
}

// Run starts the TUI with an IPC client
// If client is nil, the TUI will run in demo mode
// The IPC client will be stopped when the TUI exits
func Run(client *ipc.Client) error {
	log.Info("Starting TUI")
	if client != nil {
		log.IPC("IPC client provided, command: %v", client.Command())
	} else {
		log.Info("No IPC client, will run in demo mode")
	}

	p := tea.NewProgram(
		NewModel(client),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := p.Run()

	// Stop the IPC client when TUI exits
	if client != nil {
		log.Info("Stopping IPC client")
		client.Stop()
	}

	// Print debug log to stdout
	if log.Default != nil && log.Default.Len() > 0 {
		fmt.Println("\n--- Debug Log ---")
		fmt.Print(log.Default.Format())
		fmt.Println("--- End Debug Log ---")
	}

	log.Info("TUI exited")
	return err
}
