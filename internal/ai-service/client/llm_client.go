package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// LLMClient provide interface to Large Language Model APIs
type LLMClient struct {
	baseURL    string
	apiKey     string
	model      string
	timeout    time.Duration
	httpClient *http.Client
	logger     *slog.Logger
}

// LLMRequest represent the request payload for LLM API
type LLMRequest struct {
	Model          string         `json:"model"`
	Messages       []Message      `json:"messages"`
	ResponseFormat ResponseFormat `json:"response_format"`
}

// Message represent a single message in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ResponseFormat specify the format of the LLM response
type ResponseFormat struct {
	Type string `json:"type"`
}

// LLMResponse represent the response from LLM API
type LLMResponse struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage,omitempty"`
}

// Choice represent a single response choice
type Choice struct {
	Index   int     `json:"index"`
	Message Message `json:"message"`
}

// Usage represent token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ProcessingResult contains the result of article processing
type ProcessingResult struct {
	Summary string
}

// LLMClientInterface define the interface for LLM clients
type LLMClientInterface interface {
	ProcessArticle(ctx context.Context, title, content string) (*ProcessingResult, error)
	GetModel() string
}

// NewLLMClient create a new LLM client instance
func NewLLMClient(baseURL, apiKey, model string, timeout time.Duration, logger *slog.Logger) *LLMClient {
	return &LLMClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// ProcessArticle process article content using LLM and returns summary and tags
func (c *LLMClient) ProcessArticle(ctx context.Context, title, content string) (*ProcessingResult, error) {
	// create prompt for article processing
	prompt := c.createArticleProcessingPrompt(title, content)

	req := LLMRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		ResponseFormat: ResponseFormat{
			Type: "text",
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// TODO: request url should be configurable
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	c.logger.Debug("sending request to LLM API", "url", httpReq.URL.String(), "model", c.model)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("LLM API request failed", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("LLM API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var llmResp LLMResponse
	if err := json.Unmarshal(body, &llmResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(llmResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in LLM response")
	}

	responseText := llmResp.Choices[0].Message.Content
	if responseText == "" {
		return nil, fmt.Errorf("empty response from LLM")
	}

	c.logger.Debug("received response from LLM API", "response_length", len(responseText))

	// parse the response to extract summary and tags
	result, err := c.parseProcessingResult(responseText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	return result, nil
}

// createArticleProcessingPrompt create a prompt for article processing
func (c *LLMClient) createArticleProcessingPrompt(title, content string) string {
	prompt := fmt.Sprintf(`Please provide a concise summary of the following article in 2-3 sentences. Focus on the main topics, key insights, and most important information. Use simple chinese to respond.

Article Title: %s

Article Content: %s

Please respond with only the summary text, no additional formatting or JSON structure needed.`, title, content)

	return prompt
}

// parseProcessingResult parse the LLM response to extract summary
func (c *LLMClient) parseProcessingResult(responseText string) (*ProcessingResult, error) {
	// clean up the response text
	summary := strings.TrimSpace(responseText)

	// ensure the summary is not empty
	if summary == "" {
		return nil, fmt.Errorf("received empty summary from LLM")
	}

	// limit summary length to prevent excessively long responses
	const maxSummaryLength = 1000
	if len(summary) > maxSummaryLength {
		// find the last sentence that fits within the limit
		truncated := summary[:maxSummaryLength]
		lastPeriod := strings.LastIndex(truncated, ".")
		if lastPeriod > 0 {
			summary = summary[:lastPeriod+1]
		} else {
			summary = truncated + "..."
		}
	}

	return &ProcessingResult{
		Summary: summary,
	}, nil
}

// GetModel returns the model name being used
func (c *LLMClient) GetModel() string {
	return c.model
}
