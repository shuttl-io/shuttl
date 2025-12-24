package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/shuttl-ai/cli/log"
)

// DebugModel handles the debug log view
type DebugModel struct {
	width        int
	height       int
	scrollOffset int
	levelFilter  string
	levels       []string
	autoScroll   bool
	screenIndex  int
}

// NewDebugModel creates a new debug model
func NewDebugModel() *DebugModel {
	return &DebugModel{
		levels:      []string{"all", "DEBUG", "INFO", "WARN", "ERROR", "IPC"},
		autoScroll:  true,
		screenIndex: 0,
	}
}

func (m *DebugModel) SetScreenIndex(index int) {
	m.screenIndex = index
}

func (m *DebugModel) Init() tea.Cmd {
	return nil
}

func (m *DebugModel) GetTitle() string {
	return "🐛 Debug Log"
}

// SetSize updates the dimensions
func (m *DebugModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// GetFilteredEntries returns log entries matching the current filter
func (m DebugModel) GetFilteredEntries() []log.Entry {
	if log.Default == nil {
		return nil
	}

	entries := log.Default.GetEntries()
	if m.levelFilter == "" || m.levelFilter == "all" {
		return entries
	}

	var filtered []log.Entry
	for _, entry := range entries {
		if entry.Level == m.levelFilter {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// Update handles input for the debug view
func (m *DebugModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(int(msg.Width), int(msg.Height))
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k", "wheel down":
			m.scrollOffset++
			m.autoScroll = false
		case "down", "j", "wheel up":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
			if m.scrollOffset == 0 {
				m.autoScroll = true
			}
		case "home", "g":
			entries := m.GetFilteredEntries()
			if len(entries) > m.height-8 {
				m.scrollOffset = len(entries) - (m.height - 8)
			}
			m.autoScroll = false
		case "end", "G":
			m.scrollOffset = 0
			m.autoScroll = true
		case "a":
			m.autoScroll = !m.autoScroll
			if m.autoScroll {
				m.scrollOffset = 0
			}
		case "c":
			if log.Default != nil {
				log.Default.Clear()
			}
			m.scrollOffset = 0
		case "1":
			m.levelFilter = ""
			m.scrollOffset = 0
		case "2":
			m.levelFilter = "DEBUG"
			m.scrollOffset = 0
		case "3":
			m.levelFilter = "INFO"
			m.scrollOffset = 0
		case "4":
			m.levelFilter = "WARN"
			m.scrollOffset = 0
		case "5":
			m.levelFilter = "ERROR"
			m.scrollOffset = 0
		case "6":
			m.levelFilter = "IPC"
			m.scrollOffset = 0

		}
	default:
		log.Debug("Recieved event: %s", msg)
	}

	// Auto-scroll to bottom
	if m.autoScroll {
		m.scrollOffset = 0
	}

	return m, nil
}

// View renders the debug view
func (m *DebugModel) View() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("🐛 Debug Log"))
	b.WriteString("\n\n")

	// Filter bar
	b.WriteString(m.renderFilterBar())
	b.WriteString("\n\n")

	entries := m.GetFilteredEntries()

	if len(entries) == 0 {
		b.WriteString(HelpStyle.Render("No debug logs to display."))
		b.WriteString("\n")
	} else {
		// Calculate visible entries
		visibleHeight := m.height - 10
		if visibleHeight < 5 {
			visibleHeight = 5
		}

		endIdx := len(entries) - m.scrollOffset
		if endIdx > len(entries) {
			endIdx = len(entries)
		}
		if endIdx < 0 {
			endIdx = 0
		}

		startIdx := endIdx - visibleHeight
		if startIdx < 0 {
			startIdx = 0
		}

		for i := startIdx; i < endIdx; i++ {
			entry := entries[i]
			line := m.renderEntry(entry)
			b.WriteString(line)
			b.WriteString("\n")
		}

		// Scroll indicator
		if len(entries) > visibleHeight {
			scrollInfo := fmt.Sprintf("Showing %d-%d of %d", startIdx+1, endIdx, len(entries))
			if m.autoScroll {
				scrollInfo += " (auto-scroll ON)"
			}
			b.WriteString("\n")
			b.WriteString(HelpStyle.Render(scrollInfo))
		}
	}

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("↑/↓ scroll • 1-6 filter level • a auto-scroll • c clear • g/G top/bottom • tab switch • q quit"))

	return b.String()
}

func (m DebugModel) renderFilterBar() string {
	var parts []string

	for i, lvl := range m.levels {
		label := fmt.Sprintf("%d:%s", i+1, lvl)
		if (m.levelFilter == "" && lvl == "all") || m.levelFilter == lvl {
			label = ActiveTabStyle.Render(label)
		} else {
			label = InactiveTabStyle.Render(label)
		}
		parts = append(parts, label)
	}

	return strings.Join(parts, " ")
}

func (m DebugModel) renderEntry(entry log.Entry) string {
	timestamp := LogTimestampStyle.Render(entry.Timestamp.Format("15:04:05.000"))

	var levelStyle lipgloss.Style
	switch entry.Level {
	case "DEBUG":
		levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	case "INFO":
		levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6"))
	case "WARN":
		levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
	case "ERROR":
		levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	case "IPC":
		levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	default:
		levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	}

	level := levelStyle.Render(fmt.Sprintf("[%-5s]", entry.Level))

	// Calculate available width for message (timestamp + level + spaces = ~24 chars)
	prefixWidth := 24
	messageWidth := m.width - prefixWidth
	if messageWidth < 20 {
		messageWidth = 20
	}

	// Wrap the message to fit within the available width
	message := wrapText(entry.Message, messageWidth)

	// If message has multiple lines, indent continuation lines
	lines := strings.Split(message, "\n")
	if len(lines) > 1 {
		indent := strings.Repeat(" ", prefixWidth)
		for i := 1; i < len(lines); i++ {
			lines[i] = indent + lines[i]
		}
		message = strings.Join(lines, "\n")
	}

	return fmt.Sprintf("%s %s %s", timestamp, level, message)
}
