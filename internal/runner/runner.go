package runner

import (
	"io"
	"os/exec"
)

// Result represents the outcome of a runner iteration
type Result struct {
	// Duration of the iteration in milliseconds
	DurationMs int
	// Number of turns/exchanges in the conversation
	NumTurns int
	// Total cost in USD (if available)
	TotalCostUSD float64
	// Token usage statistics
	Usage Usage
	// Whether the iteration ended in an error
	IsError bool
	// Optional result/summary text
	Result string
}

// Usage represents token usage statistics
type Usage struct {
	InputTokens              int
	OutputTokens             int
	CacheCreationInputTokens int
	CacheReadInputTokens     int
}

// Runner defines the interface for AI CLI backends
type Runner interface {
	// Name returns a human-readable name for this runner (e.g., "claude", "codex")
	Name() string

	// Command creates an exec.Cmd configured for this runner.
	// The prompt content will be written to the command's stdin.
	Command() *exec.Cmd

	// ParseOutput reads the runner's stdout stream, writes streaming output
	// to the display writer, logs raw output to the log writer, and returns
	// the final result.
	ParseOutput(stdout io.Reader, display io.Writer, log io.Writer) (*Result, error)
}
