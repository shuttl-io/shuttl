package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#10B981")
	accentColor    = lipgloss.Color("#F59E0B")
	errorColor     = lipgloss.Color("#EF4444")
	mutedColor     = lipgloss.Color("#6B7280")
	bgColor        = lipgloss.Color("#1F2937")
	fgColor        = lipgloss.Color("#F9FAFB")

	// Base styles
	BaseStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(fgColor)

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	// Tab styles
	ActiveTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(fgColor).
			Background(primaryColor).
			Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(mutedColor).
				Padding(0, 2)

	// List styles
	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(fgColor).
				Background(primaryColor).
				Bold(true).
				Padding(0, 1)

	NormalItemStyle = lipgloss.NewStyle().
			Foreground(fgColor).
			Padding(0, 1)

	// Chat styles
	UserMessageStyle = lipgloss.NewStyle().
				Foreground(fgColor).
				Background(lipgloss.Color("#374151")).
				Padding(0, 1).
				MarginBottom(1)

	AssistantMessageStyle = lipgloss.NewStyle().
				Foreground(fgColor).
				Background(primaryColor).
				Padding(0, 1).
				MarginBottom(1)

	// Input styles
	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)

	InputPromptStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	// Log styles
	LogInfoStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	LogWarnStyle = lipgloss.NewStyle().
			Foreground(accentColor)

	LogErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	LogDebugStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	LogTimestampStyle = lipgloss.NewStyle().
				Foreground(mutedColor)

	// Status styles
	OnlineStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	OfflineStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	BusyStyle = lipgloss.NewStyle().
			Foreground(accentColor)

	// Help styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	// Loading indicator style
	LoadingStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Italic(true)

	// Box styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2)
)

func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "online":
		return OnlineStyle
	case "offline":
		return OfflineStyle
	case "busy":
		return BusyStyle
	default:
		return lipgloss.NewStyle().Foreground(mutedColor)
	}
}

func LogLevelStyle(level string) lipgloss.Style {
	switch level {
	case "info":
		return LogInfoStyle
	case "warn":
		return LogWarnStyle
	case "error":
		return LogErrorStyle
	case "debug":
		return LogDebugStyle
	default:
		return lipgloss.NewStyle()
	}
}
