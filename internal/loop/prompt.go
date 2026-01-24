package loop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

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

// buildRLMPrompt builds a prompt with RLM context and phase-specific guidance
func buildRLMPrompt(cfg Config, state *StateManager, iteration int) ([]byte, error) {
	// Read the prompt file
	promptContent, err := os.ReadFile(cfg.PromptFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt file: %w", err)
	}

	// Load session state
	session, err := state.LoadSession()
	if err != nil {
		return nil, fmt.Errorf("failed to load session state: %w", err)
	}

	// Create phase router and infer current phase
	router := NewPhaseRouter(state)
	phase, err := router.InferPhase()
	if err != nil {
		phase = PhasePlan // Default to plan on error
	}

	// Get phase-specific guidance
	phaseGuidance := router.GetPhaseGuidance(phase)

	// Load context manifest for inclusion
	context, err := state.GetContext()
	if err != nil {
		context = &ContextManifest{} // Use empty context on error
	}

	// Get recent history for context
	recentHistory, _ := state.GetRecentHistory(5)
	historySection := formatHistorySection(recentHistory)

	// Build iteration display string
	var iterationStr string
	if cfg.MaxIterations > 0 {
		iterationStr = fmt.Sprintf("%d/%d", iteration, cfg.MaxIterations)
	} else {
		iterationStr = fmt.Sprintf("%d/unlimited", iteration)
	}

	// Build RLM principle #3 based on noPush setting
	var rlmPrinciple3 string
	if cfg.NoPush {
		rlmPrinciple3 = "3. **One task per iteration**: Complete ONE task, implement changes, update state, exit."
	} else {
		rlmPrinciple3 = "3. **One task per iteration**: Complete ONE task, update state, commit, exit."
	}

	// Build system context with RLM principles
	systemContext := fmt.Sprintf(`# System Context

You are running in a **goralph RLM-enhanced agentic loop**.

## RLM Principles

1. **Context is external**: The full codebase is NOT in your context. Use tools to explore.
2. **State persists**: Discoveries are stored in %s. Reference previous findings.
%s
4. **Verify before commit**: Run relevant checks before marking complete.

## Session Info

- **Iteration:** %s
- **Session ID:** %s
- **Current Phase:** %s (%s)
- **Depth:** %d/%d

## Available State Files

- Context manifest: %s/context.json
- Previous searches: %s/search/
- Narrowed sets: %s/narrow/
- History: %s/history.jsonl
- Verification reports: %s/verification/

---

`, StateDir, rlmPrinciple3, iterationStr, session.SessionID, phase, PhaseDisplayName(phase),
		session.Depth, cfg.RLM.MaxDepth,
		StateDir, StateDir, StateDir, StateDir, StateDir)

	// Add context summary if available
	contextSection := formatContextSection(context)
	if contextSection != "" {
		systemContext += contextSection + "\n---\n\n"
	}

	// Add recent history if available
	if historySection != "" {
		systemContext += historySection + "\n---\n\n"
	}

	// Add phase-specific guidance
	systemContext += phaseGuidance + "\n\n---\n\n"

	// Add RLM marker instructions
	systemContext += getRLMMarkerInstructions(cfg.NoPush) + "\n---\n\n"

	// Add the user's task
	systemContext += "# Task\n\n"

	// Combine all parts
	combined := append([]byte(systemContext), promptContent...)
	return combined, nil
}

// formatContextSection formats the context manifest for prompt inclusion
func formatContextSection(ctx *ContextManifest) string {
	if ctx == nil {
		return ""
	}

	var section string

	// Add task summary if available
	if ctx.Task.Summary != "" {
		section += "## Task Understanding\n\n"
		section += fmt.Sprintf("**Summary:** %s\n\n", ctx.Task.Summary)
		if len(ctx.Task.Objectives) > 0 {
			section += "**Objectives:**\n"
			for _, obj := range ctx.Task.Objectives {
				section += fmt.Sprintf("- %s\n", obj)
			}
			section += "\n"
		}
	}

	// Add focus files if available
	if len(ctx.Focus.Files) > 0 {
		section += "## Current Focus\n\n"
		section += "**Files:**\n"
		for _, f := range ctx.Focus.Files {
			section += fmt.Sprintf("- %s\n", f)
		}
		section += "\n"
	}

	// Add recent discoveries (last 5)
	if len(ctx.Discoveries) > 0 {
		section += "## Recent Discoveries\n\n"
		start := 0
		if len(ctx.Discoveries) > 5 {
			start = len(ctx.Discoveries) - 5
		}
		for _, d := range ctx.Discoveries[start:] {
			section += fmt.Sprintf("- [%s] %s: %s\n", d.Phase, d.Type, d.Description)
		}
		section += "\n"
	}

	return section
}

// formatHistorySection formats recent history entries for prompt inclusion
func formatHistorySection(entries []HistoryEntry) string {
	if len(entries) == 0 {
		return ""
	}

	section := "## Recent History\n\n"
	for _, e := range entries {
		// Truncate content for prompt inclusion
		content := e.Content
		if len(content) > 200 {
			content = content[:197] + "..."
		}
		section += fmt.Sprintf("- [Iter %d, %s] %s\n", e.Iteration, e.Phase, content)
	}
	return section
}

// getRLMMarkerInstructions returns instructions for RLM output markers based on noPush setting
func getRLMMarkerInstructions(noPush bool) string {
	base := "## RLM Output Markers\n\n" +
		"Use these markers to communicate state transitions:\n\n" +
		"1. **Phase transition:** Signal which phase to enter next:\n" +
		"   `<rlm:phase>PHASE_NAME</rlm:phase>`\n" +
		"   Valid phases: PLAN, SEARCH, NARROW, ACT, VERIFY\n\n" +
		"2. **Verification passed:** Signal that verification succeeded:\n" +
		"   `<rlm:verified>true</rlm:verified>`\n\n" +
		"3. **Session complete:** Signal all tasks are done:\n" +
		"   `<promise>COMPLETE</promise>`"

	if !noPush {
		base += "\n\n**Important:** Always commit your changes before signaling completion."
	}

	return base
}

// saveContextToFile saves the context manifest to a JSON file for agent access
func saveContextToFile(ctx *ContextManifest, stateDir string) error {
	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	path := filepath.Join(stateDir, "context.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write context file: %w", err)
	}

	return nil
}
