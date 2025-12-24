package tui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

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
	Deltas  []struct {
		Delta          string
		SequenceNumber int
	}
	IsCompleted bool
}

func (c ChatMessage) String() string {
	if c.IsCompleted {
		return c.Content
	}
	deltaStrings := make([]string, len(c.Deltas))
	for i, delta := range c.Deltas {
		deltaStrings[i] = delta.Delta
	}
	return strings.Join(deltaStrings, "")
}

func (c *ChatMessage) Commit() {
	c.IsCompleted = true
	content := ""
	for _, delta := range c.Deltas {
		content += delta.Delta
	}
	c.Content = content
	c.Deltas = nil
}

type ToolCall struct {
	Name      string
	Arguments map[string]any
	CallID    string
}

// ChatSession represents an active chat session with an agent
type ChatSession struct {
	Agent               Agent
	Messages            []*ChatMessage
	currentMessageIndex int
	IsWaiting           bool // True when waiting for AI response after user sends message
}

func (c *ChatSession) UpdateMessage(delta string, index int) {
	if c.currentMessageIndex == -1 {
		c.StartNewMessageDelta()
	}
	msg := c.Messages[c.currentMessageIndex]
	msg.Deltas = append(msg.Deltas, struct {
		Delta          string
		SequenceNumber int
	}{Delta: delta, SequenceNumber: index})
	sort.SliceStable(msg.Deltas, func(i, j int) bool {
		return msg.Deltas[i].SequenceNumber < msg.Deltas[j].SequenceNumber
	})
}

func (c *ChatSession) StartNewMessageDelta() {
	c.IsWaiting = false // We've started receiving content, no longer waiting
	c.Messages = append(c.Messages, &ChatMessage{
		Role:        "agent",
		Content:     "",
		AgentID:     c.Agent.ID,
		IsCompleted: false,
		Deltas: []struct {
			Delta          string
			SequenceNumber int
		}{},
	})
	c.currentMessageIndex = len(c.Messages) - 1
}

func (c *ChatSession) CommitMessage() {
	msg := c.Messages[c.currentMessageIndex]
	msg.Commit()
	c.currentMessageIndex = -1
}

// LogEntry represents a log entry from an agent
type LogEntry struct {
	Timestamp string
	Level     string // "info", "warn", "error", "debug"
	AgentID   string
	Message   string
}
