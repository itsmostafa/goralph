package rlm

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

// Runner orchestrates an RLM session, managing the REPL loop and LLM API calls.
type Runner struct {
	config Config
	output io.Writer
}

// NewRunner creates a new RLM runner with the given configuration.
func NewRunner(config Config, output io.Writer) *Runner {
	return &Runner{
		config: config,
		output: output,
	}
}

// RunResult holds the result of an RLM session.
type RunResult struct {
	Answer          string
	Iterations      int
	TotalTokens     int
	InputTokens     int
	OutputTokens    int
	DurationMs      int
	IsError         bool
	SessionComplete bool
}

// Run executes an RLM session with the given context content and query.
func (r *Runner) Run(ctx context.Context, contextContent string, query string) (*RunResult, error) {
	startTime := time.Now()

	// Create environment
	env := NewEnvironment(contextContent, query)
	env.MaxDepth = r.config.MaxDepth
	env.CurrentDepth = 0

	// Set up recursive LLM function for sub-calls
	env.RecursiveLLM = func(subQuery, subContext string) (string, error) {
		return r.runRecursive(ctx, subQuery, subContext, env.CurrentDepth+1)
	}

	// Build system prompt
	systemPrompt := BuildSystemPrompt(len(contextContent), 0, query)

	// Initialize conversation
	messages := []Message{
		{Role: "user", Content: query},
	}

	// Create REPL executor
	executor := NewREPLExecutor(env, r.config)

	var totalInputTokens, totalOutputTokens int
	var finalAnswer string
	var sessionComplete bool

	for iteration := 1; iteration <= r.config.MaxIterations; iteration++ {
		// Call LLM
		response, inputTokens, outputTokens, err := r.callLLM(ctx, systemPrompt, messages)
		if err != nil {
			return nil, fmt.Errorf("LLM call failed at iteration %d: %w", iteration, err)
		}

		totalInputTokens += inputTokens
		totalOutputTokens += outputTokens

		// Stream the response to output
		r.streamOutput(fmt.Sprintf("\n[Iteration %d]\n", iteration))
		r.streamOutput(response + "\n")

		// Check for FINAL statement
		if answer, ok := ParseFinalStatement(response, env); ok {
			finalAnswer = answer
			sessionComplete = true
			r.streamOutput(fmt.Sprintf("\n[FINAL] %s\n", answer))
			break
		}

		// Extract and execute code
		code := ExtractCode(response)
		if code == response && !strings.Contains(response, "=") && !strings.Contains(response, "(") {
			// No code found, model might be explaining - add to messages and continue
			messages = append(messages,
				Message{Role: "assistant", Content: response},
				Message{Role: "user", Content: "Please write JavaScript code to explore the context and answer the query, or use FINAL(\"your answer\") if you have enough information."},
			)
			continue
		}

		// Execute the code in REPL
		r.streamOutput(fmt.Sprintf("\n[Executing]\n%s\n", code))
		result := executor.Execute(ctx, code)

		// Note: Variables are persisted in env.Variables by the executor
		// through the goja runtime's state

		// Format result for conversation
		resultStr := BuildREPLResultPrompt(result)
		r.streamOutput(fmt.Sprintf("\n[Result]\n%s\n", resultStr))

		// Check for FINAL_VAR after execution (variable might have been set)
		if answer, ok := ExtractFinalVar(response, env); ok {
			finalAnswer = answer
			sessionComplete = true
			r.streamOutput(fmt.Sprintf("\n[FINAL_VAR] %s\n", answer))
			break
		}

		// Add to conversation
		messages = append(messages,
			Message{Role: "assistant", Content: response},
			Message{Role: "user", Content: resultStr},
		)
	}

	return &RunResult{
		Answer:          finalAnswer,
		Iterations:      len(messages) / 2,
		TotalTokens:     totalInputTokens + totalOutputTokens,
		InputTokens:     totalInputTokens,
		OutputTokens:    totalOutputTokens,
		DurationMs:      int(time.Since(startTime).Milliseconds()),
		SessionComplete: sessionComplete,
	}, nil
}

// runRecursive runs a nested RLM call for sub-context analysis.
func (r *Runner) runRecursive(ctx context.Context, query, contextContent string, depth int) (string, error) {
	if depth >= r.config.MaxDepth {
		return "", fmt.Errorf("maximum recursion depth (%d) reached", r.config.MaxDepth)
	}

	// Create a sub-runner with reduced max iterations
	subConfig := r.config
	subConfig.MaxIterations = max(r.config.MaxIterations/2, 5)

	// Use recursive model if configured
	if r.config.RecursiveModel != "" {
		subConfig.Model = r.config.RecursiveModel
	}

	// Create a silent writer for recursive calls (or use a prefixed one)
	var buf bytes.Buffer
	subRunner := NewRunner(subConfig, &buf)

	result, err := subRunner.Run(ctx, contextContent, query)
	if err != nil {
		return "", err
	}

	return result.Answer, nil
}


// streamOutput writes to the output writer.
func (r *Runner) streamOutput(text string) {
	if r.output != nil {
		r.output.Write([]byte(text))
	}
}

// callLLM makes an API call to the configured LLM.
func (r *Runner) callLLM(ctx context.Context, systemPrompt string, messages []Message) (string, int, int, error) {
	apiKey := r.config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return "", 0, 0, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	apiBase := r.config.APIBase
	if apiBase == "" {
		apiBase = "https://api.anthropic.com"
	}

	// Build request
	reqBody := anthropicRequest{
		Model:     r.config.Model,
		MaxTokens: 4096,
		System:    systemPrompt,
		Messages:  make([]anthropicMessage, len(messages)),
	}

	for i, msg := range messages {
		reqBody.Messages[i] = anthropicMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request
	req, err := http.NewRequestWithContext(ctx, "POST", apiBase+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{
		Timeout: time.Duration(r.config.Timeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, 0, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, 0, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", 0, 0, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text from response
	var text strings.Builder
	for _, block := range apiResp.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}

	return text.String(), apiResp.Usage.InputTokens, apiResp.Usage.OutputTokens, nil
}

// Anthropic API types
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
	Usage   anthropicUsage          `json:"usage"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
