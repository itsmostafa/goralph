package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Run executes the agentic loop
func Run(cfg Config) error {
	// Default output to stdout
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	// Create provider once at start
	provider, err := NewProvider(cfg.Agent)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
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

	// Create plans directory if it doesn't exist
	if err := os.MkdirAll(PlansDir, 0755); err != nil {
		return fmt.Errorf("failed to create plans directory: %w", err)
	}

	// Reset implementation plan to initial template
	if err := resetImplementationPlan(cfg.PlanFile); err != nil {
		return err
	}

	// Print configuration
	FormatHeader(cfg.Output, cfg, branch, provider.Model())

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

		// Run iteration using the provider
		completed, err := runIteration(cfg, provider, iteration)
		if err != nil {
			return fmt.Errorf("%s iteration failed: %w", provider.Name(), err)
		}
		if completed {
			FormatSessionComplete(cfg.Output)
			break
		}

		// Push changes unless --no-push is set
		if !cfg.NoPush {
			if err := pushChanges(branch); err != nil {
				return fmt.Errorf("failed to push changes: %w", err)
			}
		}
	}

	return nil
}

func runIteration(cfg Config, provider Provider, iteration int) (bool, error) {
	// Build prompt with implementation plan
	promptContent, err := buildPromptWithPlan(cfg.PromptFile, cfg.PlanFile, cfg.Mode, iteration, cfg.MaxIterations)
	if err != nil {
		return false, err
	}

	// Create logs directory
	logsDir := filepath.Join(".ralph", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Generate timestamped log filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logsDir, timestamp+".jsonl")

	// Create log file
	logFile, err := os.Create(logPath)
	if err != nil {
		return false, fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	// Check if provider supports direct execution (e.g., RLM)
	if directRunner, ok := provider.(DirectRunner); ok {
		return runDirectIteration(cfg, directRunner, promptContent, logFile)
	}

	// Build the command using the provider
	cmd, err := provider.BuildCommand(promptContent)
	if err != nil {
		return false, fmt.Errorf("failed to build command: %w", err)
	}

	// Set up stdin with prompt content
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return false, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Capture stdout for parsing
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return false, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Connect stderr to terminal
	cmd.Stderr = os.Stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return false, fmt.Errorf("failed to start %s: %w", provider.Name(), err)
	}

	// Write prompt to stdin and close
	if _, err := stdin.Write(promptContent); err != nil {
		return false, fmt.Errorf("failed to write to stdin: %w", err)
	}
	stdin.Close()

	// Show progress indicator
	fmt.Fprintln(cfg.Output, dimStyle.Render(fmt.Sprintf("Running %s...", provider.Name())))

	// Track duration externally for providers that don't report it
	startTime := time.Now()

	// Parse output using the provider and write to log file
	resultMsg, err := provider.ParseOutput(stdout, cfg.Output, logFile)
	if err != nil {
		return false, fmt.Errorf("failed to parse output: %w", err)
	}

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return false, fmt.Errorf("%s exited with error: %w", provider.Name(), err)
	}

	// Inject duration if provider didn't supply it
	if resultMsg != nil && resultMsg.DurationMs == 0 {
		resultMsg.DurationMs = int(time.Since(startTime).Milliseconds())
	}

	// Display the final result summary
	fmt.Fprintln(cfg.Output)
	if resultMsg != nil {
		FormatIterationSummary(cfg.Output, *resultMsg)
	} else {
		fmt.Fprintln(cfg.Output, dimStyle.Render(fmt.Sprintf("Warning: No result message received from %s", provider.Name())))
	}

	// Check if agent signaled session completion
	sessionComplete := resultMsg != nil && resultMsg.SessionComplete
	return sessionComplete, nil
}

// runDirectIteration handles providers that run directly without external commands.
func runDirectIteration(cfg Config, provider DirectRunner, promptContent []byte, logFile *os.File) (bool, error) {
	// Show progress indicator
	fmt.Fprintln(cfg.Output, dimStyle.Render(fmt.Sprintf("Running %s...", provider.Name())))

	// Run the provider directly
	resultMsg, err := provider.RunDirect(promptContent, cfg.Output, logFile)
	if err != nil {
		// Still display summary on error
		if resultMsg != nil {
			fmt.Fprintln(cfg.Output)
			FormatIterationSummary(cfg.Output, *resultMsg)
		}
		return false, fmt.Errorf("%s execution failed: %w", provider.Name(), err)
	}

	// Display the final result summary
	fmt.Fprintln(cfg.Output)
	if resultMsg != nil {
		FormatIterationSummary(cfg.Output, *resultMsg)
	} else {
		fmt.Fprintln(cfg.Output, dimStyle.Render(fmt.Sprintf("Warning: No result message received from %s", provider.Name())))
	}

	// Check if agent signaled session completion
	sessionComplete := resultMsg != nil && resultMsg.SessionComplete
	return sessionComplete, nil
}
