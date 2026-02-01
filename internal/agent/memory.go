package agent

import (
	"sync"

	"github.com/taaha3244/potus/internal/context"
	"github.com/taaha3244/potus/internal/providers"
)

type Memory struct {
	mu           sync.RWMutex
	messages     []providers.Message
	tokenInfo    []context.TokenInfo
	estimator    context.TokenEstimator
	totalTokens  int
	systemTokens int
}

func NewMemory(estimator context.TokenEstimator) *Memory {
	if estimator == nil {
		estimator = context.NewSimpleEstimator()
	}
	return &Memory{
		messages:  make([]providers.Message, 0),
		tokenInfo: make([]context.TokenInfo, 0),
		estimator: estimator,
	}
}

func (m *Memory) AddUserMessage(content string) int {
	msg := providers.Message{
		Role: providers.RoleUser,
		Content: []providers.ContentBlock{
			&providers.TextContent{Text: content},
		},
	}
	return m.AddMessage(&msg)
}

func (m *Memory) AddMessage(msg *providers.Message) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	tokens := m.estimator.EstimateMessage(msg)
	info := context.TokenInfo{
		MessageIndex: len(m.messages),
		Tokens:       tokens,
		Role:         msg.Role,
		IsPrunable:   m.isPrunable(msg),
		ToolName:     m.extractToolName(msg),
		ToolUseID:    m.extractToolUseID(msg),
	}

	m.messages = append(m.messages, *msg)
	m.tokenInfo = append(m.tokenInfo, info)
	m.totalTokens += tokens

	return tokens
}

func (m *Memory) GetMessages() []providers.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	msgs := make([]providers.Message, len(m.messages))
	copy(msgs, m.messages)
	return msgs
}

func (m *Memory) GetTokenInfo() []context.TokenInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := make([]context.TokenInfo, len(m.tokenInfo))
	copy(info, m.tokenInfo)
	return info
}

func (m *Memory) GetTotalTokens() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalTokens + m.systemTokens
}

func (m *Memory) GetMessageTokens() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalTokens
}

func (m *Memory) SetSystemTokens(tokens int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.systemTokens = tokens
}

func (m *Memory) GetSystemTokens() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.systemTokens
}

func (m *Memory) ReplaceMessages(msgs []providers.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = msgs
	m.tokenInfo = make([]context.TokenInfo, len(msgs))
	m.totalTokens = 0

	for i := range msgs {
		tokens := m.estimator.EstimateMessage(&msgs[i])
		m.tokenInfo[i] = context.TokenInfo{
			MessageIndex: i,
			Tokens:       tokens,
			Role:         msgs[i].Role,
			IsPrunable:   m.isPrunable(&msgs[i]),
			ToolName:     m.extractToolName(&msgs[i]),
			ToolUseID:    m.extractToolUseID(&msgs[i]),
		}
		m.totalTokens += tokens
	}
}

func (m *Memory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = make([]providers.Message, 0)
	m.tokenInfo = make([]context.TokenInfo, 0)
	m.totalTokens = 0
}

func (m *Memory) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.messages)
}

func (m *Memory) GetEstimator() context.TokenEstimator {
	return m.estimator
}

func (m *Memory) isPrunable(msg *providers.Message) bool {
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

func (m *Memory) extractToolName(msg *providers.Message) string {
	for _, block := range msg.Content {
		if tu, ok := block.(*providers.ToolUseContent); ok {
			return tu.Name
		}
	}
	return ""
}

func (m *Memory) extractToolUseID(msg *providers.Message) string {
	for _, block := range msg.Content {
		if tr, ok := block.(*providers.ToolResultContent); ok {
			return tr.ToolUseID
		}
	}
	return ""
}

func (m *Memory) LastMessage() *providers.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.messages) == 0 {
		return nil
	}
	msg := m.messages[len(m.messages)-1]
	return &msg
}

type TokenSummary struct {
	TotalTokens    int
	SystemTokens   int
	MessageTokens  int
	MessageCount   int
	PrunableTokens int
}

func (m *Memory) GetTokenSummary() TokenSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	prunableTokens := 0
	for _, info := range m.tokenInfo {
		if info.IsPrunable {
			prunableTokens += info.Tokens
		}
	}

	return TokenSummary{
		TotalTokens:    m.totalTokens + m.systemTokens,
		SystemTokens:   m.systemTokens,
		MessageTokens:  m.totalTokens,
		MessageCount:   len(m.messages),
		PrunableTokens: prunableTokens,
	}
}
