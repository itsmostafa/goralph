package loop

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
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
	// Query makes a simple synchronous LLM query for recursive calls
	Query(prompt string) (response string, usage Usage, err error)
}

// NewProvider creates a new Provider instance based on the agent type
func NewProvider(agent AgentProvider) (Provider, error) {
	switch agent {
	case AgentClaude:
		return &ClaudeProvider{}, nil
	case AgentCodex:
		return &CodexProvider{}, nil
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

	// Check for completion promise and RLM markers in accumulated text
	if resultMsg == nil {
		resultMsg = &ResultMessage{}
	}
	accText := state.AccumulatedText.String()
	if strings.Contains(accText, CompletionPromise) {
		resultMsg.SessionComplete = true
	}
	// Detect RLM markers
	detectRLMMarkers(accText, resultMsg)

	return resultMsg, nil
}

// Query makes a simple synchronous LLM query for recursive calls
func (p *ClaudeProvider) Query(prompt string) (string, Usage, error) {
	// Create command with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude",
		"-p",
		"--output-format=json",
		"--dangerously-skip-permissions",
	)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Write prompt to stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", Usage{}, err
	}

	if err := cmd.Start(); err != nil {
		return "", Usage{}, err
	}

	// Write prompt and close stdin to signal EOF
	_, _ = stdin.Write([]byte(prompt))
	stdin.Close()

	// Wait for completion
	err = cmd.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		return "", Usage{}, fmt.Errorf("query timed out after 120s")
	}
	if err != nil {
		return "", Usage{}, fmt.Errorf("claude failed: %s", stderr.String())
	}

	// Parse JSON response
	var result struct {
		Result string `json:"result"`
		Usage  Usage  `json:"usage"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		// If JSON parsing fails, return raw output as result
		return stdout.String(), Usage{}, nil
	}

	return result.Result, result.Usage, nil
}

// processClaudeAssistantMessage extracts and streams content from assistant messages
func processClaudeAssistantMessage(line []byte, w io.Writer, state *StreamState) {
	var assistantMsg AssistantMessage
	if err := json.Unmarshal(line, &assistantMsg); err != nil {
		return
	}

	// First pass: accumulate all text
	var fullText strings.Builder
	for _, block := range assistantMsg.Message.Content {
		if block.Type == "text" {
			fullText.WriteString(block.Text)
			state.AccumulatedText.WriteString(block.Text)
		}
	}

	// Output text delta BEFORE processing tools
	currentText := fullText.String()
	if len(currentText) > state.LastTextLen {
		delta := currentText[state.LastTextLen:]
		FormatTextDelta(w, delta)
		state.LastTextLen = len(currentText)
		// Mark that we need a newline before tool output if text doesn't end with one
		if len(delta) > 0 && delta[len(delta)-1] != '\n' {
			state.NeedsNewline = true
		}
	}

	// Second pass: process tool_use blocks
	for _, block := range assistantMsg.Message.Content {
		if block.Type == "tool_use" {
			if block.ID != "" && state.ActiveTools[block.ID] == "" {
				state.ActiveTools[block.ID] = block.Name
				// Add newline before tool if needed (text didn't end with one)
				if state.NeedsNewline {
					fmt.Fprintln(w)
					state.NeedsNewline = false
				}
				FormatToolStart(w, block.ID, block.Name, state)
			}
		}
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
				FormatToolComplete(w, block.ToolUseID, toolName, state)
				state.NeedsNewline = false // Tool complete ends with newline
			}
		}
	}

	// Reset text tracking for the next assistant message turn
	// This ensures new assistant text after tools is fully displayed
	state.LastTextLen = 0
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

	// Check for completion promise and RLM markers in accumulated text
	accText := state.AccumulatedText.String()
	sessionComplete := strings.Contains(accText, CompletionPromise)

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

	// Detect RLM markers
	detectRLMMarkers(accText, result)

	return result, nil
}

// Query makes a simple synchronous LLM query for recursive calls
func (p *CodexProvider) Query(prompt string) (string, Usage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "codex",
		"exec",
		"--json",
		"--dangerously-bypass-approvals-and-sandbox",
		"-",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", Usage{}, err
	}

	if err := cmd.Start(); err != nil {
		return "", Usage{}, err
	}

	_, _ = stdin.Write([]byte(prompt))
	stdin.Close()

	err = cmd.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		return "", Usage{}, fmt.Errorf("query timed out after 120s")
	}
	if err != nil {
		return "", Usage{}, fmt.Errorf("codex failed: %s", stderr.String())
	}

	// Parse JSONL output, extract final agent_message
	var lastText string
	var totalUsage Usage
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		var event CodexItemEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		if event.Item.Type == "agent_message" && event.Item.Text != "" {
			lastText = event.Item.Text
		}
	}

	return lastText, totalUsage, nil
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
		FormatToolStart(w, item.ID, toolName, state)
	case "mcp_tool_call":
		// MCP tool invocation starting
		toolName := item.Name
		if toolName == "" {
			toolName = "mcp_tool"
		}
		state.ActiveTools[item.ID] = toolName
		FormatToolStart(w, item.ID, toolName, state)
	case "file_change":
		state.ActiveTools[item.ID] = "file_change"
		FormatToolStart(w, item.ID, "file_change", state)
	case "web_search":
		state.ActiveTools[item.ID] = "web_search"
		FormatToolStart(w, item.ID, "web_search", state)
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
			FormatToolComplete(w, item.ID, toolName, state)
		}
	case "plan_update":
		// Plan updates can be displayed as text if desired
		if item.Text != "" {
			FormatTextDelta(w, item.Text+"\n")
			state.AccumulatedText.WriteString(item.Text)
		}
	}
}

// detectRLMMarkers detects RLM markers in text and updates the result message
func detectRLMMarkers(text string, result *ResultMessage) {
	if result == nil {
		return
	}

	// Detect phase marker: <rlm:phase>PHASE</rlm:phase>
	if startIdx := strings.Index(text, RLMPhaseMarkerStart); startIdx != -1 {
		afterStart := startIdx + len(RLMPhaseMarkerStart)
		if endIdx := strings.Index(text[afterStart:], RLMPhaseMarkerEnd); endIdx != -1 {
			phaseStr := text[afterStart : afterStart+endIdx]
			result.ModePhase = strings.TrimSpace(phaseStr)
		}
	}

	// Detect verified marker: <rlm:verified>true</rlm:verified>
	if strings.Contains(text, RLMVerifiedMarkerStart+"true"+RLMVerifiedMarkerEnd) {
		result.ModeVerified = true
	}
}
