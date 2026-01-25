package loop

import (
	"fmt"
	"sync"
	"time"
)

// LMBridge routes REPL LLM calls to providers
type LMBridge interface {
	// HandleRequest makes a recursive LLM call
	HandleRequest(prompt string) (string, error)

	// HandleBatchRequest makes multiple LLM calls
	HandleBatchRequest(prompts []string) ([]string, error)

	// GetCumulativeUsage returns total token usage across all calls
	GetCumulativeUsage() RLMUsage

	// GetCallHistory returns all LLM calls made through this bridge
	GetCallHistory() []LLMCallRecord
}

// ProviderBridge is a thread-safe LMBridge implementation using the Provider interface
type ProviderBridge struct {
	provider     Provider
	maxDepth     int
	currentDepth int
	callHistory  []LLMCallRecord
	totalUsage   RLMUsage
	mu           sync.Mutex
}

// NewProviderBridge creates a new ProviderBridge with the given provider and max depth
func NewProviderBridge(provider Provider, maxDepth int) *ProviderBridge {
	return &ProviderBridge{
		provider:    provider,
		maxDepth:    maxDepth,
		callHistory: make([]LLMCallRecord, 0),
	}
}

// HandleRequest makes a recursive LLM call with depth limiting
func (b *ProviderBridge) HandleRequest(prompt string) (string, error) {
	b.mu.Lock()

	// Check depth limit
	if b.currentDepth >= b.maxDepth {
		b.mu.Unlock()
		return "", fmt.Errorf("max recursion depth (%d) exceeded", b.maxDepth)
	}

	// Increment depth (protected by mutex)
	b.currentDepth++
	depth := b.currentDepth
	b.mu.Unlock()

	// Make the actual LLM call (outside mutex - can take a while)
	startTime := time.Now()
	response, usage, err := b.provider.Query(prompt)
	durationMs := int(time.Since(startTime).Milliseconds())

	// Update state (reacquire mutex)
	b.mu.Lock()
	defer b.mu.Unlock()

	b.currentDepth--

	if err != nil {
		return "", err
	}

	// Record in history
	record := LLMCallRecord{
		Prompt:     prompt,
		Response:   response,
		Model:      b.provider.Model(),
		Depth:      depth,
		TokensIn:   usage.InputTokens,
		TokensOut:  usage.OutputTokens,
		DurationMs: durationMs,
	}
	b.callHistory = append(b.callHistory, record)

	// Accumulate usage
	b.totalUsage.InputTokens += usage.InputTokens
	b.totalUsage.OutputTokens += usage.OutputTokens

	return response, nil
}

// HandleBatchRequest makes multiple LLM calls sequentially
func (b *ProviderBridge) HandleBatchRequest(prompts []string) ([]string, error) {
	results := make([]string, len(prompts))
	for i, prompt := range prompts {
		resp, err := b.HandleRequest(prompt)
		if err != nil {
			return nil, fmt.Errorf("batch request %d failed: %w", i, err)
		}
		results[i] = resp
	}
	return results, nil
}

// GetCumulativeUsage returns total token usage
func (b *ProviderBridge) GetCumulativeUsage() RLMUsage {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.totalUsage
}

// GetCallHistory returns a copy of all LLM calls
func (b *ProviderBridge) GetCallHistory() []LLMCallRecord {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Return a copy to avoid data races
	history := make([]LLMCallRecord, len(b.callHistory))
	copy(history, b.callHistory)
	return history
}
