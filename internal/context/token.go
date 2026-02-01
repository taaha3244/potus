package context

import (
	"encoding/json"
	"time"

	"github.com/taaha3244/potus/internal/providers"
)

type TokenEstimator interface {
	EstimateTokens(text string) int
	EstimateMessage(msg *providers.Message) int
	EstimateMessages(msgs []providers.Message) int
}

type SimpleEstimator struct {
	CharsPerToken float64
}

func NewSimpleEstimator() *SimpleEstimator {
	return &SimpleEstimator{CharsPerToken: 4.0}
}

func (e *SimpleEstimator) EstimateTokens(text string) int {
	if e.CharsPerToken <= 0 {
		e.CharsPerToken = 4.0
	}
	return int(float64(len(text)) / e.CharsPerToken)
}

func (e *SimpleEstimator) EstimateMessage(msg *providers.Message) int {
	if msg == nil {
		return 0
	}

	total := 4

	for _, block := range msg.Content {
		switch b := block.(type) {
		case *providers.TextContent:
			total += e.EstimateTokens(b.Text)

		case *providers.ToolUseContent:
			total += 20
			total += e.EstimateTokens(b.Name)
			if b.Input != nil {
				inputJSON, err := json.Marshal(b.Input)
				if err == nil {
					total += e.EstimateTokens(string(inputJSON))
				}
			}

		case *providers.ToolResultContent:
			total += 10
			total += e.EstimateTokens(b.Content)

		case *providers.ImageContent:
			total += 1500
		}
	}

	return total
}

func (e *SimpleEstimator) EstimateMessages(msgs []providers.Message) int {
	total := 0
	for i := range msgs {
		total += e.EstimateMessage(&msgs[i])
	}
	return total
}

type TokenInfo struct {
	MessageIndex int
	Tokens       int
	Role         providers.MessageRole
	Timestamp    time.Time
	IsPrunable   bool
	ToolName     string
	ToolUseID    string
}

func NewTokenInfo(index int, msg *providers.Message, estimator TokenEstimator) TokenInfo {
	info := TokenInfo{
		MessageIndex: index,
		Tokens:       estimator.EstimateMessage(msg),
		Role:         msg.Role,
		Timestamp:    time.Now(),
		IsPrunable:   isPrunableMessage(msg),
	}

	for _, block := range msg.Content {
		if tr, ok := block.(*providers.ToolResultContent); ok {
			info.ToolUseID = tr.ToolUseID
			break
		}
	}

	return info
}

func isPrunableMessage(msg *providers.Message) bool {
	if msg.Role == providers.RoleTool {
		return true
	}

	for _, block := range msg.Content {
		if _, ok := block.(*providers.ToolResultContent); ok {
			return true
		}
	}

	return false
}

func EstimateSystemPrompt(prompt string) int {
	e := NewSimpleEstimator()
	return e.EstimateTokens(prompt) + 10
}
