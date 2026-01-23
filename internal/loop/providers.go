package loop

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/itsmostafa/goralph/internal/rlm"
)

// Provider defines the interface for agent providers
type Provider interface {
	// Name returns the provider name for display purposes
	Name() string
	// Model returns the model being used by this provider
	Model() string
	// BuildCommand creates the command to execute with the given prompt
	BuildCommand(prompt []byte) (*exec.Cmd, error)
	// ParseOutput parses the agent output and returns the result summary
	ParseOutput(r io.Reader, w io.Writer, logFile io.Writer) (*ResultMessage, error)
}

// DirectRunner is an optional interface for providers that run directly
// without spawning external CLI processes.
type DirectRunner interface {
	Provider
	// RunDirect executes the provider's logic directly with the given prompt.
	// Returns the result message and whether the session is complete.
	RunDirect(prompt []byte, output io.Writer, logFile io.Writer) (*ResultMessage, error)
}

// NewProvider creates a new Provider instance based on the agent type
func NewProvider(agent AgentProvider) (Provider, error) {
	switch agent {
	case AgentClaude:
		return &ClaudeProvider{}, nil
	case AgentCodex:
		return &CodexProvider{}, nil
	case AgentRLM:
		return NewRLMProvider(rlm.DefaultConfig()), nil
	default:
		return nil, fmt.Errorf("unknown agent provider: %s", agent)
	}
}

// ClaudeProvider implements Provider for Claude Code agent
type ClaudeProvider struct {
	prompt []byte
}

// Name returns the provider name
func (p *ClaudeProvider) Name() string {
	return "claude"
}

// Model returns the model being used
func (p *ClaudeProvider) Model() string {
	return "claude-sonnet-4-20250514"
}

// BuildCommand creates the claude command
func (p *ClaudeProvider) BuildCommand(prompt []byte) (*exec.Cmd, error) {
	p.prompt = prompt
	cmd := exec.Command("claude",
		"-p",
		"--dangerously-skip-permissions",
		"--output-format=stream-json",
		"--verbose",
	)
	return cmd, nil
}

// ParseOutput parses Claude's JSON stream output
func (p *ClaudeProvider) ParseOutput(r io.Reader, w io.Writer, logFile io.Writer) (*ResultMessage, error) {
	scanner := bufio.NewScanner(r)
	// Increase buffer size for large JSON lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var resultMsg *ResultMessage
	state := NewStreamState()

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Write raw JSON line to log file
		if logFile != nil {
			logFile.Write(line)
			logFile.Write([]byte("\n"))
		}

		// Parse the type field first
		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			// Not valid JSON, skip
			continue
		}

		switch msg.Type {
		case "result":
			// Parse full result message
			var result ResultMessage
			if err := json.Unmarshal(line, &result); err != nil {
				continue
			}
			result.HasCost = true // Claude provides cost data
			resultMsg = &result

		case "assistant":
			// Stream assistant content
			processClaudeAssistantMessage(line, w, state)

		case "user":
			// Check for tool results to mark tools as complete
			processClaudeUserMessage(line, w, state)

		case "system":
			// System messages (session info, etc.)
			// Could log these if verbose mode is desired
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Check for completion promise in accumulated text
	if resultMsg == nil {
		resultMsg = &ResultMessage{}
	}
	if strings.Contains(state.AccumulatedText.String(), CompletionPromise) {
		resultMsg.SessionComplete = true
	}

	return resultMsg, nil
}

// processClaudeAssistantMessage extracts and streams content from assistant messages
func processClaudeAssistantMessage(line []byte, w io.Writer, state *StreamState) {
	var assistantMsg AssistantMessage
	if err := json.Unmarshal(line, &assistantMsg); err != nil {
		return
	}

	// Build the full text from all text blocks
	var fullText strings.Builder
	for _, block := range assistantMsg.Message.Content {
		switch block.Type {
		case "text":
			fullText.WriteString(block.Text)
			state.AccumulatedText.WriteString(block.Text)
		case "tool_use":
			// Track and display tool invocations
			if block.ID != "" && state.ActiveTools[block.ID] == "" {
				state.ActiveTools[block.ID] = block.Name
				FormatToolStart(w, block.Name)
			}
		}
	}

	// Calculate and output the delta (new text since last message)
	currentText := fullText.String()
	if len(currentText) > state.LastTextLen {
		delta := currentText[state.LastTextLen:]
		FormatTextDelta(w, delta)
		state.LastTextLen = len(currentText)
	}
}

// processClaudeUserMessage checks for tool results and marks tools as complete
func processClaudeUserMessage(line []byte, w io.Writer, state *StreamState) {
	var userMsg UserMessage
	if err := json.Unmarshal(line, &userMsg); err != nil {
		return
	}

	for _, block := range userMsg.Message.Content {
		if block.Type == "tool_result" && block.ToolUseID != "" {
			toolName := state.ActiveTools[block.ToolUseID]
			if toolName != "" && !state.CompletedTools[block.ToolUseID] {
				state.CompletedTools[block.ToolUseID] = true
				FormatToolComplete(w, toolName)
			}
		}
	}
}

// CodexProvider implements Provider for OpenAI Codex agent
type CodexProvider struct {
	prompt []byte
}

// Name returns the provider name
func (p *CodexProvider) Name() string {
	return "codex"
}

// Model returns the model being used
func (p *CodexProvider) Model() string {
	return "codex-mini-latest"
}

// BuildCommand creates the codex command
func (p *CodexProvider) BuildCommand(prompt []byte) (*exec.Cmd, error) {
	p.prompt = prompt
	cmd := exec.Command("codex",
		"exec",
		"--json",
		"--dangerously-bypass-approvals-and-sandbox",
		"-",
	)
	return cmd, nil
}

// ParseOutput parses Codex's JSON stream output
func (p *CodexProvider) ParseOutput(r io.Reader, w io.Writer, logFile io.Writer) (*ResultMessage, error) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	state := NewStreamState()
	var turnCount int
	var totalUsage CodexUsage
	var hasError bool

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Write raw JSON line to log file
		if logFile != nil {
			logFile.Write(line)
			logFile.Write([]byte("\n"))
		}

		// Parse the event type
		var event CodexEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		switch event.Type {
		case "turn.completed":
			// Extract usage stats from turn.completed events
			var turnEvent CodexTurnCompletedEvent
			if err := json.Unmarshal(line, &turnEvent); err == nil {
				totalUsage.InputTokens += turnEvent.Usage.InputTokens
				totalUsage.CachedInputTokens += turnEvent.Usage.CachedInputTokens
				totalUsage.OutputTokens += turnEvent.Usage.OutputTokens
			}
		case "turn.failed":
			hasError = true
		case "error":
			hasError = true
			var errEvent CodexErrorEvent
			if err := json.Unmarshal(line, &errEvent); err == nil && errEvent.Message != "" {
				fmt.Fprintf(w, "\n%s\n", errorStyle.Render("Error: "+errEvent.Message))
			}
		case "item.started":
			processCodexItemStarted(line, w, state)
		case "item.completed":
			// Count reasoning items as "turns" since they represent model response cycles
			var itemEvent CodexItemEvent
			if err := json.Unmarshal(line, &itemEvent); err == nil {
				if itemEvent.Item.Type == "reasoning" {
					turnCount++
				}
			}
			processCodexItemCompleted(line, w, state)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Check for completion promise in accumulated text
	sessionComplete := strings.Contains(state.AccumulatedText.String(), CompletionPromise)

	// Build a result message for summary display
	result := &ResultMessage{
		Type:            "result",
		NumTurns:        turnCount,
		IsError:         hasError,
		SessionComplete: sessionComplete,
		Usage: Usage{
			InputTokens:          totalUsage.InputTokens,
			OutputTokens:         totalUsage.OutputTokens,
			CacheReadInputTokens: totalUsage.CachedInputTokens,
		},
	}

	return result, nil
}

// processCodexItemStarted handles item.started events from Codex output
func processCodexItemStarted(line []byte, w io.Writer, state *StreamState) {
	var itemEvent CodexItemEvent
	if err := json.Unmarshal(line, &itemEvent); err != nil {
		return
	}

	item := itemEvent.Item
	switch item.Type {
	case "command_execution":
		// Command execution starting - use command as the tool name
		toolName := "bash"
		if item.Command != "" {
			// Truncate long commands for display
			cmd := item.Command
			if len(cmd) > 50 {
				cmd = cmd[:47] + "..."
			}
			toolName = cmd
		}
		state.ActiveTools[item.ID] = toolName
		FormatToolStart(w, toolName)
	case "mcp_tool_call":
		// MCP tool invocation starting
		toolName := item.Name
		if toolName == "" {
			toolName = "mcp_tool"
		}
		state.ActiveTools[item.ID] = toolName
		FormatToolStart(w, toolName)
	case "file_change":
		state.ActiveTools[item.ID] = "file_change"
		FormatToolStart(w, "file_change")
	case "web_search":
		state.ActiveTools[item.ID] = "web_search"
		FormatToolStart(w, "web_search")
	}
}

// processCodexItemCompleted handles item.completed events from Codex output
func processCodexItemCompleted(line []byte, w io.Writer, state *StreamState) {
	var itemEvent CodexItemEvent
	if err := json.Unmarshal(line, &itemEvent); err != nil {
		return
	}

	item := itemEvent.Item
	switch item.Type {
	case "agent_message":
		// Text output from the agent
		if item.Text != "" {
			FormatTextDelta(w, item.Text+"\n")
			state.AccumulatedText.WriteString(item.Text)
		}
	case "reasoning":
		// Reasoning text - display as text
		if item.Text != "" {
			FormatTextDelta(w, item.Text+"\n")
			state.AccumulatedText.WriteString(item.Text)
		}
	case "command_execution", "mcp_tool_call", "file_change", "web_search":
		// Mark tool as complete
		toolName := state.ActiveTools[item.ID]
		if toolName != "" && !state.CompletedTools[item.ID] {
			state.CompletedTools[item.ID] = true
			FormatToolComplete(w, toolName)
		}
	case "plan_update":
		// Plan updates can be displayed as text if desired
		if item.Text != "" {
			FormatTextDelta(w, item.Text+"\n")
			state.AccumulatedText.WriteString(item.Text)
		}
	}
}

// RLMProvider implements Provider and DirectRunner for Recursive Language Model agent.
// RLM enables handling of large contexts by storing them as variables
// and allowing the model to explore them programmatically through a REPL.
type RLMProvider struct {
	prompt []byte
	config rlm.Config
}

// NewRLMProvider creates a new RLMProvider with the given configuration.
func NewRLMProvider(config rlm.Config) *RLMProvider {
	return &RLMProvider{config: config}
}

// Name returns the provider name
func (p *RLMProvider) Name() string {
	return "rlm"
}

// Model returns the model being used
func (p *RLMProvider) Model() string {
	return fmt.Sprintf("rlm (%s)", p.config.Model)
}

// BuildCommand is not used by RLM - it runs directly.
// Returns an error to indicate direct execution is required.
func (p *RLMProvider) BuildCommand(prompt []byte) (*exec.Cmd, error) {
	p.prompt = prompt
	return nil, fmt.Errorf("RLM provider uses direct execution - use RunDirect instead")
}

// ParseOutput is not used by RLM - it handles output internally.
func (p *RLMProvider) ParseOutput(r io.Reader, w io.Writer, logFile io.Writer) (*ResultMessage, error) {
	return nil, fmt.Errorf("RLM provider handles output internally via RunDirect")
}

// RunDirect executes the RLM session directly with the given prompt.
// This implements the DirectRunner interface.
func (p *RLMProvider) RunDirect(prompt []byte, output io.Writer, logFile io.Writer) (*ResultMessage, error) {
	// Parse the prompt to extract query and context
	// The prompt is structured with the query embedded in it
	promptStr := string(prompt)

	// Create RLM runner
	runner := rlm.NewRunner(p.config, output)

	// Run the RLM session
	// The prompt content serves as both context and query for RLM
	ctx := context.Background()
	result, err := runner.Run(ctx, promptStr, "Complete the task described in the context")
	if err != nil {
		return &ResultMessage{
			Type:    "result",
			IsError: true,
			Result:  err.Error(),
		}, err
	}

	// Log the result if logFile is provided
	if logFile != nil {
		logEntry := map[string]any{
			"type":        "rlm_result",
			"answer":      result.Answer,
			"iterations":  result.Iterations,
			"tokens":      result.TotalTokens,
			"duration_ms": result.DurationMs,
		}
		if logData, err := json.Marshal(logEntry); err == nil {
			logFile.Write(logData)
			logFile.Write([]byte("\n"))
		}
	}

	// Build result message
	return &ResultMessage{
		Type:       "result",
		DurationMs: result.DurationMs,
		NumTurns:   result.Iterations,
		Result:     result.Answer,
		Usage: Usage{
			InputTokens:  result.InputTokens,
			OutputTokens: result.OutputTokens,
		},
		SessionComplete: result.SessionComplete,
	}, nil
}
