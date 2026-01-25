package loop

import (
	"fmt"
	"io"
	"os"
)

// RalphRunner implements ModeRunner for ralph mode
type RalphRunner struct {
	output io.Writer
}

// NewRalphRunner creates a new ralph mode runner
func NewRalphRunner() *RalphRunner {
	return &RalphRunner{
		output: os.Stdout,
	}
}

// Name returns the mode name
func (r *RalphRunner) Name() string {
	return "ralph"
}

// Initialize sets up ralph mode - resets the implementation plan
func (r *RalphRunner) Initialize(cfg Config) error {
	return resetImplementationPlan(cfg.PlanFile)
}

// BuildPrompt constructs the prompt for the given iteration
func (r *RalphRunner) BuildPrompt(cfg Config, iteration int) ([]byte, error) {
	return buildPromptWithPlan(cfg.PromptFile, cfg.PlanFile, iteration, cfg.MaxIterations, cfg.NoPush)
}

// HandleResult processes the result from an agent iteration
// Ralph mode doesn't need to handle results specially
func (r *RalphRunner) HandleResult(cfg Config, result *ResultMessage, iteration int) error {
	// Ralph mode doesn't persist state between iterations
	return nil
}

// GetBannerInfo returns information for rendering the loop banner
func (r *RalphRunner) GetBannerInfo() BannerInfo {
	// Ralph mode doesn't have phase information
	return BannerInfo{}
}

// ShouldRunVerification determines if verification should run
// In ralph mode, verification runs when enabled (no agent signal required)
func (r *RalphRunner) ShouldRunVerification(cfg Config, result *ResultMessage) bool {
	return cfg.VerifyEnabled
}

// StoreVerification stores the verification report
// Ralph mode doesn't persist verification reports
func (r *RalphRunner) StoreVerification(report VerificationReport) error {
	// Ralph mode doesn't have persistent state
	return nil
}

// Output returns the writer for mode output
func (r *RalphRunner) Output() io.Writer {
	return r.output
}

// SetOutput sets the writer for mode output
func (r *RalphRunner) SetOutput(w io.Writer) {
	r.output = w
}

// resetImplementationPlan resets the implementation plan file to the initial template
func resetImplementationPlan(planFile string) error {
	template := `# Implementation Plan

## Tasks

<!-- Add tasks here as: - [ ] Task description -->

## Completed

<!-- Completed tasks will be moved here as: - [x] Task description -->
`
	if err := os.WriteFile(planFile, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to reset implementation plan: %w", err)
	}
	return nil
}

// buildPromptWithPlan reads the prompt file and appends the implementation plan with instructions
func buildPromptWithPlan(promptFile string, planFile string, iteration int, maxIterations int, noPush bool) ([]byte, error) {
	// Read the prompt file
	promptContent, err := os.ReadFile(promptFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt file: %w", err)
	}

	// Read the implementation plan
	planContent, err := os.ReadFile(planFile)
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

	// Build system context based on noPush setting
	var systemContext string
	if noPush {
		systemContext = fmt.Sprintf(`# System Context

You are running in a **goralph agentic loop** - an automated iteration system that manages your context between runs.

**Key facts:**
- Iteration: %s
- Each iteration runs with a fresh context window
- Focus on completing ONE task per iteration
- After completing a task: update the implementation plan, then exit
- The loop will automatically restart

**Workflow:**
1. Study the implementation plan below
2. Pick the most important uncompleted task
3. Complete that single task
4. Update the implementation plan to mark it complete
5. Exit - the loop handles the rest

---

`, iterationStr)
	} else {
		systemContext = fmt.Sprintf(`# System Context

You are running in a **goralph agentic loop** - an automated iteration system that manages your context between runs.

**Key facts:**
- Iteration: %s
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

`, iterationStr)
	}

	// Build task guidance based on whether max iterations is set
	var taskGuidance string
	if maxIterations > 0 {
		taskGuidance = fmt.Sprintf(`If the Tasks section is empty, analyze the project and break the work into approximately %d tasks (one per iteration).`, maxIterations)
	} else {
		taskGuidance = `If the Tasks section is empty, analyze the project and add a comprehensive list of implementation tasks.`
	}

	// Build completion steps based on noPush setting
	var completionSteps string
	if noPush {
		completionSteps = `Complete ONE task, then:
1. Update ` + "`" + planFile + "`" + ` to mark the task as completed (move to Completed section)
2. Exit`
	} else {
		completionSteps = `Complete ONE task, then:
1. Update ` + "`" + planFile + "`" + ` to mark the task as completed (move to Completed section)
2. Commit your changes with a descriptive message
3. Exit`
	}

	// Build instructions to append
	instructions := `
---

# Implementation Plan Instructions

Study the implementation plan below. Pick the most important uncompleted task.

` + taskGuidance + `

` + completionSteps + `

**Completion Promise:**
When ALL tasks in the plan are complete and there is no more work to do, output this exact line:
` + "`" + CompletionPromise + "`" + `
This signals the loop to exit gracefully instead of continuing to the next iteration.

The loop will automatically restart with a fresh context window.

---

# Current Implementation Plan

` + "```markdown\n" + string(planContent) + "\n```"

	// Combine system context + prompt + instructions + plan
	combined := append([]byte(systemContext), promptContent...)
	combined = append(combined, []byte(instructions)...)
	return combined, nil
}
