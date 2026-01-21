package loop

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Mode represents the execution mode
type Mode string

const (
	ModeBuild Mode = "build"
	ModePlan  Mode = "plan"
)

// Config holds the loop configuration
type Config struct {
	Mode          Mode
	PromptFile    string
	MaxIterations int
	Output        io.Writer
}

// Run executes the agentic loop
func Run(cfg Config) error {
	// Default output to stdout
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	// Get current git branch
	branch, err := getCurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Verify prompt file exists
	if _, err := os.Stat(cfg.PromptFile); os.IsNotExist(err) {
		return fmt.Errorf("prompt file not found: %s", cfg.PromptFile)
	}

	// Print configuration
	FormatHeader(cfg.Output, cfg, branch)

	iteration := 0
	for {
		iteration++

		// Check max iterations
		if cfg.MaxIterations > 0 && iteration > cfg.MaxIterations {
			FormatMaxIterations(cfg.Output, cfg.MaxIterations)
			break
		}

		// Show loop banner before iteration
		FormatLoopBanner(cfg.Output, iteration)

		// Run Claude iteration
		if err := runClaudeIteration(cfg.PromptFile, cfg.Output); err != nil {
			return fmt.Errorf("claude iteration failed: %w", err)
		}

		// Push changes
		if err := pushChanges(branch); err != nil {
			return fmt.Errorf("failed to push changes: %w", err)
		}
	}

	return nil
}

func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func runClaudeIteration(promptFile string, w io.Writer) error {
	// Read prompt file
	promptContent, err := os.ReadFile(promptFile)
	if err != nil {
		return fmt.Errorf("failed to read prompt file: %w", err)
	}

	// Create logs directory
	logsDir := filepath.Join(".ralph", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Generate timestamped log filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logsDir, timestamp+".jsonl")

	// Create log file
	logFile, err := os.Create(logPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	// Run claude with the prompt piped to stdin
	cmd := exec.Command("claude",
		"-p",
		"--dangerously-skip-permissions",
		"--output-format=stream-json",
		"--verbose",
	)

	// Set up stdin with prompt content
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Capture stdout for parsing
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Connect stderr to terminal
	cmd.Stderr = os.Stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start claude: %w", err)
	}

	// Write prompt to stdin and close
	if _, err := stdin.Write(promptContent); err != nil {
		return fmt.Errorf("failed to write to stdin: %w", err)
	}
	stdin.Close()

	// Show progress indicator
	fmt.Fprintln(w, dimStyle.Render("Running Claude..."))

	// Parse JSON output and write to log file
	if err := parseClaudeOutput(stdout, w, logFile); err != nil {
		return fmt.Errorf("failed to parse output: %w", err)
	}

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("claude exited with error: %w", err)
	}

	return nil
}

func parseClaudeOutput(r io.Reader, w io.Writer, logFile io.Writer) error {
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
			processAssistantMessage(line, w, state)

		case "user":
			// Check for tool results to mark tools as complete
			processUserMessage(line, w, state)

		case "system":
			// System messages (session info, etc.)
			// Could log these if verbose mode is desired
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Display the final result summary
	fmt.Fprintln(w)
	if resultMsg != nil {
		FormatIterationSummary(w, *resultMsg)
	} else {
		fmt.Fprintln(w, dimStyle.Render("Warning: No result message received from Claude"))
	}

	return nil
}

// processAssistantMessage extracts and streams content from assistant messages
func processAssistantMessage(line []byte, w io.Writer, state *StreamState) {
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

// processUserMessage checks for tool results and marks tools as complete
func processUserMessage(line []byte, w io.Writer, state *StreamState) {
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

func pushChanges(branch string) error {
	// Try to push
	cmd := exec.Command("git", "push", "origin", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// If push failed, try to create remote branch
		fmt.Println("Failed to push. Creating remote branch...")
		cmd = exec.Command("git", "push", "-u", "origin", branch)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return nil
}
