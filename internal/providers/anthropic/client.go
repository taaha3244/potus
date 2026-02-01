package anthropic

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

const (
	defaultEndpoint = "https://api.anthropic.com/v1/messages"
	apiVersion      = "2023-06-01"
)

type Client struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

func New(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	return &Client{
		apiKey:   apiKey,
		endpoint: defaultEndpoint,
		client:   &http.Client{},
	}, nil
}

func (c *Client) Name() string {
	return "anthropic"
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
			ID:          "claude-opus-4-5-20251101",
			Name:        "Claude Opus 4.5",
			Provider:    "anthropic",
			ContextSize: 200000,
			Pricing: providers.ModelPricing{
				InputPer1M:  15.00,
				OutputPer1M: 75.00,
			},
		},
		{
			ID:          "claude-sonnet-4-5-20250929",
			Name:        "Claude Sonnet 4.5",
			Provider:    "anthropic",
			ContextSize: 200000,
			Pricing: providers.ModelPricing{
				InputPer1M:  3.00,
				OutputPer1M: 15.00,
			},
		},
		{
			ID:          "claude-sonnet-4-20250514",
			Name:        "Claude Sonnet 4",
			Provider:    "anthropic",
			ContextSize: 200000,
			Pricing: providers.ModelPricing{
				InputPer1M:  3.00,
				OutputPer1M: 15.00,
			},
		},
		{
			ID:          "claude-haiku-4-5-20251015",
			Name:        "Claude Haiku 4.5",
			Provider:    "anthropic",
			ContextSize: 200000,
			Pricing: providers.ModelPricing{
				InputPer1M:  1.00,
				OutputPer1M: 5.00,
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
	httpReq.Header.Set("X-API-Key", c.apiKey)
	httpReq.Header.Set("Anthropic-Version", apiVersion)

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
		"model":      req.Model,
		"max_tokens": req.MaxTokens,
		"stream":     true,
	}

	if req.Temperature > 0 {
		apiReq["temperature"] = req.Temperature
	}

	if req.System != "" {
		apiReq["system"] = req.System
	}

	messages := make([]map[string]interface{}, 0, len(req.Messages))
	for _, msg := range req.Messages {
		if msg.Role == providers.RoleSystem {
			continue
		}

		apiMsg := map[string]interface{}{
			"role": string(msg.Role),
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

		messages = append(messages, apiMsg)
	}
	apiReq["messages"] = messages

	if len(req.Tools) > 0 {
		tools := make([]map[string]interface{}, 0, len(req.Tools))
		for _, tool := range req.Tools {
			tools = append(tools, map[string]interface{}{
				"name":         tool.Name,
				"description":  tool.Description,
				"input_schema": tool.InputSchema,
			})
		}
		apiReq["tools"] = tools
	}

	return apiReq
}

func (c *Client) convertContent(blocks []providers.ContentBlock) []map[string]interface{} {
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
				"type": "image",
				"source": map[string]interface{}{
					"type":       b.Source.Type,
					"media_type": b.Source.MediaType,
					"data":       b.Source.Data,
				},
			})
		case *providers.ToolResultContent:
			result = append(result, map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": b.ToolUseID,
				"content":     b.Content,
				"is_error":    b.IsError,
			})
		}
	}

	return result
}

func (c *Client) streamResponse(body io.ReadCloser, eventChan chan<- providers.ChatEvent) {
	defer close(eventChan)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			eventChan <- providers.ChatEvent{
				Type:  providers.EventTypeError,
				Error: fmt.Errorf("failed to parse event: %w", err),
			}
			return
		}

		c.handleEvent(event, eventChan)
	}

	if err := scanner.Err(); err != nil {
		eventChan <- providers.ChatEvent{
			Type:  providers.EventTypeError,
			Error: fmt.Errorf("scanner error: %w", err),
		}
	}
}

func (c *Client) handleEvent(event map[string]interface{}, eventChan chan<- providers.ChatEvent) {
	eventType, _ := event["type"].(string)

	switch eventType {
	case "message_start":
		eventChan <- providers.ChatEvent{
			Type: providers.EventTypeMessageStart,
		}

	case "content_block_start":

	case "content_block_delta":
		delta, ok := event["delta"].(map[string]interface{})
		if !ok {
			return
		}

		deltaType, _ := delta["type"].(string)
		switch deltaType {
		case "text_delta":
			if text, ok := delta["text"].(string); ok {
				eventChan <- providers.ChatEvent{
					Type:    providers.EventTypeTextDelta,
					Content: text,
				}
			}
		case "input_json_delta":

		}

	case "content_block_stop":
		index, _ := event["index"].(float64)
		contentBlock := event["content_block"]
		if contentBlock != nil {
			if blockMap, ok := contentBlock.(map[string]interface{}); ok {
				if blockMap["type"] == "tool_use" {
					toolUse := &providers.ToolUseContent{
						ID:    blockMap["id"].(string),
						Name:  blockMap["name"].(string),
						Input: blockMap["input"].(map[string]interface{}),
					}
					eventChan <- providers.ChatEvent{
						Type:    providers.EventTypeToolUse,
						ToolUse: toolUse,
					}
				}
			}
		}
		_ = index

	case "message_delta":

	case "message_stop":
		eventChan <- providers.ChatEvent{
			Type: providers.EventTypeMessageDone,
		}
	}
}
