package client

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestLLMClient_ProcessArticle(t *testing.T) {
	tests := []struct {
		name           string
		title          string
		content        string
		responseStatus int
		responseBody   string
		expectError    bool
		expectedResult *ProcessingResult
	}{
		{
			name:           "successful processing with JSON response",
			title:          "Test Article",
			content:        "This is a test article about technology.",
			responseStatus: http.StatusOK,
			responseBody: `{
				"choices": [{
					"message": {
						"content": "Test article about technology"
					}
				}]
			}`,
			expectError: false,
			expectedResult: &ProcessingResult{
				Summary: "Test article about technology",
			},
		},
		{
			name:           "successful processing with direct text",
			title:          "Test Article",
			content:        "This is a test article about science.",
			responseStatus: http.StatusOK,
			responseBody: `{
				"choices": [{
					"message": {
						"content": "This is a test article about science and research. It contains important information."
					}
				}]
			}`,
			expectError: false,
			expectedResult: &ProcessingResult{
				Summary: "This is a test article about science and research. It contains important information.",
			},
		},
		{
			name:           "API error response",
			title:          "Test Article",
			content:        "Test content",
			responseStatus: http.StatusInternalServerError,
			responseBody:   `{"error": "Internal server error"}`,
			expectError:    true,
		},
		{
			name:           "empty response",
			title:          "Test Article",
			content:        "Test content",
			responseStatus: http.StatusOK,
			responseBody:   `{"choices": []}`,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and headers
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
				}

				authHeader := r.Header.Get("Authorization")
				if authHeader != "Bearer test-api-key" {
					t.Errorf("Expected Authorization header with API key, got %s", authHeader)
				}

				// Verify request body
				var req LLMRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("Failed to decode request body: %v", err)
				}

				if req.Model != "test-model" {
					t.Errorf("Expected model: test-model, got %s", req.Model)
				}

				if len(req.Messages) != 1 || req.Messages[0].Role != "user" {
					t.Errorf("Expected single user message, got %+v", req.Messages)
				}

				// Send response
				w.WriteHeader(tt.responseStatus)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create client
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
			client := NewLLMClient(server.URL, "test-api-key", "test-model", time.Second*5, logger)

			// Test
			ctx := context.Background()
			result, err := client.ProcessArticle(ctx, tt.title, tt.content)

			// Verify
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Expected result, but got nil")
				return
			}

			if result.Summary != tt.expectedResult.Summary {
				t.Errorf("Expected summary: %s, got: %s", tt.expectedResult.Summary, result.Summary)
			}
		})
	}
}

func TestLLMClient_GetModel(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewLLMClient("http://example.com", "test-key", "test-model", time.Second, logger)

	if client.GetModel() != "test-model" {
		t.Errorf("Expected model: test-model, got: %s", client.GetModel())
	}
}

func TestLLMClient_CreateArticleProcessingPrompt(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewLLMClient("http://example.com", "test-key", "test-model", time.Second, logger)

	title := "Test Title"
	content := "Test content"
	prompt := client.createArticleProcessingPrompt(title, content)

	if prompt == "" {
		t.Errorf("Expected non-empty prompt")
	}

	// Verify that title and content are included in the prompt
	if len(prompt) < len(title)+len(content) {
		t.Errorf("Prompt seems too short to contain title and content")
	}
}

func TestLLMClient_ParseProcessingResult(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	client := NewLLMClient("http://example.com", "test-key", "test-model", time.Second, logger)

	tests := []struct {
		name           string
		responseText   string
		expectedResult *ProcessingResult
		expectError    bool
	}{
		{
			name:         "simple text response",
			responseText: "This is a test summary about technology and innovation.",
			expectedResult: &ProcessingResult{
				Summary: "This is a test summary about technology and innovation.",
			},
			expectError: false,
		},
		{
			name:         "text with whitespace",
			responseText: "  \n  This is a clean summary.  \n  ",
			expectedResult: &ProcessingResult{
				Summary: "This is a clean summary.",
			},
			expectError: false,
		},
		{
			name:         "empty response",
			responseText: "",
			expectError:  true,
		},
		{
			name:         "only whitespace",
			responseText: "   \n\t   ",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.parseProcessingResult(tt.responseText)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Expected result, but got nil")
				return
			}

			if result.Summary != tt.expectedResult.Summary {
				t.Errorf("Expected summary: %s, got: %s", tt.expectedResult.Summary, result.Summary)
			}
		})
	}
}
