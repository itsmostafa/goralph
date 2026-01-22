package loop

import (
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

	// ImplementationPlanFile is the path to the implementation plan
	ImplementationPlanFile = ".ralph/IMPLEMENTATION_PLAN.md"
)

// Config holds the loop configuration
type Config struct {
	Mode          Mode
	PromptFile    string
	MaxIterations int
	NoPush        bool
	CLI           CLIProvider
	Output        io.Writer
}

// Run executes the agentic loop
func Run(cfg Config) error {
	// Default output to stdout
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	// Create provider once at start
	provider, err := NewProvider(cfg.CLI)
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

	// Reset implementation plan to initial template
	if err := resetImplementationPlan(); err != nil {
		return err
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

		// Run iteration using the provider
		if err := runIteration(cfg, provider, iteration); err != nil {
			return fmt.Errorf("%s iteration failed: %w", provider.Name(), err)
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

func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// resetImplementationPlan resets the implementation plan file to the initial template
func resetImplementationPlan() error {
	template := `# Implementation Plan

## Tasks

<!-- Add tasks here as: - [ ] Task description -->

## Completed

<!-- Completed tasks will be moved here as: - [x] Task description -->
`
	if err := os.WriteFile(ImplementationPlanFile, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to reset implementation plan: %w", err)
	}
	return nil
}

// buildPromptWithPlan reads the prompt file and appends the implementation plan with instructions
func buildPromptWithPlan(promptFile string, mode Mode, iteration int, maxIterations int) ([]byte, error) {
	// Read the prompt file
	promptContent, err := os.ReadFile(promptFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt file: %w", err)
	}

	// Read the implementation plan
	planContent, err := os.ReadFile(ImplementationPlanFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read implementation plan: %w", err)
	}

	// Build iteration display string
	var iterationStr string
	if maxIterations > 0 {
		iterationStr = fmt.Sprintf("%d/%d", iteration, maxIterations)
	} else {
		iterationStr = fmt.Sprintf("%d/unlimited", iteration)
	}

	// Build system context
	systemContext := fmt.Sprintf(`# System Context

You are running in a **goralph agentic loop** - an automated iteration system that manages your context between runs.

**Key facts:**
- Iteration: %s
- Mode: %s
- Each iteration runs with a fresh context window
- Focus on completing ONE task per iteration
- After completing a task: update the implementation plan, commit changes, then exit
- The loop will automatically restart and push your changes

**Workflow:**
1. Study the implementation plan below
2. Pick the most important uncompleted task
3. Complete that single task
4. Update the implementation plan to mark it complete
5. Commit with a descriptive message
6. Exit - the loop handles the rest

---

`, iterationStr, mode)

	// Build task guidance based on whether max iterations is set
	var taskGuidance string
	if maxIterations > 0 {
		taskGuidance = fmt.Sprintf(`If the Tasks section is empty, analyze the project and break the work into approximately %d tasks (one per iteration).`, maxIterations)
	} else {
		taskGuidance = `If the Tasks section is empty, analyze the project and add a comprehensive list of implementation tasks.`
	}

	// Build instructions to append
	instructions := `
---

# Implementation Plan Instructions

Study the implementation plan below. Pick the most important uncompleted task.

` + taskGuidance + `

Complete ONE task, then:
1. Update ` + "`" + ImplementationPlanFile + "`" + ` to mark the task as completed (move to Completed section)
2. Commit your changes with a descriptive message
3. Exit

The loop will automatically restart with a fresh context window.

---

# Current Implementation Plan

` + "```markdown\n" + string(planContent) + "\n```"

	// Combine system context + prompt + instructions + plan
	combined := append([]byte(systemContext), promptContent...)
	combined = append(combined, []byte(instructions)...)
	return combined, nil
}

func runIteration(cfg Config, provider Provider, iteration int) error {
	// Build prompt with implementation plan
	promptContent, err := buildPromptWithPlan(cfg.PromptFile, cfg.Mode, iteration, cfg.MaxIterations)
	if err != nil {
		return err
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

	// Build the command using the provider
	cmd, err := provider.BuildCommand(promptContent)
	if err != nil {
		return fmt.Errorf("failed to build command: %w", err)
	}

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
		return fmt.Errorf("failed to start %s: %w", provider.Name(), err)
	}

	// Write prompt to stdin and close
	if _, err := stdin.Write(promptContent); err != nil {
		return fmt.Errorf("failed to write to stdin: %w", err)
	}
	stdin.Close()

	// Show progress indicator
	fmt.Fprintln(cfg.Output, dimStyle.Render(fmt.Sprintf("Running %s...", provider.Name())))

	// Parse output using the provider and write to log file
	resultMsg, err := provider.ParseOutput(stdout, cfg.Output, logFile)
	if err != nil {
		return fmt.Errorf("failed to parse output: %w", err)
	}

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s exited with error: %w", provider.Name(), err)
	}

	// Display the final result summary
	fmt.Fprintln(cfg.Output)
	if resultMsg != nil {
		FormatIterationSummary(cfg.Output, *resultMsg)
	} else {
		fmt.Fprintln(cfg.Output, dimStyle.Render(fmt.Sprintf("Warning: No result message received from %s", provider.Name())))
	}

	return nil
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
