package loop

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// NewModeRunner creates a new ModeRunner based on the mode
func NewModeRunner(mode Mode, cfg Config) (ModeRunner, error) {
	switch mode {
	case ModeRalph:
		return NewRalphRunner(), nil
	case ModeRLM:
		return NewRLMRunner(cfg.RLMMaxDepth), nil
	default:
		return nil, fmt.Errorf("unknown mode: %s", mode)
	}
}

// Run executes the agentic loop
func Run(cfg Config) error {
	// Default output to stdout
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	// Default mode to ralph
	if cfg.Mode == "" {
		cfg.Mode = ModeRalph
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

	// Create mode runner
	runner, err := NewModeRunner(cfg.Mode, cfg)
	if err != nil {
		return fmt.Errorf("failed to create mode runner: %w", err)
	}
	runner.SetOutput(cfg.Output)

	// Initialize the mode runner
	if err := runner.Initialize(cfg); err != nil {
		return err
	}

	// Print configuration
	FormatHeader(cfg.Output, cfg, branch, provider.Model())

	// Create verifier if verification is enabled
	var verifier *Verifier
	if cfg.VerifyEnabled {
		verifier = NewVerifier(cfg.VerifyCommands)
		if !verifier.HasCommands() {
			fmt.Fprintln(cfg.Output, dimStyle.Render("Warning: No verification commands detected for project type"))
		}
	}

	iteration := 0
	for {
		iteration++

		// Check max iterations
		if cfg.MaxIterations > 0 && iteration > cfg.MaxIterations {
			FormatMaxIterations(cfg.Output, cfg.MaxIterations)
			break
		}

		// Show loop banner before iteration (with phase if available)
		bannerInfo := runner.GetBannerInfo()
		if bannerInfo.Phase != "" {
			FormatLoopBannerWithPhase(cfg.Output, iteration, bannerInfo.Phase)
		} else {
			FormatLoopBanner(cfg.Output, iteration)
		}

		// Run iteration using the mode runner
		completed, verifyFailed, err := runIteration(cfg, provider, iteration, runner, verifier)
		if err != nil {
			return fmt.Errorf("%s iteration failed: %w", provider.Name(), err)
		}
		if completed {
			FormatSessionComplete(cfg.Output)
			break
		}

		// Skip push if verification failed
		if verifyFailed {
			fmt.Fprintln(cfg.Output, dimStyle.Render("Skipping push due to verification failure"))
			continue
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

// runIteration runs a single iteration with the mode runner and verification
func runIteration(cfg Config, provider Provider, iteration int, runner ModeRunner, verifier *Verifier) (completed bool, verifyFailed bool, err error) {
	// Build prompt using mode runner
	promptContent, err := runner.BuildPrompt(cfg, iteration)
	if err != nil {
		return false, false, err
	}

	// Create logs directory
	logsDir := filepath.Join(".ralph", "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return false, false, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Generate timestamped log filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	logPath := filepath.Join(logsDir, timestamp+".jsonl")

	// Create log file
	logFile, err := os.Create(logPath)
	if err != nil {
		return false, false, fmt.Errorf("failed to create log file: %w", err)
	}
	defer logFile.Close()

	// Build the command using the provider
	cmd, err := provider.BuildCommand(promptContent)
	if err != nil {
		return false, false, fmt.Errorf("failed to build command: %w", err)
	}

	// Set up stdin with prompt content
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return false, false, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Capture stdout for parsing
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return false, false, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Connect stderr to terminal
	cmd.Stderr = os.Stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return false, false, fmt.Errorf("failed to start %s: %w", provider.Name(), err)
	}

	// Write prompt to stdin and close
	if _, err := stdin.Write(promptContent); err != nil {
		return false, false, fmt.Errorf("failed to write to stdin: %w", err)
	}
	stdin.Close()

	// Show progress indicator
	fmt.Fprintln(cfg.Output, dimStyle.Render(fmt.Sprintf("Running %s...", provider.Name())))

	// Track duration externally for providers that don't report it
	startTime := time.Now()

	// Parse output using the provider and write to log file
	resultMsg, err := provider.ParseOutput(stdout, cfg.Output, logFile)
	if err != nil {
		return false, false, fmt.Errorf("failed to parse output: %w", err)
	}

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return false, false, fmt.Errorf("%s exited with error: %w", provider.Name(), err)
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

	// Handle result using mode runner
	if resultMsg != nil {
		if err := runner.HandleResult(cfg, resultMsg, iteration); err != nil {
			fmt.Fprintln(cfg.Output, dimStyle.Render(fmt.Sprintf("Warning: Failed to handle result: %v", err)))
		}
	}

	// Run verification if mode decides to
	runVerification := verifier != nil && verifier.HasCommands() && runner.ShouldRunVerification(cfg, resultMsg)
	if runVerification {
		fmt.Fprintln(cfg.Output)
		fmt.Fprintln(cfg.Output, dimStyle.Render("Running verification..."))

		report := verifier.Run(iteration)

		// Store verification report using mode runner
		if err := runner.StoreVerification(report); err != nil {
			fmt.Fprintln(cfg.Output, dimStyle.Render(fmt.Sprintf("Warning: Failed to store verification report: %v", err)))
		}

		if report.Passed {
			FormatVerificationPassed(cfg.Output)
		} else {
			FormatVerificationFailed(cfg.Output, report)
			return false, true, nil // Continue loop but skip push
		}
	}

	// Check if agent signaled session completion
	sessionComplete := resultMsg != nil && resultMsg.SessionComplete
	return sessionComplete, false, nil
}
