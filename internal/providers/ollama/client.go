package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/taaha3244/potus/internal/providers"
)

type Client struct {
	endpoint string
	client   *http.Client
}

func New(endpoint string) (*Client, error) {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	return &Client{
		endpoint: endpoint,
		client:   &http.Client{},
	}, nil
}

func (c *Client) Name() string {
	return "ollama"
}

func (c *Client) SupportsTools() bool {
	return true
}

func (c *Client) SupportsVision() bool {
	return true
}

func (c *Client) ListModels(ctx context.Context) ([]providers.Model, error) {
	resp, err := http.Get(c.endpoint + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name       string `json:"name"`
			ModifiedAt string `json:"modified_at"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]providers.Model, 0, len(result.Models))
	for _, m := range result.Models {
		models = append(models, providers.Model{
			ID:          m.Name,
			Name:        m.Name,
			Provider:    "ollama",
			ContextSize: 4096,
			Pricing: providers.ModelPricing{
				InputPer1M:  0,
				OutputPer1M: 0,
			},
		})
	}

	return models, nil
}

func (c *Client) Chat(ctx context.Context, req *providers.ChatRequest) (<-chan providers.ChatEvent, error) {
	apiReq := c.buildRequest(req)

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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

	if req.Temperature > 0 {
		apiReq["options"] = map[string]interface{}{
			"temperature": req.Temperature,
		}
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

		content := ""
		images := []string{}
		var toolCalls []map[string]interface{}

		for _, block := range msg.Content {
			switch b := block.(type) {
			case *providers.TextContent:
				content = b.Text
			case *providers.ImageContent:
				images = append(images, b.Source.Data)
			case *providers.ToolUseContent:
				toolCalls = append(toolCalls, map[string]interface{}{
					"function": map[string]interface{}{
						"name":      b.Name,
						"arguments": b.Input,
					},
				})
			case *providers.ToolResultContent:
				content = b.Content
			}
		}

		apiMsg["content"] = content

		if len(images) > 0 {
			apiMsg["images"] = images
		}

		if len(toolCalls) > 0 {
			apiMsg["tool_calls"] = toolCalls
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

func (c *Client) streamResponse(body io.ReadCloser, eventChan chan<- providers.ChatEvent) {
	defer close(eventChan)
	defer body.Close()

	eventChan <- providers.ChatEvent{Type: providers.EventTypeMessageStart}

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Bytes()

		var chunk map[string]interface{}
		if err := json.Unmarshal(line, &chunk); err != nil {
			eventChan <- providers.ChatEvent{
				Type:  providers.EventTypeError,
				Error: fmt.Errorf("failed to parse chunk: %w", err),
			}
			return
		}

		if done, ok := chunk["done"].(bool); ok && done {
			eventChan <- providers.ChatEvent{Type: providers.EventTypeMessageDone}
			break
		}

		if message, ok := chunk["message"].(map[string]interface{}); ok {
			if content, ok := message["content"].(string); ok && content != "" {
				eventChan <- providers.ChatEvent{
					Type:    providers.EventTypeTextDelta,
					Content: content,
				}
			}

			if toolCalls, ok := message["tool_calls"].([]interface{}); ok {
				for _, tc := range toolCalls {
					toolCall := tc.(map[string]interface{})
					if function, ok := toolCall["function"].(map[string]interface{}); ok {
						name, _ := function["name"].(string)
						arguments, _ := function["arguments"].(map[string]interface{})

						eventChan <- providers.ChatEvent{
							Type: providers.EventTypeToolUse,
							ToolUse: &providers.ToolUseContent{
								ID:    fmt.Sprintf("tool_%s", name),
								Name:  name,
								Input: arguments,
							},
						}
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		eventChan <- providers.ChatEvent{
			Type:  providers.EventTypeError,
			Error: fmt.Errorf("scanner error: %w", err),
		}
	}
}
