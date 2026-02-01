package context

import (
	"sync"
)

type Budget struct {
	mu                sync.RWMutex
	maxTokens         int
	reserveForResponse int
	modelContextSize   int
	totalInputTokens   int
	totalOutputTokens  int
	sessionCost        float64
	inputPricePer1M    float64
	outputPricePer1M   float64
	warnThreshold      float64
	compactThreshold   float64
}

type BudgetConfig struct {
	MaxTokens          int
	ReserveForResponse int
	ModelContextSize   int
	WarnThreshold      float64
	CompactThreshold   float64
}

type BudgetSnapshot struct {
	CurrentContextTokens int
	MaxContextTokens     int
	UsagePercent         float64
	SessionInputTokens   int
	SessionOutputTokens  int
	SessionCost          float64
	RemainingTokens      int
	AtWarningLevel       bool
	AtCompactLevel       bool
}

func NewBudget(cfg BudgetConfig) *Budget {
	warnThreshold := cfg.WarnThreshold
	if warnThreshold <= 0 {
		warnThreshold = 0.80
	}

	compactThreshold := cfg.CompactThreshold
	if compactThreshold <= 0 {
		compactThreshold = 0.90
	}

	return &Budget{
		maxTokens:          cfg.MaxTokens,
		reserveForResponse: cfg.ReserveForResponse,
		modelContextSize:   cfg.ModelContextSize,
		warnThreshold:      warnThreshold,
		compactThreshold:   compactThreshold,
	}
}

func (b *Budget) SetPricing(inputPer1M, outputPer1M float64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.inputPricePer1M = inputPer1M
	b.outputPricePer1M = outputPer1M
}

func (b *Budget) RecordUsage(inputTokens, outputTokens int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.totalInputTokens += inputTokens
	b.totalOutputTokens += outputTokens

	inputCost := float64(inputTokens) / 1_000_000 * b.inputPricePer1M
	outputCost := float64(outputTokens) / 1_000_000 * b.outputPricePer1M
	b.sessionCost += inputCost + outputCost
}

func (b *Budget) GetEffectiveLimit() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	limit := b.maxTokens
	if b.modelContextSize > 0 && b.modelContextSize < limit {
		limit = b.modelContextSize
	}
	return limit - b.reserveForResponse
}

func (b *Budget) GetSnapshot(currentContextTokens int) BudgetSnapshot {
	b.mu.RLock()
	defer b.mu.RUnlock()

	effectiveMax := b.getEffectiveLimitUnlocked()
	usagePercent := 0.0
	if effectiveMax > 0 {
		usagePercent = float64(currentContextTokens) / float64(effectiveMax) * 100
	}

	return BudgetSnapshot{
		CurrentContextTokens: currentContextTokens,
		MaxContextTokens:     effectiveMax,
		UsagePercent:         usagePercent,
		SessionInputTokens:   b.totalInputTokens,
		SessionOutputTokens:  b.totalOutputTokens,
		SessionCost:          b.sessionCost,
		RemainingTokens:      effectiveMax - currentContextTokens,
		AtWarningLevel:       usagePercent >= b.warnThreshold*100,
		AtCompactLevel:       usagePercent >= b.compactThreshold*100,
	}
}

func (b *Budget) ShouldWarn(currentContextTokens int) bool {
	effectiveMax := b.GetEffectiveLimit()
	if effectiveMax <= 0 {
		return false
	}
	usagePercent := float64(currentContextTokens) / float64(effectiveMax)
	return usagePercent >= b.warnThreshold
}

func (b *Budget) ShouldCompact(currentContextTokens int) bool {
	effectiveMax := b.GetEffectiveLimit()
	if effectiveMax <= 0 {
		return false
	}
	usagePercent := float64(currentContextTokens) / float64(effectiveMax)
	return usagePercent >= b.compactThreshold
}

func (b *Budget) GetSessionCost() float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.sessionCost
}

func (b *Budget) GetSessionTokens() (input, output int) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.totalInputTokens, b.totalOutputTokens
}

func (b *Budget) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.totalInputTokens = 0
	b.totalOutputTokens = 0
	b.sessionCost = 0
}

func (b *Budget) UpdateModelContextSize(size int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.modelContextSize = size
}

func (b *Budget) getEffectiveLimitUnlocked() int {
	limit := b.maxTokens
	if b.modelContextSize > 0 && b.modelContextSize < limit {
		limit = b.modelContextSize
	}
	return limit - b.reserveForResponse
}
