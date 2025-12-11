package tui

import tea "github.com/charmbracelet/bubbletea"

// Screen represents the different screens in the TUI
type ScreenNumber int

type activateScreenMsg int

func activateScreen(index int) tea.Cmd {
	return func() tea.Msg {
		return activateScreenMsg(index)
	}
}

type activateScreenObjMsg struct {
	obj Screen
}

func activateScreenObj(obj Screen) tea.Cmd {
	return func() tea.Msg {
		return activateScreenObjMsg{obj: obj}
	}
}

type Screen interface {
	tea.Model
	GetTitle() string
	SetScreenIndex(index int)
}

const (
	ScreenAgentPicker ScreenNumber = iota
	ScreenChat
	ScreenLogs
	ScreenDebug
)

// Agent represents an AI agent that can be chatted with
type Agent struct {
	ID          string
	Name        string
	Description string
	Status      string // "online", "offline", "busy"
}

// ChatMessage represents a message in a chat session
type ChatMessage struct {
	Role    string // "user" or "assistant"
	Content string
	AgentID string
}

// ChatSession represents an active chat session with an agent
type ChatSession struct {
	Agent    Agent
	Messages []ChatMessage
}

// LogEntry represents a log entry from an agent
type LogEntry struct {
	Timestamp string
	Level     string // "info", "warn", "error", "debug"
	AgentID   string
	Message   string
}
