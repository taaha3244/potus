package context

import (
	"context"
	"fmt"
	"strings"

	"github.com/taaha3244/potus/internal/providers"
)

const (
	DefaultProtectedMessages = 6
	DefaultMaxSummaryTokens  = 1000
	SummarySystemPrompt      = "You are a conversation summarizer. Be concise and preserve key technical details. Focus on information that would be needed to continue the conversation effectively."
)

const SummaryPromptTemplate = `Summarize this conversation concisely, preserving:
1. Key decisions made
2. Important file paths and code discussed
3. Any errors encountered and solutions found
4. Current task state and next steps

Keep the summary focused and actionable. Maximum 500 words.

Conversation:
%s

Provide a concise summary:`

type Compactor struct {
	provider          providers.Provider
	estimator         TokenEstimator
	protectedMessages int
	maxSummaryTokens  int
}

type CompactorConfig struct {
	Provider          providers.Provider
	Estimator         TokenEstimator
	ProtectedMessages int
	MaxSummaryTokens  int
}

type CompactResult struct {
	OriginalMessages   int
	CompactedMessages  int
	OriginalTokens     int
	CompactedTokens    int
	SummarizedMessages int
	Summary            string
}

func NewCompactor(cfg CompactorConfig) *Compactor {
	protectedMessages := cfg.ProtectedMessages
	if protectedMessages <= 0 {
		protectedMessages = DefaultProtectedMessages
	}

	maxSummaryTokens := cfg.MaxSummaryTokens
	if maxSummaryTokens <= 0 {
		maxSummaryTokens = DefaultMaxSummaryTokens
	}

	estimator := cfg.Estimator
	if estimator == nil {
		estimator = NewSimpleEstimator()
	}

	return &Compactor{
		provider:          cfg.Provider,
		estimator:         estimator,
		protectedMessages: protectedMessages,
		maxSummaryTokens:  maxSummaryTokens,
	}
}

func (c *Compactor) Compact(ctx context.Context, messages []providers.Message) ([]providers.Message, CompactResult, error) {
	result := CompactResult{
		OriginalMessages: len(messages),
		OriginalTokens:   c.estimator.EstimateMessages(messages),
	}

	if len(messages) <= c.protectedMessages {
		result.CompactedMessages = len(messages)
		result.CompactedTokens = result.OriginalTokens
		return messages, result, nil
	}

	toSummarize := messages[:len(messages)-c.protectedMessages]
	toPreserve := messages[len(messages)-c.protectedMessages:]
	result.SummarizedMessages = len(toSummarize)

	summary, err := c.generateSummary(ctx, toSummarize)
	if err != nil {
		return nil, result, fmt.Errorf("failed to generate summary: %w", err)
	}
	result.Summary = summary

	compactedMessages := make([]providers.Message, 0, len(toPreserve)+2)

	compactedMessages = append(compactedMessages, providers.Message{
		Role: providers.RoleUser,
		Content: []providers.ContentBlock{
			&providers.TextContent{
				Text: fmt.Sprintf("[Previous Conversation Summary]\n%s\n[End Summary]", summary),
			},
		},
	})

	compactedMessages = append(compactedMessages, providers.Message{
		Role: providers.RoleAssistant,
		Content: []providers.ContentBlock{
			&providers.TextContent{
				Text: "I understand the context from our previous conversation. I'll continue helping you with this understanding.",
			},
		},
	})

	compactedMessages = append(compactedMessages, toPreserve...)

	result.CompactedMessages = len(compactedMessages)
	result.CompactedTokens = c.estimator.EstimateMessages(compactedMessages)

	return compactedMessages, result, nil
}

func (c *Compactor) generateSummary(ctx context.Context, messages []providers.Message) (string, error) {
	conversationText := c.formatConversation(messages)

	summarizePrompt := fmt.Sprintf(SummaryPromptTemplate, conversationText)

	req := &providers.ChatRequest{
		Messages: []providers.Message{
			{
				Role: providers.RoleUser,
				Content: []providers.ContentBlock{
					&providers.TextContent{Text: summarizePrompt},
				},
			},
		},
		MaxTokens: c.maxSummaryTokens,
		System:    SummarySystemPrompt,
	}

	events, err := c.provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("provider chat failed: %w", err)
	}

	var summary strings.Builder
	for event := range events {
		switch event.Type {
		case providers.EventTypeTextDelta:
			summary.WriteString(event.Content)
		case providers.EventTypeError:
			return "", event.Error
		}
	}

	return strings.TrimSpace(summary.String()), nil
}

func (c *Compactor) formatConversation(messages []providers.Message) string {
	var builder strings.Builder

	for _, msg := range messages {
		role := formatRole(msg.Role)

		for _, block := range msg.Content {
			switch b := block.(type) {
			case *providers.TextContent:
				builder.WriteString(fmt.Sprintf("%s: %s\n", role, b.Text))

			case *providers.ToolUseContent:
				builder.WriteString(fmt.Sprintf("%s: [Called tool: %s]\n", role, b.Name))

			case *providers.ToolResultContent:
				content := b.Content
				if len(content) > 500 {
					content = content[:500] + "...[truncated]"
				}
				status := "success"
				if b.IsError {
					status = "error"
				}
				builder.WriteString(fmt.Sprintf("Tool Result (%s): %s\n", status, content))
			}
		}
	}

	return builder.String()
}

func formatRole(role providers.MessageRole) string {
	switch role {
	case providers.RoleUser:
		return "User"
	case providers.RoleAssistant:
		return "Assistant"
	case providers.RoleSystem:
		return "System"
	case providers.RoleTool:
		return "Tool"
	default:
		return string(role)
	}
}

func (c *Compactor) ShouldCompact(messages []providers.Message, currentTokens, maxTokens int) bool {
	if len(messages) <= c.protectedMessages+2 {
		return false
	}
	return float64(currentTokens) >= float64(maxTokens)*0.90
}

func (c *Compactor) EstimateSavings(messages []providers.Message) int {
	if len(messages) <= c.protectedMessages {
		return 0
	}

	toSummarize := messages[:len(messages)-c.protectedMessages]
	toSummarizeTokens := c.estimator.EstimateMessages(toSummarize)

	estimatedSummaryTokens := int(float64(toSummarizeTokens)*0.20) + 200

	return toSummarizeTokens - estimatedSummaryTokens
}
