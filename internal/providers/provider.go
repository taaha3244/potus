package providers

import (
	"context"
	"time"
)

const DefaultTimeout = 30 * time.Second

type Provider interface {
	Chat(ctx context.Context, req *ChatRequest) (<-chan ChatEvent, error)
	ListModels(ctx context.Context) ([]Model, error)
	SupportsTools() bool
	SupportsVision() bool
	Name() string
}

type ChatRequest struct {
	Messages    []Message
	Tools       []Tool
	MaxTokens   int
	Temperature float64
	Model       string
	System      string
}

type Message struct {
	Role    MessageRole
	Content []ContentBlock
}

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
)

type ContentBlock interface {
	Type() ContentType
}

type ContentType string

const (
	ContentTypeText       ContentType = "text"
	ContentTypeImage      ContentType = "image"
	ContentTypeToolUse    ContentType = "tool_use"
	ContentTypeToolResult ContentType = "tool_result"
)

type TextContent struct {
	Text string
}

func (t *TextContent) Type() ContentType { return ContentTypeText }

type ImageContent struct {
	Source ImageSource
}

func (i *ImageContent) Type() ContentType { return ContentTypeImage }

type ImageSource struct {
	Type      string
	MediaType string
	Data      string
}

type ToolUseContent struct {
	ID    string
	Name  string
	Input map[string]interface{}
}

func (t *ToolUseContent) Type() ContentType { return ContentTypeToolUse }

type ToolResultContent struct {
	ToolUseID string
	Content   string
	IsError   bool
}

func (t *ToolResultContent) Type() ContentType { return ContentTypeToolResult }

type Tool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}

type ChatEvent struct {
	Type    EventType
	Content string
	ToolUse *ToolUseContent
	Usage   *Usage
	Error   error
}

type EventType string

const (
	EventTypeTextDelta    EventType = "text_delta"
	EventTypeToolUse      EventType = "tool_use"
	EventTypeMessageStart EventType = "message_start"
	EventTypeMessageDone  EventType = "message_done"
	EventTypeError        EventType = "error"
)

type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

type Model struct {
	ID          string
	Name        string
	Provider    string
	ContextSize int
	Pricing     ModelPricing
}

type ModelPricing struct {
	InputPer1M  float64
	OutputPer1M float64
	CachedPer1M float64
}
