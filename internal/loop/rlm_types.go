package loop

import "time"

// REPLResult represents the result from executing code in the REPL
type REPLResult struct {
	Output     string          `json:"output"`
	Variables  map[string]any  `json:"variables"`
	ExecTimeMs int             `json:"exec_time_ms"`
	LLMCalls   []LLMCallRecord `json:"llm_calls"`
	Error      string          `json:"error,omitempty"`
}

// LLMCallRecord tracks a recursive LLM call from the REPL
type LLMCallRecord struct {
	Prompt     string `json:"prompt"`
	Response   string `json:"response"`
	Model      string `json:"model,omitempty"`
	Depth      int    `json:"depth"`
	TokensIn   int    `json:"tokens_in"`
	TokensOut  int    `json:"tokens_out"`
	DurationMs int    `json:"duration_ms"`
}

// RLMSession tracks the overall RLM session state
type RLMSession struct {
	SessionID     string    `json:"session_id"`
	Iteration     int       `json:"iteration"`
	MaxDepth      int       `json:"max_depth"`
	CurrentDepth  int       `json:"current_depth"`
	TotalLLMCalls int       `json:"total_llm_calls"`
	TotalCostUSD  float64   `json:"total_cost_usd"`
	FinalAnswer   string    `json:"final_answer,omitempty"`
	IsComplete    bool      `json:"is_complete"`
	StartedAt     time.Time `json:"started_at"`
}

// QueryMetadata provides context information for the system prompt
type QueryMetadata struct {
	ContextType string `json:"context_type"`
	TotalLength int    `json:"total_length"`
}

// RLMUsage tracks token usage for RLM operations
type RLMUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
