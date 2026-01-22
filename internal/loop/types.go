package loop

import "fmt"

// CLIProvider represents the CLI provider to use
type CLIProvider string

const (
	CLIClaude CLIProvider = "claude"
	CLICodex  CLIProvider = "codex"
)

// ValidateCLIProvider checks if the given CLI provider is valid
func ValidateCLIProvider(cli string) (CLIProvider, error) {
	switch CLIProvider(cli) {
	case CLIClaude:
		return CLIClaude, nil
	case CLICodex:
		return CLICodex, nil
	default:
		return "", fmt.Errorf("unknown CLI provider: %q (valid options: claude, codex)", cli)
	}
}

// Message represents a generic JSON message from Claude stream output
type Message struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype,omitempty"`
}

// SystemMessage represents a system message from Claude
type SystemMessage struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype"`
	Model     string `json:"model"`
	SessionID string `json:"session_id"`
}

// ResultMessage represents the final result message from Claude
type ResultMessage struct {
	Type         string  `json:"type"`
	Subtype      string  `json:"subtype"`
	IsError      bool    `json:"is_error"`
	DurationMs   int     `json:"duration_ms"`
	NumTurns     int     `json:"num_turns"`
	Result       string  `json:"result"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	Usage        Usage   `json:"usage"`
}

// Usage represents token usage statistics
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// AssistantMessage represents an assistant message from Claude stream output
type AssistantMessage struct {
	Type    string           `json:"type"`
	Message AssistantContent `json:"message"`
}

// AssistantContent represents the content within an assistant message
type AssistantContent struct {
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents a single content block (text or tool_use)
type ContentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`
}

// UserMessage represents a user message (often contains tool results)
type UserMessage struct {
	Type    string      `json:"type"`
	Message UserContent `json:"message"`
}

// UserContent represents the content within a user message
type UserContent struct {
	Content []ToolResultBlock `json:"content"`
}

// ToolResultBlock represents a tool result in a user message
type ToolResultBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id,omitempty"`
}

// StreamState tracks the state during streaming output
type StreamState struct {
	LastTextLen    int
	ActiveTools    map[string]string // tool ID -> tool name
	CompletedTools map[string]bool
}

// NewStreamState creates a new StreamState with initialized maps
func NewStreamState() *StreamState {
	return &StreamState{
		ActiveTools:    make(map[string]string),
		CompletedTools: make(map[string]bool),
	}
}
