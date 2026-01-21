package loop

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
