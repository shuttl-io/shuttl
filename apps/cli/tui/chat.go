package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ChatModel handles the chat interface
type ChatModel struct {
	sessions      map[string]*ChatSession // Map of agent ID to chat session
	activeAgentID string                  // Currently active agent
	agentOrder    []string                // Order of agents (for tab switching)
	input         string
	cursorPos     int
	width         int
	height        int
	scrollOffset  int
	screenIndex   int
}

// NewChatModel creates a new chat model
func NewChatModel() *ChatModel {
	return &ChatModel{
		sessions:    make(map[string]*ChatSession),
		agentOrder:  []string{},
		screenIndex: 0,
	}
}

func (m *ChatModel) SetScreenIndex(index int) {
	m.screenIndex = index
}

func (m *ChatModel) Init() tea.Cmd {
	return nil
}

func (m *ChatModel) GetTitle() string {
	return "💬 Chat with an Agent"
}

// SetSize updates the dimensions
func (m *ChatModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// StartSession starts a new chat session with an agent
func (m *ChatModel) StartSession(agent Agent) {
	if _, exists := m.sessions[agent.ID]; !exists {
		m.sessions[agent.ID] = &ChatSession{
			Agent:    agent,
			Messages: []ChatMessage{},
		}
		m.agentOrder = append(m.agentOrder, agent.ID)
	}
	m.activeAgentID = agent.ID
}

// HasSessions returns true if there are active sessions
func (m ChatModel) HasSessions() bool {
	return len(m.sessions) > 0
}

// GetActiveSession returns the currently active session
func (m *ChatModel) GetActiveSession() *ChatSession {
	if m.activeAgentID == "" {
		return nil
	}
	return m.sessions[m.activeAgentID]
}

// SwitchToNextAgent switches to the next agent session
func (m *ChatModel) SwitchToNextAgent() {
	if len(m.agentOrder) <= 1 {
		return
	}
	for i, id := range m.agentOrder {
		if id == m.activeAgentID {
			nextIdx := (i + 1) % len(m.agentOrder)
			m.activeAgentID = m.agentOrder[nextIdx]
			m.scrollOffset = 0
			return
		}
	}
}

// SwitchToPrevAgent switches to the previous agent session
func (m *ChatModel) SwitchToPrevAgent() {
	if len(m.agentOrder) <= 1 {
		return
	}
	for i, id := range m.agentOrder {
		if id == m.activeAgentID {
			prevIdx := (i - 1 + len(m.agentOrder)) % len(m.agentOrder)
			m.activeAgentID = m.agentOrder[prevIdx]
			m.scrollOffset = 0
			return
		}
	}
}

// AddMessage adds a message to the current session
func (m *ChatModel) AddMessage(role, content string) {
	session := m.GetActiveSession()
	if session != nil {
		session.Messages = append(session.Messages, ChatMessage{
			Role:    role,
			Content: content,
			AgentID: m.activeAgentID,
		})
	}
}

// ClearInput clears the input field
func (m *ChatModel) ClearInput() {
	m.input = ""
	m.cursorPos = 0
}

// GetInput returns the current input
func (m *ChatModel) GetInput() string {
	return m.input
}

// Update handles input for the chat
func (m *ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(int(msg.Width), int(msg.Height))
		return m, nil

	case ChatMessage:
		m.AddMessage(msg.Role, msg.Content)
		if msg.Role == "user" {
			return m, func() tea.Msg {
				return ChatMessage{Role: "assistant", Content: "This is a simulated response from the agent.", AgentID: m.activeAgentID}
			}
		}
		return m, nil

	case selectAgentMsg:
		m.StartSession(*msg.agent)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "left":
			if m.cursorPos > 0 {
				m.cursorPos--
			}
		case "right":
			if m.cursorPos < len(m.input) {
				m.cursorPos++
			}
		case "backspace":
			if m.cursorPos > 0 {
				m.input = m.input[:m.cursorPos-1] + m.input[m.cursorPos:]
				m.cursorPos--
			}
		case "delete":
			if m.cursorPos < len(m.input) {
				m.input = m.input[:m.cursorPos] + m.input[m.cursorPos+1:]
			}
		case "ctrl+a":
			m.cursorPos = 0
		case "ctrl+e":
			m.cursorPos = len(m.input)
		case "ctrl+n":
			m.SwitchToNextAgent()
		case "ctrl+p":
			m.SwitchToPrevAgent()
		case "pgup":
			m.scrollOffset += 5
		case "pgdown":
			if m.scrollOffset > 0 {
				m.scrollOffset -= 5
				if m.scrollOffset < 0 {
					m.scrollOffset = 0
				}
			}

		case "enter":
			// m.AddMessage("user", m.input)
			input := m.input
			m.ClearInput()
			return m, func() tea.Msg {
				return ChatMessage{Role: "user", Content: input, AgentID: m.activeAgentID}
			}
		default:
			// Handle regular character input
			if len(msg.String()) == 1 {
				m.input = m.input[:m.cursorPos] + msg.String() + m.input[m.cursorPos:]
				m.cursorPos++
			}
		}
	}
	return m, nil
}

// View renders the chat interface
func (m *ChatModel) View() string {
	var b strings.Builder

	// Tab bar for multiple sessions
	if len(m.sessions) > 0 {
		var tabs []string
		for _, id := range m.agentOrder {
			session := m.sessions[id]
			if id == m.activeAgentID {
				tabs = append(tabs, ActiveTabStyle.Render(session.Agent.Name))
			} else {
				tabs = append(tabs, InactiveTabStyle.Render(session.Agent.Name))
			}
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
		b.WriteString("\n\n")
	}

	session := m.GetActiveSession()
	if session == nil {
		b.WriteString(HelpStyle.Render("No active chat session. Select an agent from the Agents tab."))
		b.WriteString("\n\n")
		b.WriteString(HelpStyle.Render("tab switch screen • esc quit"))
		return b.String()
	}

	b.WriteString(TitleStyle.Render(fmt.Sprintf("💬 Chat with %s", session.Agent.Name)))
	b.WriteString("\n\n")

	// Messages area
	if len(session.Messages) == 0 {
		b.WriteString(HelpStyle.Render("Start a conversation by typing a message below."))
		b.WriteString("\n")
	} else {
		// Calculate visible messages based on scroll
		visibleHeight := m.height - 10 // Reserve space for UI elements
		if visibleHeight < 5 {
			visibleHeight = 5
		}

		messages := session.Messages
		startIdx := 0
		if len(messages) > visibleHeight-m.scrollOffset {
			startIdx = len(messages) - visibleHeight + m.scrollOffset
			if startIdx < 0 {
				startIdx = 0
			}
		}

		endIdx := len(messages) - m.scrollOffset
		if endIdx > len(messages) {
			endIdx = len(messages)
		}
		if endIdx < 0 {
			endIdx = 0
		}

		for i := startIdx; i < endIdx; i++ {
			msg := messages[i]
			var style lipgloss.Style
			var prefix string

			if msg.Role == "user" {
				style = UserMessageStyle
				prefix = "You: "
			} else {
				style = AssistantMessageStyle
				prefix = fmt.Sprintf("%s: ", session.Agent.Name)
			}

			// Word wrap the message
			maxWidth := m.width - 10
			if maxWidth < 20 {
				maxWidth = 20
			}
			content := wrapText(prefix+msg.Content, maxWidth)
			b.WriteString(style.Render(content))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// Input area
	inputDisplay := m.input
	if m.cursorPos <= len(inputDisplay) {
		// Show cursor
		before := inputDisplay[:m.cursorPos]
		after := ""
		if m.cursorPos < len(inputDisplay) {
			after = inputDisplay[m.cursorPos:]
		}
		inputDisplay = before + "█" + after
	}

	prompt := InputPromptStyle.Render("> ")
	inputBox := InputStyle.Width(m.width - 6).Render(inputDisplay)
	b.WriteString(prompt + inputBox)

	b.WriteString("\n\n")

	// Help
	helpText := "enter send • ctrl+n/p switch agent • pgup/pgdn scroll • tab switch screen • esc quit"
	b.WriteString(HelpStyle.Render(helpText))

	return b.String()
}

// wrapText wraps text to a specified width
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
	lineLen := 0

	for i, word := range words {
		if i > 0 {
			if lineLen+1+len(word) > width {
				result.WriteString("\n")
				lineLen = 0
			} else {
				result.WriteString(" ")
				lineLen++
			}
		}
		result.WriteString(word)
		lineLen += len(word)
	}

	return result.String()
}
