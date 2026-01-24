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

	// Initialize RLM state if RLM mode is enabled
	var stateManager *StateManager
	if cfg.RLM.Enabled {
		stateManager = NewStateManager(StateDir)
		if _, err := stateManager.InitSession(); err != nil {
			return fmt.Errorf("failed to initialize RLM session: %w", err)
		}
	} else {
		// Reset implementation plan to initial template (non-RLM mode)
		if err := resetImplementationPlan(cfg.PlanFile); err != nil {
			return err
		}
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

		// Show loop banner before iteration (with phase if RLM mode)
		if cfg.RLM.Enabled && stateManager != nil {
			session, _ := stateManager.LoadSession()
			if session != nil {
				FormatLoopBannerWithPhase(cfg.Output, iteration, session.Phase)
			} else {
				FormatLoopBanner(cfg.Output, iteration)
			}
		} else {
			FormatLoopBanner(cfg.Output, iteration)
		}

		// Run iteration using the provider
		completed, verifyFailed, err := runIterationWithRLM(cfg, provider, iteration, stateManager, verifier)
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

// runIterationWithRLM runs a single iteration with optional RLM state management and verification
func runIterationWithRLM(cfg Config, provider Provider, iteration int, state *StateManager, verifier *Verifier) (completed bool, verifyFailed bool, err error) {
	var promptContent []byte

	// Build prompt based on mode
	if cfg.RLM.Enabled && state != nil {
		// Update session iteration
		session, loadErr := state.LoadSession()
		if loadErr != nil {
			return false, false, fmt.Errorf("failed to load session: %w", loadErr)
		}
		session.Iteration = iteration
		if saveErr := state.SaveSession(session); saveErr != nil {
			return false, false, fmt.Errorf("failed to save session: %w", saveErr)
		}

		// Build RLM prompt
		promptContent, err = buildRLMPrompt(cfg, state, iteration)
		if err != nil {
			return false, false, err
		}
	} else {
		// Build standard prompt
		promptContent, err = buildPromptWithPlan(cfg.PromptFile, cfg.PlanFile, iteration, cfg.MaxIterations, cfg.NoPush)
		if err != nil {
			return false, false, err
		}
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

	// Update RLM state if enabled
	if cfg.RLM.Enabled && state != nil && resultMsg != nil {
		// Update phase if detected
		if resultMsg.RLMPhase != "" {
			if updateErr := state.UpdateIteration(resultMsg.RLMPhase); updateErr != nil {
				fmt.Fprintln(cfg.Output, dimStyle.Render(fmt.Sprintf("Warning: Failed to update RLM phase: %v", updateErr)))
			}
		}

		// Record iteration in history
		historyEntry := HistoryEntry{
			Iteration: iteration,
			Role:      "assistant",
			Content:   fmt.Sprintf("Iteration %d completed", iteration),
			Phase:     resultMsg.RLMPhase,
		}
		if appendErr := state.AppendHistory(historyEntry); appendErr != nil {
			fmt.Fprintln(cfg.Output, dimStyle.Render(fmt.Sprintf("Warning: Failed to append history: %v", appendErr)))
		}
	}

	// Run verification if enabled (in RLM mode, also requires agent signal)
	runVerification := verifier != nil && verifier.HasCommands()
	if cfg.RLM.Enabled {
		// In RLM mode, only run verification when agent signals <rlm:verified>
		runVerification = runVerification && resultMsg != nil && resultMsg.RLMVerified
	}
	if runVerification {
		fmt.Fprintln(cfg.Output)
		fmt.Fprintln(cfg.Output, dimStyle.Render("Running verification..."))

		report := verifier.Run(iteration)

		// Store verification report if state manager available
		if state != nil {
			if storeErr := state.StoreVerification(report); storeErr != nil {
				fmt.Fprintln(cfg.Output, dimStyle.Render(fmt.Sprintf("Warning: Failed to store verification report: %v", storeErr)))
			}
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
