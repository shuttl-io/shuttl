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

type WarningScreenDetails struct {
	Title       string
	Description string
	Action      string
}

// DisconnectionWarning is shown when the IPC client unexpectedly loses connection
type DisconnectionWarning struct {
	Reason string
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
	currentScreen        ScreenNumber
	screens              []Screen
	activeScreenIndex    int
	width                int
	height               int
	ipcClient            *ipc.Client
	quitting             bool
	showingWarning       bool
	warningSecondsLeft   int
	demoMode             bool
	rootCtx              context.Context
	cancel               context.CancelFunc
	warningScreenDetails *WarningScreenDetails
	disconnectionWarning *DisconnectionWarning
}

// NewModel creates a new TUI model
func NewModel(client *ipc.Client) Model {
	agentPicker := NewAgentPickerModel([]Agent{}, client)
	chat := NewChatModel(client)
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
		currentScreen:        ScreenAgentPicker,
		screens:              screens,
		activeScreenIndex:    0,
		ipcClient:            client,
		showingWarning:       showWarning,
		warningSecondsLeft:   10,
		demoMode:             false,
		rootCtx:              rootCtx,
		cancel:               cancel,
		warningScreenDetails: nil,
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
	case *ipc.BadApiStatusError:
		m.showingWarning = true
		m.warningSecondsLeft = 30
		m.warningScreenDetails = &WarningScreenDetails{
			Title:       fmt.Sprintf("API returned status %d: %s", msg.StatusCode, msg.Body),
			Description: "The API returned a status code that was not 200. Please check your API URL and try again.",
		}
		return m, tickCmd()
	case *ipc.InvalidAgentModelsError:
		m.showingWarning = true
		m.warningSecondsLeft = 10
		availableModels := []string{}
		for _, model := range msg.AvailableModels {
			availableModels = append(availableModels, model.Name)
		}
		m.warningScreenDetails = &WarningScreenDetails{
			Title:       fmt.Sprintf("Agents have invalid models: %v", msg.Agents),
			Description: "The agents have invalid models. Please check your API URL and try again.",
			Action:      "Available models: " + strings.Join(availableModels, ", "),
		}
		return m, tickCmd()
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
				m.warningScreenDetails = nil
				// If we have an IPC client, request agents
				if m.ipcClient != nil && m.ipcClient.IsRunning() {
					return m, requestAgentsCmd(m.ipcClient)
				}
				return m, m.loadDemoData()
			}
		}
		if m.ipcClient != nil && m.ipcClient.IsRunning() {
			log.Debug("IPC client is running, sending tick")
		} else if m.ipcClient != nil && !m.ipcClient.IsRunning() && m.disconnectionWarning == nil {
			log.Error("Unexpectedly lost IPC client connection")
			m.disconnectionWarning = &DisconnectionWarning{
				Reason: "The child process has unexpectedly terminated.",
			}
		}
		return m, tickCmd()

	case tea.KeyMsg:
		// Handle disconnection warning - only allow quit
		if m.disconnectionWarning != nil {
			switch msg.String() {
			case "ctrl+q", "ctrl+c", "q", "enter", " ", "esc":
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

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

	// Don't process screen updates while showing warning or disconnection
	if m.showingWarning || m.disconnectionWarning != nil {
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

	// Show disconnection warning if connection was lost
	if m.disconnectionWarning != nil {
		return m.renderDisconnectionWarning()
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
	warningScreenDetails := m.warningScreenDetails
	if warningScreenDetails == nil {
		warningScreenDetails = &WarningScreenDetails{
			Title:       "No shuttl.json configuration found!",
			Description: "The CLI could not find a shuttl.json file in the current directory or any parent directory.",
		}
	}

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
		Render(warningScreenDetails.Title)

	description := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		MarginTop(1).
		Width(50).
		Align(lipgloss.Center).
		Render(warningScreenDetails.Description)
	// if m.warningScreenDetails.Action != "" {
	configExample := ""
	configExampleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280")).
		MarginTop(1)
	if warningScreenDetails.Action == "" {
		configExample = configExampleStyle.Render("Expected format:\n\n  {\n    \"app\": \"./path/to/your/app\"\n  }")
	} else {
		configExample = configExampleStyle.Render(warningScreenDetails.Action)
	}

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

func (m Model) renderDisconnectionWarning() string {
	var b strings.Builder

	// Large, dramatic disconnection warning
	warningBox := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#EF4444")).
		Padding(3, 6).
		Align(lipgloss.Center)

	// Large skull/warning icon block
	dangerIcon := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444")).
		Bold(true).
		Render("💀  CONNECTION LOST  💀")

	// Big prominent title
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444")).
		Bold(true).
		MarginTop(2).
		Render("╔══════════════════════════════════════════╗\n" +
			"║      CHILD PROCESS DISCONNECTED!        ║\n" +
			"╚══════════════════════════════════════════╝")

	// Reason
	reason := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FBBF24")).
		MarginTop(1).
		Bold(true).
		Render(m.disconnectionWarning.Reason)

	// Description
	description := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		MarginTop(2).
		Width(55).
		Align(lipgloss.Center).
		Render("The CLI has lost communication with the Shuttl application.\n" +
			"This may have been caused by a crash, timeout, or the process being killed.")

	// What to do
	whatToDo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#60A5FA")).
		MarginTop(2).
		Width(55).
		Align(lipgloss.Center).
		Render("Please check your application logs and restart the CLI.")

	// Exit hint
	exitHint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F9FAFB")).
		Background(lipgloss.Color("#EF4444")).
		Bold(true).
		MarginTop(2).
		Padding(0, 2).
		Render("Press Q, ENTER, or ESC to exit")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		dangerIcon,
		title,
		reason,
		description,
		whatToDo,
		exitHint,
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
	log.Default.SetMode(log.LogToEntries)
	defer log.Default.SetMode(log.LogToConsole)
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
