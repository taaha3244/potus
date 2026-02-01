package context

import (
	"context"
	"fmt"
	"sync"

	"github.com/taaha3244/potus/internal/providers"
)

type ContextAction int

const (
	ActionNone ContextAction = iota
	ActionWarn
	ActionPrune
	ActionCompact
)

func (a ContextAction) String() string {
	switch a {
	case ActionNone:
		return "none"
	case ActionWarn:
		return "warn"
	case ActionPrune:
		return "prune"
	case ActionCompact:
		return "compact"
	default:
		return "unknown"
	}
}

type Manager struct {
	mu             sync.RWMutex
	estimator      TokenEstimator
	compactor      *Compactor
	pruner         *Pruner
	projectFiles   *ProjectFiles
	budget         *Budget
	autoCompact    bool
	autoPrune      bool
	eventChan      chan<- ContextEvent
	projectContext *ProjectContext
}

type ManagerConfig struct {
	Provider            providers.Provider
	MaxTokens           int
	ReserveForResponse  int
	ModelContextSize    int
	WarnThreshold       float64
	CompactThreshold    float64
	AutoCompact         bool
	AutoPrune           bool
	ProtectedTools      []string
	LoadProjectContext  bool
	ProjectContextFiles []string
	MaxProjectTokens    int
	EventChan           chan<- ContextEvent
}

func NewManager(cfg ManagerConfig) *Manager {
	estimator := NewSimpleEstimator()

	budget := NewBudget(BudgetConfig{
		MaxTokens:          cfg.MaxTokens,
		ReserveForResponse: cfg.ReserveForResponse,
		ModelContextSize:   cfg.ModelContextSize,
		WarnThreshold:      cfg.WarnThreshold,
		CompactThreshold:   cfg.CompactThreshold,
	})

	pruner := NewPruner(PrunerConfig{
		ProtectedTools:  cfg.ProtectedTools,
		ProtectionRatio: 0.30,
	})

	var compactor *Compactor
	if cfg.Provider != nil {
		compactor = NewCompactor(CompactorConfig{
			Provider:          cfg.Provider,
			Estimator:         estimator,
			ProtectedMessages: 6,
			MaxSummaryTokens:  1000,
		})
	}

	projectFiles := NewProjectFiles(ProjectFilesConfig{
		ContextFileNames: cfg.ProjectContextFiles,
		MaxTokens:        cfg.MaxProjectTokens,
	})

	return &Manager{
		estimator:    estimator,
		compactor:    compactor,
		pruner:       pruner,
		projectFiles: projectFiles,
		budget:       budget,
		autoCompact:  cfg.AutoCompact,
		autoPrune:    cfg.AutoPrune,
		eventChan:    cfg.EventChan,
	}
}

func (m *Manager) LoadProjectContext(workDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx, err := m.projectFiles.Load(workDir, m.estimator)
	if err != nil {
		return fmt.Errorf("failed to load project context: %w", err)
	}

	m.projectContext = ctx
	return nil
}

func (m *Manager) GetProjectContextForPrompt() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.projectContext == nil {
		return ""
	}

	return m.projectFiles.FormatForSystemPrompt(m.projectContext)
}

func (m *Manager) GetProjectContextTokens() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.projectContext == nil {
		return 0
	}

	return m.projectContext.TotalTokens
}

func (m *Manager) CheckContext(currentTokens int) ContextAction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := m.budget.GetSnapshot(currentTokens)

	if snapshot.AtCompactLevel {
		return ActionCompact
	}

	if snapshot.AtWarningLevel {
		return ActionWarn
	}

	return ActionNone
}

func (m *Manager) PrepareContext(
	ctx context.Context,
	messages []providers.Message,
	tokenInfo []TokenInfo,
) ([]providers.Message, error) {
	currentTokens := m.calculateTokens(tokenInfo)
	action := m.CheckContext(currentTokens)

	switch action {
	case ActionWarn:
		m.emitEvent(NewWarningEvent(
			currentTokens,
			m.budget.GetEffectiveLimit(),
			"Approaching context limit",
		))
		return messages, nil

	case ActionCompact:
		if m.autoCompact && m.compactor != nil {
			compacted, result, err := m.compactor.Compact(ctx, messages)
			if err != nil {
				m.emitEvent(NewErrorEvent(err))
				return messages, err
			}

			m.emitEvent(NewCompactedEvent(
				result.CompactedTokens,
				m.budget.GetEffectiveLimit(),
				fmt.Sprintf("Compacted %d messages, saved ~%d tokens",
					result.SummarizedMessages,
					result.OriginalTokens-result.CompactedTokens),
			))

			return compacted, nil
		}

		if m.autoPrune && m.pruner.ShouldPrune(tokenInfo) {
			pruned, result := m.pruner.Prune(messages, tokenInfo)
			m.emitEvent(NewPrunedEvent(
				result.TokensSaved,
				fmt.Sprintf("Pruned %d tool results, saved ~%d tokens",
					result.MessagesPruned, result.TokensSaved),
			))
			return pruned, nil
		}

		m.emitEvent(NewWarningEvent(
			currentTokens,
			m.budget.GetEffectiveLimit(),
			"Context limit reached, consider starting a new conversation",
		))
		return messages, nil
	}

	return messages, nil
}

func (m *Manager) EstimateTokens(messages []providers.Message) int {
	return m.estimator.EstimateMessages(messages)
}

func (m *Manager) EstimateMessage(msg *providers.Message) int {
	return m.estimator.EstimateMessage(msg)
}

func (m *Manager) RecordUsage(inputTokens, outputTokens int) {
	m.budget.RecordUsage(inputTokens, outputTokens)
}

func (m *Manager) SetPricing(inputPer1M, outputPer1M float64) {
	m.budget.SetPricing(inputPer1M, outputPer1M)
}

func (m *Manager) GetBudgetSnapshot(currentTokens int) BudgetSnapshot {
	return m.budget.GetSnapshot(currentTokens)
}

func (m *Manager) GetEffectiveLimit() int {
	return m.budget.GetEffectiveLimit()
}

func (m *Manager) GetEstimator() TokenEstimator {
	return m.estimator
}

func (m *Manager) UpdateModelContextSize(size int) {
	m.budget.UpdateModelContextSize(size)
}

func (m *Manager) calculateTokens(info []TokenInfo) int {
	total := 0
	for _, i := range info {
		total += i.Tokens
	}
	return total
}

func (m *Manager) emitEvent(event ContextEvent) {
	if m.eventChan != nil {
		select {
		case m.eventChan <- event:
		default:
		}
	}
}

func (m *Manager) Prune(messages []providers.Message, tokenInfo []TokenInfo) ([]providers.Message, PruneResult) {
	return m.pruner.Prune(messages, tokenInfo)
}

func (m *Manager) Compact(ctx context.Context, messages []providers.Message) ([]providers.Message, CompactResult, error) {
	if m.compactor == nil {
		return messages, CompactResult{}, fmt.Errorf("compactor not available (no provider configured)")
	}
	return m.compactor.Compact(ctx, messages)
}

func (m *Manager) GetLoadedProjectFiles() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.projectFiles.GetLoadedFiles(m.projectContext)
}
