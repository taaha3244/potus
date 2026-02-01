package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	Primary   = lipgloss.AdaptiveColor{Light: "#7c3aed", Dark: "#a78bfa"}
	Secondary = lipgloss.AdaptiveColor{Light: "#06b6d4", Dark: "#22d3ee"}
	Success   = lipgloss.AdaptiveColor{Light: "#10b981", Dark: "#34d399"}
	Warning   = lipgloss.AdaptiveColor{Light: "#f59e0b", Dark: "#fbbf24"}
	Error     = lipgloss.AdaptiveColor{Light: "#ef4444", Dark: "#f87171"}
	Muted     = lipgloss.AdaptiveColor{Light: "#64748b", Dark: "#94a3b8"}
	BorderColor    = lipgloss.AdaptiveColor{Light: "#e2e8f0", Dark: "#334155"}

	StatusBar = lipgloss.NewStyle().
			Background(Primary).
			Foreground(lipgloss.Color("#ffffff")).
			Padding(0, 1)

	UserMessage = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			MarginTop(1)

	AssistantMessage = lipgloss.NewStyle().
				Foreground(Secondary).
				Bold(true).
				MarginTop(1)

	ToolCall = lipgloss.NewStyle().
			Foreground(Warning).
			Italic(true)

	ToolResult = lipgloss.NewStyle().
			Foreground(Muted).
			Faint(true)

	ErrorMessage = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	InputPrompt = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	BorderedBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor)

	CodeBlock = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "#f1f5f9", Dark: "#1e293b"}).
			Foreground(lipgloss.AdaptiveColor{Light: "#334155", Dark: "#e2e8f0"}).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	// Diff styles for tool preview
	DiffAdd = lipgloss.NewStyle().
		Foreground(Success)

	DiffRemove = lipgloss.NewStyle().
			Foreground(Error)

	DiffHeader = lipgloss.NewStyle().
			Foreground(Muted).
			Bold(true)

	DiffContext = lipgloss.NewStyle().
			Foreground(Muted)

	ConfirmPrompt = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			Background(lipgloss.AdaptiveColor{Light: "#f1f5f9", Dark: "#1e293b"}).
			Padding(0, 1)

	ConfirmKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"}).
			Background(Primary).
			Padding(0, 1)
)
