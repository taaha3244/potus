package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/taaha3244/potus/internal/tools"
)

const (
	// DefaultTimeout for web requests
	DefaultTimeout = 30 * time.Second

	// MaxBodySize limits response size to prevent memory issues
	MaxBodySize = 1024 * 1024 // 1MB

	// MaxOutputLength limits the output to prevent token overflow
	MaxOutputLength = 50000
)

// FetchTool fetches content from URLs
type FetchTool struct {
	client *http.Client
}

// NewFetchTool creates a new web fetch tool
func NewFetchTool() *FetchTool {
	return &FetchTool{
		client: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

func (t *FetchTool) Name() string {
	return "web_fetch"
}

func (t *FetchTool) Description() string {
	return "Fetch content from a URL. Converts HTML to readable text. Useful for reading documentation, API references, or any web page."
}

func (t *FetchTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "The URL to fetch content from",
			},
			"extract_text": map[string]interface{}{
				"type":        "boolean",
				"description": "If true, extracts readable text from HTML (default: true)",
			},
		},
		"required": []string{"url"},
	}
}

func (t *FetchTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.Result, error) {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return nil, fmt.Errorf("url parameter is required")
	}

	// Validate URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	extractText := true
	if et, ok := params["extract_text"].(bool); ok {
		extractText = et
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return tools.NewErrorResult(fmt.Errorf("failed to create request: %w", err)), nil
	}

	// Set user agent to appear as a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; POTUS/1.0; +https://github.com/taaha3244/potus)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	// Execute request
	resp, err := t.client.Do(req)
	if err != nil {
		return tools.NewErrorResult(fmt.Errorf("failed to fetch URL: %w", err)), nil
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode >= 400 {
		return tools.NewErrorResult(fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)), nil
	}

	// Read body with size limit
	limitedReader := io.LimitReader(resp.Body, MaxBodySize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return tools.NewErrorResult(fmt.Errorf("failed to read response: %w", err)), nil
	}

	content := string(body)

	// Extract text from HTML if requested
	contentType := resp.Header.Get("Content-Type")
	if extractText && strings.Contains(contentType, "text/html") {
		content = extractTextFromHTML(content)
	}

	// Truncate if too long
	if len(content) > MaxOutputLength {
		content = content[:MaxOutputLength] + "\n\n[Content truncated due to length...]"
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("URL: %s\n", url))
	output.WriteString(fmt.Sprintf("Status: %d\n", resp.StatusCode))
	output.WriteString(fmt.Sprintf("Content-Type: %s\n", contentType))
	output.WriteString(fmt.Sprintf("Content-Length: %d bytes\n\n", len(body)))
	output.WriteString(content)

	return tools.NewResult(output.String()), nil
}

// extractTextFromHTML removes HTML tags and extracts readable text
func extractTextFromHTML(html string) string {
	// Remove script and style elements
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	html = scriptRe.ReplaceAllString(html, "")

	styleRe := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	html = styleRe.ReplaceAllString(html, "")

	// Remove HTML comments
	commentRe := regexp.MustCompile(`(?s)<!--.*?-->`)
	html = commentRe.ReplaceAllString(html, "")

	// Replace common block elements with newlines
	blockElements := []string{"</p>", "</div>", "</h1>", "</h2>", "</h3>", "</h4>", "</h5>", "</h6>", "</li>", "</tr>", "<br>", "<br/>", "<br />"}
	for _, elem := range blockElements {
		html = strings.ReplaceAll(html, elem, elem+"\n")
	}

	// Remove all HTML tags
	tagRe := regexp.MustCompile(`<[^>]+>`)
	text := tagRe.ReplaceAllString(html, "")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&apos;", "'")

	// Clean up whitespace
	// Replace multiple spaces with single space
	spaceRe := regexp.MustCompile(`[ \t]+`)
	text = spaceRe.ReplaceAllString(text, " ")

	// Replace multiple newlines with double newline
	newlineRe := regexp.MustCompile(`\n{3,}`)
	text = newlineRe.ReplaceAllString(text, "\n\n")

	// Trim each line
	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}
