// Package rlm implements the Recursive Language Model pattern for goralph.
// RLM enables handling of large contexts by storing them as variables
// and allowing the model to explore them programmatically through a REPL.
package rlm

import (
	"regexp"
)

// Config holds configuration for an RLM instance.
type Config struct {
	// Model is the primary model name for the root call (e.g., "claude-sonnet-4-20250514")
	Model string

	// RecursiveModel is an optional cheaper model for recursive sub-calls.
	// If empty, uses Model for all calls.
	RecursiveModel string

	// APIBase is an optional base URL for the API (for custom endpoints)
	APIBase string

	// APIKey is the API key for authentication
	APIKey string

	// MaxDepth is the maximum recursion depth for nested RLM calls (default: 5)
	MaxDepth int

	// MaxIterations is the maximum REPL iterations per call (default: 30)
	MaxIterations int

	// Temperature for LLM calls (default: 0)
	Temperature float64

	// Timeout in seconds for each LLM call (default: 120)
	Timeout int

	// MaxOutputChars is the maximum characters to return from REPL execution (default: 2000)
	MaxOutputChars int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Model:          "claude-sonnet-4-20250514",
		MaxDepth:       5,
		MaxIterations:  30,
		Temperature:    0,
		Timeout:        120,
		MaxOutputChars: 2000,
	}
}

// Message represents a conversation message for LLM API calls.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Environment holds the REPL execution environment.
// Variables are stored here and accessible to executed code.
type Environment struct {
	// Context is the main document/codebase content to analyze
	Context string

	// Query is the user's question or task
	Query string

	// Variables holds user-defined variables created during REPL execution
	Variables map[string]any

	// RE provides access to regex functions
	RE *RegexModule

	// RecursiveLLM is a function to spawn sub-RLM calls
	RecursiveLLM func(subQuery, subContext string) (string, error)

	// CurrentDepth tracks the current recursion depth
	CurrentDepth int

	// MaxDepth is the maximum allowed recursion depth
	MaxDepth int
}

// NewEnvironment creates a new REPL environment with the given context and query.
func NewEnvironment(context, query string) *Environment {
	return &Environment{
		Context:   context,
		Query:     query,
		Variables: make(map[string]any),
		RE:        &RegexModule{},
	}
}

// RegexModule provides regex functions for the REPL environment.
type RegexModule struct{}

// FindAll finds all matches of pattern in text.
func (r *RegexModule) FindAll(pattern, text string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return re.FindAllString(text, -1), nil
}

// Search finds the first match of pattern in text.
func (r *RegexModule) Search(pattern, text string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	match := re.FindString(text)
	return match, nil
}

// Split splits text by pattern.
func (r *RegexModule) Split(pattern, text string, n int) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return re.Split(text, n), nil
}

// Replace replaces matches of pattern in text with repl.
func (r *RegexModule) Replace(pattern, text, repl string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	return re.ReplaceAllString(text, repl), nil
}

// CompletionResult holds the result of an RLM completion.
type CompletionResult struct {
	// Answer is the final answer extracted from the RLM session
	Answer string

	// Iterations is the number of REPL iterations used
	Iterations int

	// Depth is the maximum recursion depth reached
	Depth int

	// LLMCalls is the total number of LLM API calls made
	LLMCalls int

	// TotalTokens is the approximate total tokens used (if available)
	TotalTokens int
}

// REPLResult holds the result of a single REPL execution.
type REPLResult struct {
	// Output is the execution output (stdout + expression results)
	Output string

	// Error is any error that occurred during execution
	Error error

	// Truncated indicates if output was truncated due to length
	Truncated bool
}

// FinalStatement represents a parsed FINAL() or FINAL_VAR() statement.
type FinalStatement struct {
	// Type is either "FINAL" or "FINAL_VAR"
	Type string

	// Value is the direct value for FINAL("value")
	Value string

	// VarName is the variable name for FINAL_VAR(varname)
	VarName string
}
