package context

import (
	"github.com/taaha3244/potus/internal/providers"
)

var DefaultProtectedTools = []string{
	"file_read",
	"read_file",
	"search_content",
	"grep",
	"glob",
}

type Pruner struct {
	protectedTools  map[string]bool
	protectionRatio float64
}

type PrunerConfig struct {
	ProtectedTools  []string
	ProtectionRatio float64
}

type PruneResult struct {
	OriginalMessages int
	PrunedMessages   int
	TokensSaved      int
	MessagesPruned   int
}

func NewPruner(cfg PrunerConfig) *Pruner {
	protectedTools := make(map[string]bool)

	for _, tool := range DefaultProtectedTools {
		protectedTools[tool] = true
	}

	for _, tool := range cfg.ProtectedTools {
		protectedTools[tool] = true
	}

	protectionRatio := cfg.ProtectionRatio
	if protectionRatio <= 0 || protectionRatio >= 1.0 {
		protectionRatio = 0.30
	}

	return &Pruner{
		protectedTools:  protectedTools,
		protectionRatio: protectionRatio,
	}
}

func (p *Pruner) Prune(messages []providers.Message, tokenInfo []TokenInfo) ([]providers.Message, PruneResult) {
	result := PruneResult{
		OriginalMessages: len(messages),
	}

	if len(messages) == 0 || len(tokenInfo) == 0 {
		result.PrunedMessages = len(messages)
		return messages, result
	}

	totalTokens := 0
	for _, info := range tokenInfo {
		totalTokens += info.Tokens
	}

	protectedThreshold := int(float64(totalTokens) * p.protectionRatio)

	runningTokens := 0
	cutoffIndex := len(messages)
	for i := len(tokenInfo) - 1; i >= 0; i-- {
		runningTokens += tokenInfo[i].Tokens
		if runningTokens >= protectedThreshold {
			cutoffIndex = i
			break
		}
	}

	prunedMessages := make([]providers.Message, 0, len(messages))
	estimator := NewSimpleEstimator()

	for i, msg := range messages {
		if i >= cutoffIndex {
			prunedMessages = append(prunedMessages, msg)
			continue
		}

		if i < len(tokenInfo) && tokenInfo[i].IsPrunable {
			if p.isProtectedTool(tokenInfo[i].ToolName) {
				prunedMessages = append(prunedMessages, msg)
				continue
			}

			prunedMsg := p.pruneMessage(msg)
			oldTokens := tokenInfo[i].Tokens
			newTokens := estimator.EstimateMessage(&prunedMsg)
			result.TokensSaved += oldTokens - newTokens
			result.MessagesPruned++
			prunedMessages = append(prunedMessages, prunedMsg)
		} else {
			prunedMessages = append(prunedMessages, msg)
		}
	}

	result.PrunedMessages = len(prunedMessages)
	return prunedMessages, result
}

func (p *Pruner) pruneMessage(msg providers.Message) providers.Message {
	newContent := make([]providers.ContentBlock, 0, len(msg.Content))

	for _, block := range msg.Content {
		switch b := block.(type) {
		case *providers.ToolResultContent:
			newContent = append(newContent, &providers.ToolResultContent{
				ToolUseID: b.ToolUseID,
				Content:   "[Previous tool result pruned for context management]",
				IsError:   b.IsError,
			})
		default:
			newContent = append(newContent, block)
		}
	}

	return providers.Message{
		Role:    msg.Role,
		Content: newContent,
	}
}

func (p *Pruner) isProtectedTool(toolName string) bool {
	return p.protectedTools[toolName]
}

func (p *Pruner) AddProtectedTool(toolName string) {
	p.protectedTools[toolName] = true
}

func (p *Pruner) RemoveProtectedTool(toolName string) {
	delete(p.protectedTools, toolName)
}

func (p *Pruner) GetProtectedTools() []string {
	tools := make([]string, 0, len(p.protectedTools))
	for tool := range p.protectedTools {
		tools = append(tools, tool)
	}
	return tools
}

func (p *Pruner) ShouldPrune(tokenInfo []TokenInfo) bool {
	prunableTokens := 0
	totalTokens := 0

	for _, info := range tokenInfo {
		totalTokens += info.Tokens
		if info.IsPrunable && !p.isProtectedTool(info.ToolName) {
			prunableTokens += info.Tokens
		}
	}

	return prunableTokens > 0 && float64(prunableTokens)/float64(totalTokens) >= 0.10
}
