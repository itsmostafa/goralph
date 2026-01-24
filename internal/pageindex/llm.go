package pageindex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// LLMProvider defines the interface for LLM interactions.
type LLMProvider interface {
	// Complete sends a prompt and returns the response text.
	Complete(ctx context.Context, prompt string) (string, error)

	// CompleteWithHistory sends a prompt with conversation history.
	CompleteWithHistory(ctx context.Context, messages []Message) (string, error)

	// Model returns the model identifier being used.
	Model() string
}

// Message represents a chat message for conversation history.
type Message struct {
	Role    string `json:"role"`    // "user", "assistant", or "system"
	Content string `json:"content"` // Message content
}

// OpenAIProvider implements LLMProvider using OpenAI's API.
type OpenAIProvider struct {
	apiKey     string
	model      string
	httpClient *http.Client
	maxRetries int
}

// OpenAIRequest represents the request body for OpenAI's chat completions API.
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
}

// OpenAIMessage represents a message in the OpenAI format.
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents the response from OpenAI's chat completions API.
type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      OpenAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *OpenAIError `json:"error,omitempty"`
}

// OpenAIError represents an error response from OpenAI.
type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(model string) (*OpenAIProvider, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("CHATGPT_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY or CHATGPT_API_KEY environment variable not set")
	}
	return &OpenAIProvider{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 120 * time.Second},
		maxRetries: 10,
	}, nil
}

// Model returns the model identifier.
func (p *OpenAIProvider) Model() string {
	return p.model
}

// Complete sends a prompt and returns the response.
func (p *OpenAIProvider) Complete(ctx context.Context, prompt string) (string, error) {
	return p.CompleteWithHistory(ctx, []Message{{Role: "user", Content: prompt}})
}

// CompleteWithHistory sends a prompt with conversation history.
func (p *OpenAIProvider) CompleteWithHistory(ctx context.Context, messages []Message) (string, error) {
	// Convert to OpenAI format
	oaiMessages := make([]OpenAIMessage, len(messages))
	for i, m := range messages {
		oaiMessages[i] = OpenAIMessage{Role: m.Role, Content: m.Content}
	}

	req := OpenAIRequest{
		Model:       p.model,
		Messages:    oaiMessages,
		Temperature: 0,
	}

	var lastErr error
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		body, err := json.Marshal(req)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			lastErr = err
			time.Sleep(time.Second)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			time.Sleep(time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
			time.Sleep(time.Second)
			continue
		}

		var oaiResp OpenAIResponse
		if err := json.Unmarshal(respBody, &oaiResp); err != nil {
			lastErr = fmt.Errorf("failed to parse response: %w", err)
			time.Sleep(time.Second)
			continue
		}

		if oaiResp.Error != nil {
			lastErr = fmt.Errorf("API error: %s", oaiResp.Error.Message)
			time.Sleep(time.Second)
			continue
		}

		if len(oaiResp.Choices) == 0 {
			lastErr = fmt.Errorf("no choices in response")
			time.Sleep(time.Second)
			continue
		}

		return oaiResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("max retries exceeded: %w", lastErr)
}

// CompleteWithFinishReason sends a prompt and returns both response and finish reason.
func (p *OpenAIProvider) CompleteWithFinishReason(ctx context.Context, prompt string, history []Message) (string, string, error) {
	messages := append(history, Message{Role: "user", Content: prompt})
	oaiMessages := make([]OpenAIMessage, len(messages))
	for i, m := range messages {
		oaiMessages[i] = OpenAIMessage{Role: m.Role, Content: m.Content}
	}

	req := OpenAIRequest{
		Model:       p.model,
		Messages:    oaiMessages,
		Temperature: 0,
	}

	var lastErr error
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		body, err := json.Marshal(req)
		if err != nil {
			return "", "", fmt.Errorf("failed to marshal request: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
		if err != nil {
			return "", "", fmt.Errorf("failed to create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

		resp, err := p.httpClient.Do(httpReq)
		if err != nil {
			lastErr = err
			time.Sleep(time.Second)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			time.Sleep(time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
			time.Sleep(time.Second)
			continue
		}

		var oaiResp OpenAIResponse
		if err := json.Unmarshal(respBody, &oaiResp); err != nil {
			lastErr = fmt.Errorf("failed to parse response: %w", err)
			time.Sleep(time.Second)
			continue
		}

		if oaiResp.Error != nil {
			lastErr = fmt.Errorf("API error: %s", oaiResp.Error.Message)
			time.Sleep(time.Second)
			continue
		}

		if len(oaiResp.Choices) == 0 {
			lastErr = fmt.Errorf("no choices in response")
			time.Sleep(time.Second)
			continue
		}

		finishReason := oaiResp.Choices[0].FinishReason
		if finishReason == "length" {
			finishReason = "max_output_reached"
		} else {
			finishReason = "finished"
		}

		return oaiResp.Choices[0].Message.Content, finishReason, nil
	}

	return "", "", fmt.Errorf("max retries exceeded: %w", lastErr)
}

// ExtractJSON extracts and parses JSON from an LLM response.
// It handles responses wrapped in ```json ... ``` blocks.
func ExtractJSON[T any](content string) (T, error) {
	var result T

	// Try to extract from code block
	content = strings.TrimSpace(content)
	if startIdx := strings.Index(content, "```json"); startIdx != -1 {
		startIdx += 7
		if endIdx := strings.LastIndex(content, "```"); endIdx > startIdx {
			content = content[startIdx:endIdx]
		}
	} else if startIdx := strings.Index(content, "```"); startIdx != -1 {
		startIdx += 3
		if endIdx := strings.LastIndex(content[startIdx:], "```"); endIdx != -1 {
			content = content[startIdx : startIdx+endIdx]
		}
	}

	// Clean up common issues
	content = strings.TrimSpace(content)
	content = strings.ReplaceAll(content, "None", "null")

	// First attempt
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// Try fixing trailing commas
		content = strings.ReplaceAll(content, ",]", "]")
		content = strings.ReplaceAll(content, ",}", "}")
		if err := json.Unmarshal([]byte(content), &result); err != nil {
			return result, fmt.Errorf("failed to parse JSON: %w (content: %s)", err, truncate(content, 200))
		}
	}

	return result, nil
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
