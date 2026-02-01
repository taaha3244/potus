package context

type ContextEventType string

const (
	EventTypeUsageUpdate ContextEventType = "usage_update"
	EventTypeWarning     ContextEventType = "warning"
	EventTypePruned      ContextEventType = "pruned"
	EventTypeCompacted   ContextEventType = "compacted"
	EventTypeError       ContextEventType = "error"
)

type ContextEvent struct {
	Type          ContextEventType
	CurrentTokens int
	MaxTokens     int
	UsagePercent  float64
	Message       string
	Cost          float64
	Error         error
}

func NewUsageEvent(current, max int, cost float64) ContextEvent {
	percent := 0.0
	if max > 0 {
		percent = float64(current) / float64(max) * 100
	}
	return ContextEvent{
		Type:          EventTypeUsageUpdate,
		CurrentTokens: current,
		MaxTokens:     max,
		UsagePercent:  percent,
		Cost:          cost,
	}
}

func NewWarningEvent(current, max int, message string) ContextEvent {
	percent := 0.0
	if max > 0 {
		percent = float64(current) / float64(max) * 100
	}
	return ContextEvent{
		Type:          EventTypeWarning,
		CurrentTokens: current,
		MaxTokens:     max,
		UsagePercent:  percent,
		Message:       message,
	}
}

func NewCompactedEvent(current, max int, message string) ContextEvent {
	percent := 0.0
	if max > 0 {
		percent = float64(current) / float64(max) * 100
	}
	return ContextEvent{
		Type:          EventTypeCompacted,
		CurrentTokens: current,
		MaxTokens:     max,
		UsagePercent:  percent,
		Message:       message,
	}
}

func NewPrunedEvent(tokensSaved int, message string) ContextEvent {
	return ContextEvent{
		Type:    EventTypePruned,
		Message: message,
	}
}

func NewErrorEvent(err error) ContextEvent {
	return ContextEvent{
		Type:    EventTypeError,
		Message: err.Error(),
		Error:   err,
	}
}
