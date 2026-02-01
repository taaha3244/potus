package context

import (
	"errors"
	"testing"
)

func TestContextEventType(t *testing.T) {
	tests := []struct {
		eventType ContextEventType
		expected  string
	}{
		{EventTypeUsageUpdate, "usage_update"},
		{EventTypeWarning, "warning"},
		{EventTypePruned, "pruned"},
		{EventTypeCompacted, "compacted"},
		{EventTypeError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.eventType) != tt.expected {
				t.Errorf("ContextEventType = %s, want %s", string(tt.eventType), tt.expected)
			}
		})
	}
}

func TestNewUsageEvent(t *testing.T) {
	tests := []struct {
		name            string
		current         int
		max             int
		cost            float64
		expectedPercent float64
	}{
		{
			name:            "50% usage",
			current:         50000,
			max:             100000,
			cost:            0.05,
			expectedPercent: 50.0,
		},
		{
			name:            "0% usage",
			current:         0,
			max:             100000,
			cost:            0.0,
			expectedPercent: 0.0,
		},
		{
			name:            "100% usage",
			current:         100000,
			max:             100000,
			cost:            1.23,
			expectedPercent: 100.0,
		},
		{
			name:            "max is zero (edge case)",
			current:         1000,
			max:             0,
			cost:            0.01,
			expectedPercent: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewUsageEvent(tt.current, tt.max, tt.cost)

			if event.Type != EventTypeUsageUpdate {
				t.Errorf("Type = %v, want %v", event.Type, EventTypeUsageUpdate)
			}
			if event.CurrentTokens != tt.current {
				t.Errorf("CurrentTokens = %d, want %d", event.CurrentTokens, tt.current)
			}
			if event.MaxTokens != tt.max {
				t.Errorf("MaxTokens = %d, want %d", event.MaxTokens, tt.max)
			}
			if event.UsagePercent != tt.expectedPercent {
				t.Errorf("UsagePercent = %f, want %f", event.UsagePercent, tt.expectedPercent)
			}
			if event.Cost != tt.cost {
				t.Errorf("Cost = %f, want %f", event.Cost, tt.cost)
			}
		})
	}
}

func TestNewWarningEvent(t *testing.T) {
	tests := []struct {
		name            string
		current         int
		max             int
		message         string
		expectedPercent float64
	}{
		{
			name:            "80% warning",
			current:         80000,
			max:             100000,
			message:         "Approaching context limit",
			expectedPercent: 80.0,
		},
		{
			name:            "95% warning",
			current:         95000,
			max:             100000,
			message:         "Critical context limit",
			expectedPercent: 95.0,
		},
		{
			name:            "max is zero",
			current:         1000,
			max:             0,
			message:         "Warning",
			expectedPercent: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewWarningEvent(tt.current, tt.max, tt.message)

			if event.Type != EventTypeWarning {
				t.Errorf("Type = %v, want %v", event.Type, EventTypeWarning)
			}
			if event.CurrentTokens != tt.current {
				t.Errorf("CurrentTokens = %d, want %d", event.CurrentTokens, tt.current)
			}
			if event.MaxTokens != tt.max {
				t.Errorf("MaxTokens = %d, want %d", event.MaxTokens, tt.max)
			}
			if event.UsagePercent != tt.expectedPercent {
				t.Errorf("UsagePercent = %f, want %f", event.UsagePercent, tt.expectedPercent)
			}
			if event.Message != tt.message {
				t.Errorf("Message = %q, want %q", event.Message, tt.message)
			}
		})
	}
}

func TestNewCompactedEvent(t *testing.T) {
	tests := []struct {
		name            string
		current         int
		max             int
		message         string
		expectedPercent float64
	}{
		{
			name:            "compacted to 50%",
			current:         50000,
			max:             100000,
			message:         "Compacted 10 messages",
			expectedPercent: 50.0,
		},
		{
			name:            "compacted with details",
			current:         30000,
			max:             100000,
			message:         "Saved 40000 tokens",
			expectedPercent: 30.0,
		},
		{
			name:            "max is zero",
			current:         1000,
			max:             0,
			message:         "Compacted",
			expectedPercent: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewCompactedEvent(tt.current, tt.max, tt.message)

			if event.Type != EventTypeCompacted {
				t.Errorf("Type = %v, want %v", event.Type, EventTypeCompacted)
			}
			if event.CurrentTokens != tt.current {
				t.Errorf("CurrentTokens = %d, want %d", event.CurrentTokens, tt.current)
			}
			if event.MaxTokens != tt.max {
				t.Errorf("MaxTokens = %d, want %d", event.MaxTokens, tt.max)
			}
			if event.UsagePercent != tt.expectedPercent {
				t.Errorf("UsagePercent = %f, want %f", event.UsagePercent, tt.expectedPercent)
			}
			if event.Message != tt.message {
				t.Errorf("Message = %q, want %q", event.Message, tt.message)
			}
		})
	}
}

func TestNewPrunedEvent(t *testing.T) {
	tests := []struct {
		name        string
		tokensSaved int
		message     string
	}{
		{
			name:        "pruned with savings",
			tokensSaved: 5000,
			message:     "Pruned 3 tool results, saved 5000 tokens",
		},
		{
			name:        "pruned with details",
			tokensSaved: 10000,
			message:     "Old tool results removed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewPrunedEvent(tt.tokensSaved, tt.message)

			if event.Type != EventTypePruned {
				t.Errorf("Type = %v, want %v", event.Type, EventTypePruned)
			}
			if event.Message != tt.message {
				t.Errorf("Message = %q, want %q", event.Message, tt.message)
			}
		})
	}
}

func TestNewErrorEvent(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "simple error",
			err:  errors.New("context management failed"),
		},
		{
			name: "wrapped error",
			err:  errors.New("compaction failed: provider error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := NewErrorEvent(tt.err)

			if event.Type != EventTypeError {
				t.Errorf("Type = %v, want %v", event.Type, EventTypeError)
			}
			if event.Error != tt.err {
				t.Errorf("Error = %v, want %v", event.Error, tt.err)
			}
			if event.Message != tt.err.Error() {
				t.Errorf("Message = %q, want %q", event.Message, tt.err.Error())
			}
		})
	}
}

func TestContextEventFields(t *testing.T) {
	// Test that all fields can be set directly
	event := ContextEvent{
		Type:          EventTypeUsageUpdate,
		CurrentTokens: 50000,
		MaxTokens:     100000,
		UsagePercent:  50.0,
		Message:       "Test message",
		Cost:          0.05,
		Error:         nil,
	}

	if event.Type != EventTypeUsageUpdate {
		t.Error("Type field not set correctly")
	}
	if event.CurrentTokens != 50000 {
		t.Error("CurrentTokens field not set correctly")
	}
	if event.MaxTokens != 100000 {
		t.Error("MaxTokens field not set correctly")
	}
	if event.UsagePercent != 50.0 {
		t.Error("UsagePercent field not set correctly")
	}
	if event.Message != "Test message" {
		t.Error("Message field not set correctly")
	}
	if event.Cost != 0.05 {
		t.Error("Cost field not set correctly")
	}
	if event.Error != nil {
		t.Error("Error field should be nil")
	}
}

func TestUsagePercentCalculation(t *testing.T) {
	// Test precise percentage calculations
	tests := []struct {
		current  int
		max      int
		expected float64
	}{
		{25000, 100000, 25.0},
		{33333, 100000, 33.333},
		{66666, 100000, 66.666},
		{1, 3, 33.33333333333333},
	}

	for _, tt := range tests {
		event := NewUsageEvent(tt.current, tt.max, 0)
		// Allow small floating point differences
		diff := event.UsagePercent - tt.expected
		if diff < -0.001 || diff > 0.001 {
			t.Errorf("UsagePercent for %d/%d = %f, want ~%f", tt.current, tt.max, event.UsagePercent, tt.expected)
		}
	}
}
