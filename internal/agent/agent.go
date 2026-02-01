package agent

import (
	"fmt"
	"os"

	"github.com/taaha3244/potus/internal/config"
	"github.com/taaha3244/potus/internal/context"
	"github.com/taaha3244/potus/internal/permissions"
	"github.com/taaha3244/potus/internal/providers"
	"github.com/taaha3244/potus/internal/tools"
	gocontext "context"
)

const MaxToolIterations = 10

type Agent struct {
	provider       providers.Provider
	toolRegistry   *tools.Registry
	memory         *Memory
	executor       *Executor
	contextManager *context.Manager
	systemPrompt   string
	maxTokens      int
	temperature    float64
	model          string
	confirmChan    chan Decision
	settings       *permissions.Settings
	workDir        string
}

type Config struct {
	Provider      providers.Provider
	ToolRegistry  *tools.Registry
	SystemPrompt  string
	MaxTokens     int
	Temperature   float64
	Model         string
	ContextConfig *config.ContextConfig
	ModelInfo     *providers.Model
	WorkDir       string
	ConfirmChan   chan Decision
	Settings      *permissions.Settings
}

func New(cfg *Config) *Agent {
	var ctxManager *context.Manager

	if cfg.ContextConfig != nil {
		ctxManagerCfg := context.ManagerConfig{
			Provider:            cfg.Provider,
			MaxTokens:           cfg.ContextConfig.MaxTokens,
			ReserveForResponse:  cfg.ContextConfig.ReserveForResponse,
			WarnThreshold:       cfg.ContextConfig.WarnThreshold,
			CompactThreshold:    cfg.ContextConfig.CompactThreshold,
			AutoCompact:         cfg.ContextConfig.AutoCompact,
			AutoPrune:           cfg.ContextConfig.AutoPrune,
			ProtectedTools:      cfg.ContextConfig.ProtectedTools,
			LoadProjectContext:  cfg.ContextConfig.LoadProjectContext,
			ProjectContextFiles: cfg.ContextConfig.ProjectContextFiles,
			MaxProjectTokens:    cfg.ContextConfig.MaxProjectContextTokens,
		}

		if cfg.ModelInfo != nil {
			ctxManagerCfg.ModelContextSize = cfg.ModelInfo.ContextSize
		}

		ctxManager = context.NewManager(ctxManagerCfg)

		if cfg.ModelInfo != nil {
			ctxManager.SetPricing(
				cfg.ModelInfo.Pricing.InputPer1M,
				cfg.ModelInfo.Pricing.OutputPer1M,
			)
		}

		if cfg.ContextConfig.LoadProjectContext {
			workDir := cfg.WorkDir
			if workDir == "" {
				workDir, _ = os.Getwd()
			}
			_ = ctxManager.LoadProjectContext(workDir)
		}
	}

	systemPrompt := cfg.SystemPrompt
	if ctxManager != nil {
		projectContext := ctxManager.GetProjectContextForPrompt()
		if projectContext != "" {
			systemPrompt = systemPrompt + projectContext
		}
	}

	var estimator context.TokenEstimator
	if ctxManager != nil {
		estimator = ctxManager.GetEstimator()
	}
	memory := NewMemory(estimator)

	if estimator != nil {
		memory.SetSystemTokens(context.EstimateSystemPrompt(systemPrompt))
	}

	workDir := cfg.WorkDir
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	executor := NewExecutorWithConfig(&ExecutorConfig{
		Registry: cfg.ToolRegistry,
		Settings: cfg.Settings,
		WorkDir:  workDir,
	})

	return &Agent{
		provider:       cfg.Provider,
		toolRegistry:   cfg.ToolRegistry,
		memory:         memory,
		executor:       executor,
		contextManager: ctxManager,
		systemPrompt:   systemPrompt,
		maxTokens:      cfg.MaxTokens,
		temperature:    cfg.Temperature,
		model:          cfg.Model,
		confirmChan:    cfg.ConfirmChan,
		settings:       cfg.Settings,
		workDir:        workDir,
	}
}

func (a *Agent) ProcessMessage(ctx gocontext.Context, userMessage string) (<-chan Event, error) {
	eventChan := make(chan Event, 100)
	go a.processLoop(ctx, userMessage, eventChan)
	return eventChan, nil
}

func (a *Agent) processLoop(ctx gocontext.Context, userMessage string, eventChan chan<- Event) {
	defer close(eventChan)

	// Set up the confirmation function that bridges to TUI
	if a.confirmChan != nil {
		a.executor.confirmFn = func(toolName, action, preview string) (Decision, error) {
			// Emit preview event to TUI
			eventChan <- Event{
				Type:    EventTypeToolPreview,
				Content: preview,
				ToolUse: &providers.ToolUseContent{Name: toolName},
			}
			// Block until TUI responds
			select {
			case decision := <-a.confirmChan:
				return decision, nil
			case <-ctx.Done():
				return DecisionDeny, ctx.Err()
			}
		}
	}

	a.memory.AddUserMessage(userMessage)
	a.emitTokenUpdate(eventChan)

	for i := 0; i < MaxToolIterations; i++ {
		messages := a.memory.GetMessages()
		tokenInfo := a.memory.GetTokenInfo()

		if a.contextManager != nil {
			preparedMsgs, err := a.contextManager.PrepareContext(ctx, messages, tokenInfo)
			if err != nil {
				eventChan <- Event{
					Type:  EventTypeError,
					Error: fmt.Errorf("context management failed: %w", err),
				}
			} else if len(preparedMsgs) != len(messages) {
				a.memory.ReplaceMessages(preparedMsgs)
				messages = preparedMsgs

				eventChan <- Event{
					Type:    EventTypeContextUpdate,
					Content: "Conversation history was optimized to manage context size.",
				}
				a.emitTokenUpdate(eventChan)
			}
		}

		req := &providers.ChatRequest{
			Messages:    messages,
			Tools:       a.toolRegistry.ToProviderTools(),
			MaxTokens:   a.maxTokens,
			Temperature: a.temperature,
			Model:       a.model,
			System:      a.systemPrompt,
		}

		chatEvents, err := a.provider.Chat(ctx, req)
		if err != nil {
			eventChan <- Event{
				Type:  EventTypeError,
				Error: fmt.Errorf("failed to call provider: %w", err),
			}
			return
		}

		assistantMessage := &providers.Message{
			Role:    providers.RoleAssistant,
			Content: []providers.ContentBlock{},
		}

		var textBuffer string
		toolCalls := []*providers.ToolUseContent{}

		for chatEvent := range chatEvents {
			switch chatEvent.Type {
			case providers.EventTypeTextDelta:
				textBuffer += chatEvent.Content
				eventChan <- Event{
					Type:    EventTypeTextDelta,
					Content: chatEvent.Content,
				}

			case providers.EventTypeToolUse:
				if textBuffer != "" {
					assistantMessage.Content = append(assistantMessage.Content, &providers.TextContent{
						Text: textBuffer,
					})
					textBuffer = ""
				}

				assistantMessage.Content = append(assistantMessage.Content, chatEvent.ToolUse)
				toolCalls = append(toolCalls, chatEvent.ToolUse)

				eventChan <- Event{
					Type:    EventTypeToolCall,
					ToolUse: chatEvent.ToolUse,
				}

			case providers.EventTypeMessageDone:
				if textBuffer != "" {
					assistantMessage.Content = append(assistantMessage.Content, &providers.TextContent{
						Text: textBuffer,
					})
					textBuffer = ""
				}

				if chatEvent.Usage != nil && a.contextManager != nil {
					a.contextManager.RecordUsage(chatEvent.Usage.InputTokens, chatEvent.Usage.OutputTokens)
				}

				eventChan <- Event{
					Type:  EventTypeMessageDone,
					Usage: chatEvent.Usage,
				}

			case providers.EventTypeError:
				eventChan <- Event{
					Type:  EventTypeError,
					Error: chatEvent.Error,
				}
				return
			}
		}

		a.memory.AddMessage(assistantMessage)
		a.emitTokenUpdate(eventChan)

		if len(toolCalls) == 0 {
			break
		}

		toolResults := []*providers.ToolResultContent{}
		for _, toolCall := range toolCalls {
			result, err := a.executor.Execute(ctx, toolCall)

			toolResult := &providers.ToolResultContent{
				ToolUseID: toolCall.ID,
			}

			if err != nil {
				toolResult.IsError = true
				toolResult.Content = err.Error()
			} else {
				toolResult.Content = result.Output
				toolResult.IsError = !result.Success
			}

			toolResults = append(toolResults, toolResult)

			eventChan <- Event{
				Type:       EventTypeToolResult,
				ToolResult: toolResult,
			}
		}

		toolMessage := &providers.Message{
			Role:    providers.RoleTool,
			Content: make([]providers.ContentBlock, len(toolResults)),
		}
		for i, tr := range toolResults {
			toolMessage.Content[i] = tr
		}
		a.memory.AddMessage(toolMessage)

		a.emitTokenUpdate(eventChan)
	}
}

func (a *Agent) emitTokenUpdate(eventChan chan<- Event) {
	if a.contextManager == nil {
		return
	}

	currentTokens := a.memory.GetTotalTokens()
	snapshot := a.contextManager.GetBudgetSnapshot(currentTokens)

	eventChan <- Event{
		Type: EventTypeTokenUpdate,
		TokenInfo: &TokenUpdateInfo{
			CurrentTokens: snapshot.CurrentContextTokens,
			MaxTokens:     snapshot.MaxContextTokens,
			UsagePercent:  snapshot.UsagePercent,
			SessionTokens: snapshot.SessionInputTokens + snapshot.SessionOutputTokens,
			Cost:          snapshot.SessionCost,
			AtWarning:     snapshot.AtWarningLevel,
		},
	}
}

func (a *Agent) GetMemory() *Memory {
	return a.memory
}

func (a *Agent) GetContextManager() *context.Manager {
	return a.contextManager
}

func (a *Agent) GetTokenSummary() TokenSummary {
	return a.memory.GetTokenSummary()
}

type Event struct {
	Type       EventType
	Content    string
	ToolUse    *providers.ToolUseContent
	ToolResult *providers.ToolResultContent
	TokenInfo  *TokenUpdateInfo
	Usage      *providers.Usage
	Error      error
}

type EventType string

const (
	EventTypeTextDelta     EventType = "text_delta"
	EventTypeToolCall      EventType = "tool_call"
	EventTypeToolResult    EventType = "tool_result"
	EventTypeMessageDone   EventType = "message_done"
	EventTypeError         EventType = "error"
	EventTypeTokenUpdate   EventType = "token_update"
	EventTypeContextUpdate EventType = "context_update"
	EventTypeToolPreview   EventType = "tool_preview"
)

type TokenUpdateInfo struct {
	CurrentTokens int
	MaxTokens     int
	UsagePercent  float64
	SessionTokens int
	Cost          float64
	AtWarning     bool
}
