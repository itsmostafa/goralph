package loop

import (
	"fmt"
	"io"
)

// Mode represents the execution mode
type Mode string

const (
	ModeBuild Mode = "build"
	ModePlan  Mode = "plan"

	// ImplementationPlanFile is the path to the implementation plan
	ImplementationPlanFile = ".ralph/IMPLEMENTATION_PLAN.md"
)

// Config holds the loop configuration
type Config struct {
	Mode          Mode
	PromptFile    string
	MaxIterations int
	NoPush        bool
	Agent         AgentProvider
	Output        io.Writer
}

// AgentProvider represents the agent provider to use
type AgentProvider string

const (
	AgentClaude AgentProvider = "claude"
	AgentCodex  AgentProvider = "codex"
)

// ValidateAgentProvider checks if the given agent provider is valid
func ValidateAgentProvider(agent string) (AgentProvider, error) {
	switch AgentProvider(agent) {
	case AgentClaude:
		return AgentClaude, nil
	case AgentCodex:
		return AgentCodex, nil
	default:
		return "", fmt.Errorf("unknown agent provider: %q (valid options: claude, codex)", agent)
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

// CodexEvent represents a generic event from Codex CLI JSON output
type CodexEvent struct {
	Type string `json:"type"`
}

// CodexThreadStartedEvent represents a thread.started event from Codex CLI
type CodexThreadStartedEvent struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id"`
}

// CodexTurnCompletedEvent represents a turn.completed event from Codex CLI
type CodexTurnCompletedEvent struct {
	Type  string     `json:"type"`
	Usage CodexUsage `json:"usage,omitempty"`
}

// CodexUsage represents token usage statistics from Codex CLI
type CodexUsage struct {
	InputTokens       int `json:"input_tokens"`
	CachedInputTokens int `json:"cached_input_tokens"`
	OutputTokens      int `json:"output_tokens"`
}

// CodexErrorEvent represents an error event from Codex CLI
type CodexErrorEvent struct {
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// CodexItemEvent represents an item.started or item.completed event from Codex CLI
type CodexItemEvent struct {
	Type string    `json:"type"`
	Item CodexItem `json:"item"`
}

// CodexItem represents an item object within item.* events
type CodexItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"` // agent_message, reasoning, command_execution, file_change, mcp_tool_call, web_search, plan_update
	Status  string `json:"status,omitempty"`
	Text    string `json:"text,omitempty"`
	Command string `json:"command,omitempty"`
	Name    string `json:"name,omitempty"` // For MCP tool calls
}
