package loop

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Provider defines the interface for CLI providers
type Provider interface {
	// Name returns the provider name for display purposes
	Name() string
	// BuildCommand creates the command to execute with the given prompt
	BuildCommand(prompt []byte) (*exec.Cmd, error)
	// ParseOutput parses the CLI output and returns the result summary
	ParseOutput(r io.Reader, w io.Writer, logFile io.Writer) (*ResultMessage, error)
}

// NewProvider creates a new Provider instance based on the CLI type
func NewProvider(cli CLIProvider) (Provider, error) {
	switch cli {
	case CLIClaude:
		return &ClaudeProvider{}, nil
	case CLICodex:
		return &CodexProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown CLI provider: %s", cli)
	}
}

// ClaudeProvider implements Provider for Claude Code CLI
type ClaudeProvider struct {
	prompt []byte
}

// Name returns the provider name
func (p *ClaudeProvider) Name() string {
	return "claude"
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

// CodexProvider implements Provider for OpenAI Codex CLI
type CodexProvider struct {
	prompt []byte
}

// Name returns the provider name
func (p *CodexProvider) Name() string {
	return "codex"
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
// This is a placeholder - Task 3 will implement full Codex parsing
func (p *CodexProvider) ParseOutput(r io.Reader, w io.Writer, logFile io.Writer) (*ResultMessage, error) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	state := NewStreamState()
	var turnCount int

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
		case "turn.started":
			turnCount++
		case "turn.completed", "turn.failed":
			// Turn finished
		case "error":
			// Handle error events
			var errEvent CodexErrorEvent
			if err := json.Unmarshal(line, &errEvent); err == nil && errEvent.Message != "" {
				fmt.Fprintf(w, "\n%s\n", errorStyle.Render("Error: "+errEvent.Message))
			}
		default:
			// Handle item.* events
			if strings.HasPrefix(event.Type, "item.") {
				processCodexItemEvent(line, w, state)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Build a result message for summary display
	// Codex doesn't provide the same stats, so we provide what we can
	result := &ResultMessage{
		Type:     "result",
		NumTurns: turnCount,
		// Other fields will be zero/empty - Task 3 will enhance this
	}

	return result, nil
}

// processCodexItemEvent handles item.* events from Codex output
func processCodexItemEvent(line []byte, w io.Writer, state *StreamState) {
	var itemEvent CodexItemEvent
	if err := json.Unmarshal(line, &itemEvent); err != nil {
		return
	}

	switch itemEvent.Type {
	case "item.message":
		// Text output from the agent
		if text, ok := itemEvent.Content.(string); ok {
			FormatTextDelta(w, text)
		}
	case "item.reasoning":
		// Reasoning text (could display differently if desired)
		if text, ok := itemEvent.Content.(string); ok {
			FormatTextDelta(w, text)
		}
	case "item.command_start":
		// Command execution starting
		if name, ok := itemEvent.Content.(string); ok {
			FormatToolStart(w, name)
		}
	case "item.command_end":
		// Command execution complete
		if name, ok := itemEvent.Content.(string); ok {
			FormatToolComplete(w, name)
		}
	case "item.mcp_tool_start":
		// MCP tool invocation starting
		if name, ok := itemEvent.Content.(string); ok {
			FormatToolStart(w, name)
		}
	case "item.mcp_tool_end":
		// MCP tool invocation complete
		if name, ok := itemEvent.Content.(string); ok {
			FormatToolComplete(w, name)
		}
	}
}
