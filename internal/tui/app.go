package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/taaha3244/potus/internal/agent"
	"github.com/taaha3244/potus/internal/tui/styles"
)

type Model struct {
	agent          *agent.Agent
	viewport       viewport.Model
	textarea       textarea.Model
	messages       []Message
	width          int
	height         int
	ready          bool
	err            error
	status         StatusInfo
	eventChan      <-chan agent.Event
	processingMsg  bool
	confirmChan    chan<- agent.Decision
	pendingPreview bool
}

type Message struct {
	Role    string
	Content string
}

type StatusInfo struct {
	Model         string
	Tokens        int
	MaxTokens     int
	UsagePercent  float64
	Cost          float64
	ElapsedTime   string
	ContextStatus string
	AtWarning     bool
}

type AgentEventMsg struct {
	Event agent.Event
}

type errMsg struct {
	err error
}

type continueStreamMsg struct{}

type startStreamMsg struct {
	events <-chan agent.Event
}

func New(ag *agent.Agent, model string, confirmChan chan<- agent.Decision) Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Ctrl+C to quit)"
	ta.Focus()
	ta.CharLimit = 0
	ta.SetWidth(100)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(100, 20)

	return Model{
		agent:       ag,
		viewport:    vp,
		textarea:    ta,
		messages:    make([]Message, 0),
		confirmChan: confirmChan,
		status: StatusInfo{
			Model: model,
		},
	}
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-6)
			m.textarea.SetWidth(msg.Width - 4)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 6
			m.textarea.SetWidth(msg.Width - 4)
		}

	case tea.KeyMsg:
		// Handle confirmation keypresses when preview is pending
		if m.pendingPreview {
			switch msg.String() {
			case "y", "Y":
				if m.confirmChan != nil {
					m.confirmChan <- agent.DecisionApprove
				}
				m.pendingPreview = false
				m.messages = append(m.messages, Message{
					Role:    "system",
					Content: "Approved",
				})
				m.updateViewport()
				return m, m.readNextEvent()
			case "n", "N":
				if m.confirmChan != nil {
					m.confirmChan <- agent.DecisionDeny
				}
				m.pendingPreview = false
				m.messages = append(m.messages, Message{
					Role:    "system",
					Content: "Denied",
				})
				m.updateViewport()
				return m, m.readNextEvent()
			case "a", "A":
				if m.confirmChan != nil {
					m.confirmChan <- agent.DecisionAlwaysAllow
				}
				m.pendingPreview = false
				m.messages = append(m.messages, Message{
					Role:    "system",
					Content: "Always allowed (saved to .potus/settings.json)",
				})
				m.updateViewport()
				return m, m.readNextEvent()
			case "ctrl+c", "esc":
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			if m.processingMsg {
				return m, nil
			}

			userInput := m.textarea.Value()
			if strings.TrimSpace(userInput) == "" {
				return m, nil
			}

			m.messages = append(m.messages, Message{
				Role:    "user",
				Content: userInput,
			})

			m.textarea.Reset()
			m.updateViewport()
			m.processingMsg = true

			return m, m.processUserMessage(userInput)
		}

	case startStreamMsg:
		m.eventChan = msg.events
		return m, m.readNextEvent()

	case AgentEventMsg:
		return m.handleAgentEvent(msg.Event)

	case continueStreamMsg:
		if m.eventChan != nil {
			return m, m.readNextEvent()
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		m.processingMsg = false
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	statusBar := m.renderStatusBar()
	chatView := m.viewport.View()

	var inputView string
	if m.pendingPreview {
		inputView = m.renderConfirmPrompt()
	} else {
		inputView = m.renderInput()
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		statusBar,
		chatView,
		inputView,
	)
}

func (m Model) renderConfirmPrompt() string {
	yKey := styles.ConfirmKey.Render("y")
	nKey := styles.ConfirmKey.Render("n")
	aKey := styles.ConfirmKey.Render("a")

	prompt := fmt.Sprintf("  %s approve  %s deny  %s always allow", yKey, nKey, aKey)
	return styles.ConfirmPrompt.Width(m.width).Render(prompt)
}

func (m Model) renderStatusBar() string {
	left := fmt.Sprintf("POTUS | %s", m.status.Model)

	var tokenDisplay string
	if m.status.MaxTokens > 0 {
		var tokenColor lipgloss.AdaptiveColor
		if m.status.UsagePercent >= 90 {
			tokenColor = styles.Error
		} else if m.status.UsagePercent >= 80 {
			tokenColor = styles.Warning
		} else {
			tokenColor = styles.Success
		}

		tokenStyle := lipgloss.NewStyle().Foreground(tokenColor)
		tokenDisplay = tokenStyle.Render(fmt.Sprintf("%d/%d (%.1f%%)",
			m.status.Tokens,
			m.status.MaxTokens,
			m.status.UsagePercent))
	} else {
		tokenDisplay = fmt.Sprintf("%d", m.status.Tokens)
	}

	right := fmt.Sprintf("Tokens: %s | Cost: $%.4f", tokenDisplay, m.status.Cost)

	if m.status.ContextStatus != "" {
		statusStyle := lipgloss.NewStyle().Foreground(styles.Warning)
		right += fmt.Sprintf(" | %s", statusStyle.Render(m.status.ContextStatus))
	}

	if m.status.ElapsedTime != "" {
		right += fmt.Sprintf(" | %s", m.status.ElapsedTime)
	}

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := m.width - leftWidth - rightWidth - 2
	if gap < 1 {
		gap = 1
	}

	bar := left + strings.Repeat(" ", gap) + right

	return styles.StatusBar.Width(m.width).Render(bar)
}

func (m Model) renderInput() string {
	prompt := styles.InputPrompt.Render("> ")
	input := m.textarea.View()

	return lipgloss.JoinHorizontal(lipgloss.Top, prompt, input)
}

func (m *Model) updateViewport() {
	var content strings.Builder

	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			content.WriteString(styles.UserMessage.Render("You: ") + msg.Content + "\n\n")
		case "assistant":
			content.WriteString(styles.AssistantMessage.Render("POTUS: ") + msg.Content + "\n\n")
		case "tool_call":
			content.WriteString(styles.ToolCall.Render("-> " + msg.Content) + "\n")
		case "tool_result":
			content.WriteString(styles.ToolResult.Render("   " + msg.Content) + "\n\n")
		case "tool_preview":
			content.WriteString(m.renderDiff(msg.Content) + "\n")
		case "system":
			systemStyle := lipgloss.NewStyle().Foreground(styles.Muted).Italic(true)
			content.WriteString(systemStyle.Render("[System] "+msg.Content) + "\n\n")
		case "error":
			content.WriteString(styles.ErrorMessage.Render("Error: " + msg.Content) + "\n\n")
		}
	}

	m.viewport.SetContent(content.String())
	m.viewport.GotoBottom()
}

func (m *Model) renderDiff(content string) string {
	var result strings.Builder
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			result.WriteString(styles.DiffHeader.Render(line) + "\n")
		} else if strings.HasPrefix(line, "+") {
			result.WriteString(styles.DiffAdd.Render(line) + "\n")
		} else if strings.HasPrefix(line, "-") {
			result.WriteString(styles.DiffRemove.Render(line) + "\n")
		} else if strings.HasPrefix(line, "@@") {
			result.WriteString(styles.DiffHeader.Render(line) + "\n")
		} else if strings.HasPrefix(line, " ") {
			result.WriteString(styles.DiffContext.Render(line) + "\n")
		} else if strings.HasPrefix(line, "Tool:") {
			result.WriteString(styles.ToolCall.Render(line) + "\n")
		} else if strings.HasPrefix(line, "$") {
			// Bash command preview
			result.WriteString(styles.ToolCall.Render(line) + "\n")
		} else {
			result.WriteString(line + "\n")
		}
	}

	return result.String()
}

func (m Model) processUserMessage(input string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		events, err := m.agent.ProcessMessage(ctx, input)
		if err != nil {
			return errMsg{err}
		}
		return startStreamMsg{events: events}
	}
}

func (m Model) readNextEvent() tea.Cmd {
	return func() tea.Msg {
		if m.eventChan == nil {
			return nil
		}

		event, ok := <-m.eventChan
		if !ok {
			return nil
		}
		return AgentEventMsg{Event: event}
	}
}

func (m *Model) handleAgentEvent(event agent.Event) (tea.Model, tea.Cmd) {
	switch event.Type {
	case agent.EventTypeToolPreview:
		toolName := ""
		if event.ToolUse != nil {
			toolName = event.ToolUse.Name
		}
		m.messages = append(m.messages, Message{
			Role:    "tool_preview",
			Content: fmt.Sprintf("Tool: %s\n%s", toolName, event.Content),
		})
		m.pendingPreview = true
		m.updateViewport()
		// Don't read next event - wait for user confirmation
		return m, nil

	case agent.EventTypeTextDelta:
		if len(m.messages) == 0 || m.messages[len(m.messages)-1].Role != "assistant" {
			m.messages = append(m.messages, Message{
				Role:    "assistant",
				Content: event.Content,
			})
		} else {
			m.messages[len(m.messages)-1].Content += event.Content
		}
		m.updateViewport()
		return m, m.waitForNextEvent()

	case agent.EventTypeToolCall:
		toolMsg := fmt.Sprintf("Calling tool: %s", event.ToolUse.Name)
		m.messages = append(m.messages, Message{
			Role:    "tool_call",
			Content: toolMsg,
		})
		m.updateViewport()
		return m, m.waitForNextEvent()

	case agent.EventTypeToolResult:
		result := event.ToolResult.Content
		if len(result) > 200 {
			result = result[:200] + "..."
		}
		m.messages = append(m.messages, Message{
			Role:    "tool_result",
			Content: result,
		})
		m.updateViewport()
		return m, m.waitForNextEvent()

	case agent.EventTypeTokenUpdate:
		if event.TokenInfo != nil {
			m.status.Tokens = event.TokenInfo.CurrentTokens
			m.status.MaxTokens = event.TokenInfo.MaxTokens
			m.status.UsagePercent = event.TokenInfo.UsagePercent
			m.status.Cost = event.TokenInfo.Cost
			m.status.AtWarning = event.TokenInfo.AtWarning

			if event.TokenInfo.AtWarning && m.status.ContextStatus == "" {
				m.status.ContextStatus = "Warning"
			}
		}
		return m, m.waitForNextEvent()

	case agent.EventTypeContextUpdate:
		m.status.ContextStatus = "Compacted"
		m.messages = append(m.messages, Message{
			Role:    "system",
			Content: event.Content,
		})
		m.updateViewport()
		return m, m.waitForNextEvent()

	case agent.EventTypeError:
		m.messages = append(m.messages, Message{
			Role:    "error",
			Content: event.Error.Error(),
		})
		m.updateViewport()
		m.processingMsg = false
		m.eventChan = nil
		return m, nil

	case agent.EventTypeMessageDone:
		m.processingMsg = false
		m.eventChan = nil
		if m.status.UsagePercent < 80 {
			m.status.ContextStatus = ""
		}
		return m, nil
	}

	return m, m.waitForNextEvent()
}

func (m Model) waitForNextEvent() tea.Cmd {
	return func() tea.Msg {
		return continueStreamMsg{}
	}
}

func Run(ag *agent.Agent, model string, confirmChan chan agent.Decision) error {
	p := tea.NewProgram(
		New(ag, model, confirmChan),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}
