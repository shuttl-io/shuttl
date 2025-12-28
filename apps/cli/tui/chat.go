package tui

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shuttl-ai/cli/ipc"
	"github.com/shuttl-ai/cli/log"
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
	ipcClient     *ipc.Client

	// File picker state
	filePicker      filepicker.Model
	showFilePicker  bool
	attachedFiles   []ipc.FileAttachment
	filePickerError string
}

// NewChatModel creates a new chat model
func NewChatModel(ipcClient *ipc.Client) *ChatModel {
	// Initialize file picker
	fp := filepicker.New()
	fp.CurrentDirectory, _ = os.Getwd()
	fp.ShowHidden = false
	fp.ShowPermissions = false
	fp.ShowSize = true
	fp.Height = 15

	return &ChatModel{
		sessions:      make(map[string]*ChatSession),
		agentOrder:    []string{},
		screenIndex:   0,
		ipcClient:     ipcClient,
		filePicker:    fp,
		attachedFiles: []ipc.FileAttachment{},
	}
}

func (m *ChatModel) SetScreenIndex(index int) {
	m.screenIndex = index
}

func (m *ChatModel) Init() tea.Cmd {
	return m.filePicker.Init()
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
			Agent:               agent,
			Messages:            []*ChatMessage{},
			currentMessageIndex: -1,
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
		session.Messages = append(session.Messages, &ChatMessage{
			Role:        role,
			Content:     content,
			AgentID:     m.activeAgentID,
			IsCompleted: true,
		})
	}
}

// ClearInput clears the input field
func (m *ChatModel) ClearInput() {
	m.input = ""
	m.cursorPos = 0
}

// ClearAttachments clears all attached files
func (m *ChatModel) ClearAttachments() {
	m.attachedFiles = []ipc.FileAttachment{}
}

// AttachFile reads a file and adds it to the attachments
func (m *ChatModel) AttachFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Determine MIME type based on extension
	mimeType := getMimeType(filePath)

	attachment := ipc.FileAttachment{
		Name:     filepath.Base(filePath),
		Path:     filePath,
		Content:  base64.StdEncoding.EncodeToString(content),
		MimeType: mimeType,
	}

	m.attachedFiles = append(m.attachedFiles, attachment)
	return nil
}

// RemoveAttachment removes an attachment by index
func (m *ChatModel) RemoveAttachment(index int) {
	if index >= 0 && index < len(m.attachedFiles) {
		m.attachedFiles = append(m.attachedFiles[:index], m.attachedFiles[index+1:]...)
	}
}

// getMimeType returns the MIME type based on file extension
// Uses Go's mime package with fallbacks for programming language files
func getMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Custom mappings for programming languages and formats not in Go's mime package
	customMimeTypes := map[string]string{
		".ts":    "application/typescript",
		".tsx":   "application/typescript",
		".jsx":   "application/javascript",
		".go":    "text/x-go",
		".py":    "text/x-python",
		".rb":    "text/x-ruby",
		".java":  "text/x-java",
		".c":     "text/x-c",
		".h":     "text/x-c",
		".cpp":   "text/x-c++",
		".hpp":   "text/x-c++",
		".cc":    "text/x-c++",
		".rs":    "text/x-rust",
		".swift": "text/x-swift",
		".kt":    "text/x-kotlin",
		".scala": "text/x-scala",
		".r":     "text/x-r",
		".lua":   "text/x-lua",
		".pl":    "text/x-perl",
		".php":   "text/x-php",
		".cs":    "text/x-csharp",
		".fs":    "text/x-fsharp",
		".hs":    "text/x-haskell",
		".elm":   "text/x-elm",
		".ex":    "text/x-elixir",
		".exs":   "text/x-elixir",
		".erl":   "text/x-erlang",
		".clj":   "text/x-clojure",
		".ml":    "text/x-ocaml",
		".vim":   "text/x-vim",
		".sql":   "application/sql",
		".md":    "text/markdown",
		".yaml":  "application/x-yaml",
		".yml":   "application/x-yaml",
		".toml":  "application/toml",
		".ini":   "text/plain",
		".cfg":   "text/plain",
		".conf":  "text/plain",
		".env":   "text/plain",
		".sh":    "application/x-sh",
		".bash":  "application/x-sh",
		".zsh":   "application/x-sh",
		".fish":  "application/x-sh",
		".ps1":   "application/x-powershell",
		".bat":   "application/x-bat",
		".cmd":   "application/x-bat",
	}

	// Check custom mappings first
	if mimeType, ok := customMimeTypes[ext]; ok {
		return mimeType
	}

	// Use Go's mime package for standard types
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		// Strip charset parameter if present (e.g., "text/plain; charset=utf-8" -> "text/plain")
		if idx := strings.Index(mimeType, ";"); idx != -1 {
			mimeType = strings.TrimSpace(mimeType[:idx])
		}
		return mimeType
	}

	// Default fallback
	return "application/octet-stream"
}

// GetInput returns the current input
func (m *ChatModel) GetInput() string {
	return m.input
}

type chatStreamMsg struct {
	currentMessage *ipc.ChatParsedResult
	error          error
	channel        chan *ipc.ChatParsedResult
	errChan        chan error
	agentID        string
}

type endChatStreamMsg struct {
}

type toolCallMsg struct {
	toolCall *ipc.ToolCallResult
}

func streamChat(channel chan *ipc.ChatParsedResult, errChan chan error, agentID string) tea.Cmd {
	return func() tea.Msg {
		currentMessage, ok := <-channel
		if !ok {
			return endChatStreamMsg{}
		}
		var err error
		select {
		case err = <-errChan:
		default:
			err = nil
		}
		valu := chatStreamMsg{
			currentMessage: currentMessage,
			error:          err,
			channel:        channel,
			errChan:        errChan,
			agentID:        agentID,
		}
		return valu
	}
}

func updateMessage(textDelta *ipc.TextDeltaResult) tea.Cmd {
	return func() tea.Msg {
		return *textDelta
	}
}

// Update handles input for the chat
func (m *ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle file picker mode first
	if m.showFilePicker {
		return m.updateFilePicker(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(int(msg.Width), int(msg.Height))
		m.filePicker.Height = m.height - 10
		return m, nil

	case ChatMessage:
		m.AddMessage(msg.Role, msg.Content)
		if msg.Role == "user" {
			// Set waiting state before starting the chat
			session := m.GetActiveSession()
			if session != nil {
				session.IsWaiting = true
			}
			channel, errChan := m.ipcClient.StartChatWithAttachments(context.Background(), m.activeAgentID, msg.Content, m.attachedFiles)
			m.ClearAttachments()
			return m, streamChat(channel, errChan, m.activeAgentID)
		}
		return m, nil

	case ipc.TextDeltaResult:
		m.GetActiveSession().UpdateMessage(msg.OutputTextDelta.Delta, msg.OutputTextDelta.SequenceNumber)
		return m, nil

	case chatStreamMsg:
		session := m.GetActiveSession()
		if session == nil {
			log.Error("No active session")
			return m, nil
		}

		if msg.currentMessage.Type == "output_text_delta" {
			return m, tea.Batch(
				updateMessage(msg.currentMessage.TextDelta),
				streamChat(msg.channel, msg.errChan, msg.agentID),
			)
		}
		if msg.currentMessage.Type == "output_text" {
			session.CommitMessage()
		}

		return m, streamChat(msg.channel, msg.errChan, msg.agentID)
		// for result := range msg.channel {
		// 	if result.Status != nil && result.Status.Status == "completed" {
		// 		return m, nil
		// 	}
		// 	m.AddMessage("assistant", result.FinalOutput.OutputText.Text)
		// }
		// return m, nil
	case selectAgentMsg:
		m.StartSession(*msg.agent)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+f":
			// Open file picker
			if m.GetActiveSession() != nil {
				m.showFilePicker = true
				m.filePickerError = ""
				return m, m.filePicker.Init()
			}
		case "ctrl+x":
			// Remove last attached file
			if len(m.attachedFiles) > 0 {
				m.RemoveAttachment(len(m.attachedFiles) - 1)
			}
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

// updateFilePicker handles updates when the file picker is shown
func (m *ChatModel) updateFilePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+f":
			// Close file picker
			m.showFilePicker = false
			m.filePickerError = ""
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.filePicker, cmd = m.filePicker.Update(msg)

	// Check if a file was selected
	if didSelect, path := m.filePicker.DidSelectFile(msg); didSelect {
		err := m.AttachFile(path)
		if err != nil {
			m.filePickerError = err.Error()
		} else {
			m.showFilePicker = false
			m.filePickerError = ""
		}
		return m, nil
	}

	// Check if a disabled file was selected (e.g., directory)
	if didSelect, path := m.filePicker.DidSelectDisabledFile(msg); didSelect {
		m.filePickerError = fmt.Sprintf("Cannot attach: %s", path)
		return m, cmd
	}

	return m, cmd
}

// View renders the chat interface
func (m *ChatModel) View() string {
	// Show file picker if active
	if m.showFilePicker {
		return m.renderFilePicker()
	}

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
			// content := wrapText(prefix+msg.Content, maxWidth)
			b.WriteString(style.Width(maxWidth).Render(prefix + msg.String()))
			b.WriteString("\n")
		}
	}

	// Loading indicator
	if session.IsWaiting {
		b.WriteString(LoadingStyle.Render("⏳ Thinking..."))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Show attached files
	if len(m.attachedFiles) > 0 {
		b.WriteString(AttachedFileStyle.Render("📎 Attached files:"))
		b.WriteString("\n")
		for i, file := range m.attachedFiles {
			b.WriteString(AttachedFileStyle.Render(fmt.Sprintf("   %d. %s", i+1, file.Name)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

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
	helpText := "enter send • ctrl+f attach file • ctrl+x remove file • ctrl+n/p switch agent • tab switch screen • esc quit"
	b.WriteString(HelpStyle.Render(helpText))

	return b.String()
}

// renderFilePicker renders the file picker overlay
func (m *ChatModel) renderFilePicker() string {
	var b strings.Builder

	b.WriteString(FilePickerTitleStyle.Render("📁 Select a file to attach"))
	b.WriteString("\n\n")

	b.WriteString(m.filePicker.View())

	if m.filePickerError != "" {
		b.WriteString("\n")
		b.WriteString(LogErrorStyle.Render("⚠️  " + m.filePickerError))
	}

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("enter select • esc cancel • ↑/↓ navigate"))

	return FilePickerStyle.Width(m.width - 4).Render(b.String())
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
