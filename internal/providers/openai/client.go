package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/taaha3244/potus/internal/providers"
)

const defaultEndpoint = "https://api.openai.com/v1/chat/completions"

type Client struct {
	apiKey       string
	organization string
	endpoint     string
	client       *http.Client
}

func New(apiKey, organization string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	return &Client{
		apiKey:       apiKey,
		organization: organization,
		endpoint:     defaultEndpoint,
		client:       &http.Client{},
	}, nil
}

func (c *Client) Name() string {
	return "openai"
}

func (c *Client) SupportsTools() bool {
	return true
}

func (c *Client) SupportsVision() bool {
	return true
}

func (c *Client) ListModels(ctx context.Context) ([]providers.Model, error) {
	return []providers.Model{
		{
			ID:          "gpt-5.2",
			Name:        "GPT-5.2",
			Provider:    "openai",
			ContextSize: 400000,
			Pricing: providers.ModelPricing{
				InputPer1M:  5.00,
				OutputPer1M: 15.00,
			},
		},
		{
			ID:          "gpt-5",
			Name:        "GPT-5",
			Provider:    "openai",
			ContextSize: 200000,
			Pricing: providers.ModelPricing{
				InputPer1M:  5.00,
				OutputPer1M: 15.00,
			},
		},
		{
			ID:          "gpt-5-mini",
			Name:        "GPT-5 Mini",
			Provider:    "openai",
			ContextSize: 200000,
			Pricing: providers.ModelPricing{
				InputPer1M:  0.30,
				OutputPer1M: 1.20,
			},
		},
		{
			ID:          "gpt-4.1",
			Name:        "GPT-4.1",
			Provider:    "openai",
			ContextSize: 1000000,
			Pricing: providers.ModelPricing{
				InputPer1M:  2.00,
				OutputPer1M: 8.00,
			},
		},
		{
			ID:          "o4-mini",
			Name:        "O4 Mini",
			Provider:    "openai",
			ContextSize: 200000,
			Pricing: providers.ModelPricing{
				InputPer1M:  1.10,
				OutputPer1M: 4.40,
			},
		},
	}, nil
}

func (c *Client) Chat(ctx context.Context, req *providers.ChatRequest) (<-chan providers.ChatEvent, error) {
	apiReq := c.buildRequest(req)

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	if c.organization != "" {
		httpReq.Header.Set("OpenAI-Organization", c.organization)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	eventChan := make(chan providers.ChatEvent, 10)

	go c.streamResponse(resp.Body, eventChan)

	return eventChan, nil
}

func (c *Client) buildRequest(req *providers.ChatRequest) map[string]interface{} {
	apiReq := map[string]interface{}{
		"model":  req.Model,
		"stream": true,
	}

	if req.MaxTokens > 0 {
		apiReq["max_tokens"] = req.MaxTokens
	}

	if req.Temperature > 0 {
		apiReq["temperature"] = req.Temperature
	}

	messages := make([]map[string]interface{}, 0, len(req.Messages))

	if req.System != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": req.System,
		})
	}

	for _, msg := range req.Messages {
		apiMsg := map[string]interface{}{
			"role": c.convertRole(msg.Role),
		}

		if len(msg.Content) == 1 {
			if textContent, ok := msg.Content[0].(*providers.TextContent); ok {
				apiMsg["content"] = textContent.Text
			} else {
				apiMsg["content"] = c.convertContent(msg.Content)
			}
		} else {
			apiMsg["content"] = c.convertContent(msg.Content)
		}

		if msg.Role == providers.RoleAssistant {
			toolCalls := c.extractToolCalls(msg.Content)
			if len(toolCalls) > 0 {
				apiMsg["tool_calls"] = toolCalls
				if len(msg.Content) == len(toolCalls) {
					delete(apiMsg, "content")
				}
			}
		}

		if msg.Role == providers.RoleTool {
			if len(msg.Content) > 0 {
				if toolResult, ok := msg.Content[0].(*providers.ToolResultContent); ok {
					apiMsg["tool_call_id"] = toolResult.ToolUseID
					apiMsg["content"] = toolResult.Content
				}
			}
		}

		messages = append(messages, apiMsg)
	}

	apiReq["messages"] = messages

	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, 0, len(req.Tools))
		for _, tool := range req.Tools {
			tools = append(tools, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        tool.Name,
					"description": tool.Description,
					"parameters":  tool.InputSchema,
				},
			})
		}
		apiReq["tools"] = tools
	}

	return apiReq
}

func (c *Client) convertRole(role providers.MessageRole) string {
	switch role {
	case providers.RoleTool:
		return "tool"
	default:
		return string(role)
	}
}

func (c *Client) convertContent(blocks []providers.ContentBlock) interface{} {
	if len(blocks) == 0 {
		return ""
	}

	if len(blocks) == 1 {
		if text, ok := blocks[0].(*providers.TextContent); ok {
			return text.Text
		}
	}

	result := make([]map[string]interface{}, 0, len(blocks))
	for _, block := range blocks {
		switch b := block.(type) {
		case *providers.TextContent:
			result = append(result, map[string]interface{}{
				"type": "text",
				"text": b.Text,
			})
		case *providers.ImageContent:
			result = append(result, map[string]interface{}{
				"type": "image_url",
				"image_url": map[string]interface{}{
					"url": fmt.Sprintf("data:%s;base64,%s", b.Source.MediaType, b.Source.Data),
				},
			})
		}
	}

	return result
}

func (c *Client) extractToolCalls(blocks []providers.ContentBlock) []map[string]interface{} {
	var toolCalls []map[string]interface{}

	for _, block := range blocks {
		if toolUse, ok := block.(*providers.ToolUseContent); ok {
			args, _ := json.Marshal(toolUse.Input)
			toolCalls = append(toolCalls, map[string]interface{}{
				"id":   toolUse.ID,
				"type": "function",
				"function": map[string]interface{}{
					"name":      toolUse.Name,
					"arguments": string(args),
				},
			})
		}
	}

	return toolCalls
}

func (c *Client) streamResponse(body io.ReadCloser, eventChan chan<- providers.ChatEvent) {
	defer close(eventChan)
	defer body.Close()

	eventChan <- providers.ChatEvent{Type: providers.EventTypeMessageStart}

	scanner := bufio.NewScanner(body)
	currentToolCall := make(map[string]interface{})
	accumulatedArgs := ""

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			eventChan <- providers.ChatEvent{
				Type:  providers.EventTypeError,
				Error: fmt.Errorf("failed to parse chunk: %w", err),
			}
			return
		}

		choices, ok := chunk["choices"].([]interface{})
		if !ok || len(choices) == 0 {
			continue
		}

		choice := choices[0].(map[string]interface{})
		delta, ok := choice["delta"].(map[string]interface{})
		if !ok {
			continue
		}

		if content, ok := delta["content"].(string); ok && content != "" {
			eventChan <- providers.ChatEvent{
				Type:    providers.EventTypeTextDelta,
				Content: content,
			}
		}

		if toolCalls, ok := delta["tool_calls"].([]interface{}); ok {
			for _, tc := range toolCalls {
				toolCall := tc.(map[string]interface{})

				if id, ok := toolCall["id"].(string); ok {
					currentToolCall["id"] = id
				}

				if function, ok := toolCall["function"].(map[string]interface{}); ok {
					if name, ok := function["name"].(string); ok {
						currentToolCall["name"] = name
					}
					if args, ok := function["arguments"].(string); ok {
						accumulatedArgs += args
					}
				}
			}
		}

		if finishReason, ok := choice["finish_reason"].(string); ok && finishReason == "tool_calls" {
			if currentToolCall["id"] != nil && currentToolCall["name"] != nil {
				var input map[string]interface{}
				json.Unmarshal([]byte(accumulatedArgs), &input)

				eventChan <- providers.ChatEvent{
					Type: providers.EventTypeToolUse,
					ToolUse: &providers.ToolUseContent{
						ID:    currentToolCall["id"].(string),
						Name:  currentToolCall["name"].(string),
						Input: input,
					},
				}
			}
		}
	}

	eventChan <- providers.ChatEvent{Type: providers.EventTypeMessageDone}

	if err := scanner.Err(); err != nil {
		eventChan <- providers.ChatEvent{
			Type:  providers.EventTypeError,
			Error: fmt.Errorf("scanner error: %w", err),
		}
	}
}
