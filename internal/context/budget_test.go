package context

import (
	"testing"
)

func TestNewBudget(t *testing.T) {
	cfg := BudgetConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
		ModelContextSize:   128000,
		WarnThreshold:      0.80,
		CompactThreshold:   0.90,
	}

	budget := NewBudget(cfg)

	if budget == nil {
		t.Fatal("NewBudget returned nil")
	}

	effectiveLimit := budget.GetEffectiveLimit()
	// Should be min(MaxTokens, ModelContextSize) - ReserveForResponse
	expected := 100000 - 8192
	if effectiveLimit != expected {
		t.Errorf("GetEffectiveLimit() = %d, want %d", effectiveLimit, expected)
	}
}

func TestBudget_DefaultThresholds(t *testing.T) {
	// Test that defaults are applied when thresholds are 0
	// Use no reserve for simpler percentage math
	cfg := BudgetConfig{
		MaxTokens:          100000,
		ReserveForResponse: 0,
		WarnThreshold:      0, // Should default to 0.80
		CompactThreshold:   0, // Should default to 0.90
	}

	budget := NewBudget(cfg)

	// Exactly at 80% should warn
	if !budget.ShouldWarn(80000) {
		t.Error("Should warn at 80%")
	}

	if !budget.ShouldWarn(80001) {
		t.Error("Should warn above 80%")
	}

	if budget.ShouldWarn(79999) {
		t.Error("Should not warn below 80%")
	}
}

func TestBudget_GetEffectiveLimit(t *testing.T) {
	tests := []struct {
		name             string
		maxTokens        int
		modelContextSize int
		reserve          int
		expected         int
	}{
		{
			name:             "max tokens is smaller",
			maxTokens:        100000,
			modelContextSize: 200000,
			reserve:          8192,
			expected:         100000 - 8192,
		},
		{
			name:             "model context is smaller",
			maxTokens:        200000,
			modelContextSize: 100000,
			reserve:          8192,
			expected:         100000 - 8192,
		},
		{
			name:             "model context size is 0",
			maxTokens:        100000,
			modelContextSize: 0,
			reserve:          8192,
			expected:         100000 - 8192,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget := NewBudget(BudgetConfig{
				MaxTokens:          tt.maxTokens,
				ModelContextSize:   tt.modelContextSize,
				ReserveForResponse: tt.reserve,
			})

			got := budget.GetEffectiveLimit()
			if got != tt.expected {
				t.Errorf("GetEffectiveLimit() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestBudget_RecordUsage(t *testing.T) {
	budget := NewBudget(BudgetConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
	})

	// Set pricing
	budget.SetPricing(3.0, 15.0) // $3/1M input, $15/1M output

	// Record some usage
	budget.RecordUsage(1000, 500)

	input, output := budget.GetSessionTokens()
	if input != 1000 {
		t.Errorf("SessionInputTokens = %d, want 1000", input)
	}
	if output != 500 {
		t.Errorf("SessionOutputTokens = %d, want 500", output)
	}

	// Check cost calculation
	// Input: 1000 / 1M * $3 = $0.003
	// Output: 500 / 1M * $15 = $0.0075
	// Total: $0.0105
	expectedCost := 0.0105
	actualCost := budget.GetSessionCost()
	if actualCost < expectedCost-0.0001 || actualCost > expectedCost+0.0001 {
		t.Errorf("SessionCost = %f, want %f", actualCost, expectedCost)
	}

	// Record more usage
	budget.RecordUsage(2000, 1000)

	input, output = budget.GetSessionTokens()
	if input != 3000 {
		t.Errorf("SessionInputTokens after second call = %d, want 3000", input)
	}
	if output != 1500 {
		t.Errorf("SessionOutputTokens after second call = %d, want 1500", output)
	}
}

func TestBudget_GetSnapshot(t *testing.T) {
	budget := NewBudget(BudgetConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
		WarnThreshold:      0.80,
		CompactThreshold:   0.90,
	})

	budget.SetPricing(3.0, 15.0)
	budget.RecordUsage(1000, 500)

	currentTokens := 50000
	snapshot := budget.GetSnapshot(currentTokens)

	if snapshot.CurrentContextTokens != currentTokens {
		t.Errorf("CurrentContextTokens = %d, want %d", snapshot.CurrentContextTokens, currentTokens)
	}

	expectedMax := 100000 - 8192
	if snapshot.MaxContextTokens != expectedMax {
		t.Errorf("MaxContextTokens = %d, want %d", snapshot.MaxContextTokens, expectedMax)
	}

	expectedPercent := float64(currentTokens) / float64(expectedMax) * 100
	if snapshot.UsagePercent < expectedPercent-0.1 || snapshot.UsagePercent > expectedPercent+0.1 {
		t.Errorf("UsagePercent = %f, want %f", snapshot.UsagePercent, expectedPercent)
	}

	if snapshot.RemainingTokens != expectedMax-currentTokens {
		t.Errorf("RemainingTokens = %d, want %d", snapshot.RemainingTokens, expectedMax-currentTokens)
	}

	// At 50%, should not be at warning or compact level
	if snapshot.AtWarningLevel {
		t.Error("Should not be at warning level at 50%")
	}
	if snapshot.AtCompactLevel {
		t.Error("Should not be at compact level at 50%")
	}
}

func TestBudget_WarningAndCompactLevels(t *testing.T) {
	budget := NewBudget(BudgetConfig{
		MaxTokens:          100000,
		ReserveForResponse: 0, // No reserve for simpler math
		WarnThreshold:      0.80,
		CompactThreshold:   0.90,
	})

	tests := []struct {
		name          string
		currentTokens int
		atWarning     bool
		atCompact     bool
	}{
		{
			name:          "at 50%",
			currentTokens: 50000,
			atWarning:     false,
			atCompact:     false,
		},
		{
			name:          "at 79%",
			currentTokens: 79000,
			atWarning:     false,
			atCompact:     false,
		},
		{
			name:          "at 80%",
			currentTokens: 80000,
			atWarning:     true,
			atCompact:     false,
		},
		{
			name:          "at 85%",
			currentTokens: 85000,
			atWarning:     true,
			atCompact:     false,
		},
		{
			name:          "at 89%",
			currentTokens: 89000,
			atWarning:     true,
			atCompact:     false,
		},
		{
			name:          "at 90%",
			currentTokens: 90000,
			atWarning:     true,
			atCompact:     true,
		},
		{
			name:          "at 95%",
			currentTokens: 95000,
			atWarning:     true,
			atCompact:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := budget.GetSnapshot(tt.currentTokens)

			if snapshot.AtWarningLevel != tt.atWarning {
				t.Errorf("AtWarningLevel = %v, want %v", snapshot.AtWarningLevel, tt.atWarning)
			}
			if snapshot.AtCompactLevel != tt.atCompact {
				t.Errorf("AtCompactLevel = %v, want %v", snapshot.AtCompactLevel, tt.atCompact)
			}
		})
	}
}

func TestBudget_ShouldWarn(t *testing.T) {
	budget := NewBudget(BudgetConfig{
		MaxTokens:          100000,
		ReserveForResponse: 0,
		WarnThreshold:      0.80,
	})

	if budget.ShouldWarn(79999) {
		t.Error("Should not warn below 80%")
	}

	if !budget.ShouldWarn(80000) {
		t.Error("Should warn at 80%")
	}

	if !budget.ShouldWarn(90000) {
		t.Error("Should warn above 80%")
	}
}

func TestBudget_ShouldCompact(t *testing.T) {
	budget := NewBudget(BudgetConfig{
		MaxTokens:          100000,
		ReserveForResponse: 0,
		CompactThreshold:   0.90,
	})

	if budget.ShouldCompact(89999) {
		t.Error("Should not compact below 90%")
	}

	if !budget.ShouldCompact(90000) {
		t.Error("Should compact at 90%")
	}

	if !budget.ShouldCompact(95000) {
		t.Error("Should compact above 90%")
	}
}

func TestBudget_Reset(t *testing.T) {
	budget := NewBudget(BudgetConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
	})

	budget.SetPricing(3.0, 15.0)
	budget.RecordUsage(1000, 500)

	// Verify there's usage
	input, output := budget.GetSessionTokens()
	if input == 0 || output == 0 {
		t.Error("Should have usage before reset")
	}

	// Reset
	budget.Reset()

	input, output = budget.GetSessionTokens()
	if input != 0 || output != 0 {
		t.Error("Tokens should be 0 after reset")
	}

	if budget.GetSessionCost() != 0 {
		t.Error("Cost should be 0 after reset")
	}
}

func TestBudget_UpdateModelContextSize(t *testing.T) {
	budget := NewBudget(BudgetConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
		ModelContextSize:   128000,
	})

	// Initial effective limit
	initialLimit := budget.GetEffectiveLimit()
	expectedInitial := 100000 - 8192
	if initialLimit != expectedInitial {
		t.Errorf("Initial limit = %d, want %d", initialLimit, expectedInitial)
	}

	// Update to smaller model context
	budget.UpdateModelContextSize(50000)

	newLimit := budget.GetEffectiveLimit()
	expectedNew := 50000 - 8192
	if newLimit != expectedNew {
		t.Errorf("New limit = %d, want %d", newLimit, expectedNew)
	}
}

func TestBudget_ConcurrentAccess(t *testing.T) {
	budget := NewBudget(BudgetConfig{
		MaxTokens:          100000,
		ReserveForResponse: 8192,
	})

	budget.SetPricing(3.0, 15.0)

	done := make(chan bool, 10)

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func() {
			budget.RecordUsage(100, 50)
			done <- true
		}()
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_ = budget.GetSnapshot(50000)
			_ = budget.GetSessionCost()
			_ = budget.GetEffectiveLimit()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Verify final state
	input, output := budget.GetSessionTokens()
	if input != 1000 {
		t.Errorf("Final input tokens = %d, want 1000", input)
	}
	if output != 500 {
		t.Errorf("Final output tokens = %d, want 500", output)
	}
}
