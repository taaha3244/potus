package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/taaha3244/potus/internal/tools"
)

// SearchTool performs web searches
type SearchTool struct {
	client *http.Client
}

// NewSearchTool creates a new web search tool
func NewSearchTool() *SearchTool {
	return &SearchTool{
		client: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

func (t *SearchTool) Name() string {
	return "web_search"
}

func (t *SearchTool) Description() string {
	return "Search the web for information. Returns a list of search results with titles, URLs, and snippets. Useful for finding documentation, solutions, or current information."
}

func (t *SearchTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The search query",
			},
			"num_results": map[string]interface{}{
				"type":        "integer",
				"description": "Number of results to return (default: 10, max: 20)",
			},
		},
		"required": []string{"query"},
	}
}

func (t *SearchTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	numResults := 10
	if nr, ok := params["num_results"].(float64); ok {
		numResults = int(nr)
		if numResults > 20 {
			numResults = 20
		}
		if numResults < 1 {
			numResults = 1
		}
	}

	// Use DuckDuckGo HTML search (no API key required)
	results, err := t.searchDuckDuckGo(ctx, query, numResults)
	if err != nil {
		return tools.NewErrorResult(fmt.Errorf("search failed: %w", err)), nil
	}

	if len(results) == 0 {
		return tools.NewResult("No results found for: " + query), nil
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Search results for: %s\n\n", query))

	for i, result := range results {
		output.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))
		output.WriteString(fmt.Sprintf("   URL: %s\n", result.URL))
		if result.Snippet != "" {
			output.WriteString(fmt.Sprintf("   %s\n", result.Snippet))
		}
		output.WriteString("\n")
	}

	return tools.NewResult(output.String()), nil
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string
	URL     string
	Snippet string
}

// searchDuckDuckGo uses DuckDuckGo's instant answer API
func (t *SearchTool) searchDuckDuckGo(ctx context.Context, query string, numResults int) ([]SearchResult, error) {
	// DuckDuckGo Instant Answer API
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1",
		url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "POTUS/1.0 (AI Coding Assistant)")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse DuckDuckGo response
	var ddgResp struct {
		Abstract       string `json:"Abstract"`
		AbstractURL    string `json:"AbstractURL"`
		AbstractSource string `json:"AbstractSource"`
		Answer         string `json:"Answer"`
		AnswerType     string `json:"AnswerType"`
		RelatedTopics  []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"RelatedTopics"`
		Results []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"Results"`
	}

	if err := json.Unmarshal(body, &ddgResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var results []SearchResult

	// Add abstract if available
	if ddgResp.Abstract != "" && ddgResp.AbstractURL != "" {
		results = append(results, SearchResult{
			Title:   ddgResp.AbstractSource,
			URL:     ddgResp.AbstractURL,
			Snippet: ddgResp.Abstract,
		})
	}

	// Add answer if available
	if ddgResp.Answer != "" {
		results = append(results, SearchResult{
			Title:   "Direct Answer",
			URL:     "",
			Snippet: ddgResp.Answer,
		})
	}

	// Add results
	for _, r := range ddgResp.Results {
		if len(results) >= numResults {
			break
		}
		if r.FirstURL != "" {
			results = append(results, SearchResult{
				Title:   extractTitle(r.Text),
				URL:     r.FirstURL,
				Snippet: r.Text,
			})
		}
	}

	// Add related topics
	for _, topic := range ddgResp.RelatedTopics {
		if len(results) >= numResults {
			break
		}
		if topic.FirstURL != "" {
			results = append(results, SearchResult{
				Title:   extractTitle(topic.Text),
				URL:     topic.FirstURL,
				Snippet: topic.Text,
			})
		}
	}

	return results, nil
}

// extractTitle tries to extract a title from search result text
func extractTitle(text string) string {
	// Take first sentence or first 100 chars
	if idx := strings.Index(text, " - "); idx > 0 && idx < 100 {
		return text[:idx]
	}
	if len(text) > 100 {
		return text[:100] + "..."
	}
	return text
}
