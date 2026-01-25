package loop

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RLMRunner implements ModeRunner for true RLM mode with REPL environment
type RLMRunner struct {
	output   io.Writer
	session  *RLMSession
	repl     REPLExecutor
	bridge   LMBridge
	maxDepth int
	provider Provider
}

// NewRLMRunner creates a new RLM mode runner
func NewRLMRunner(maxDepth int) *RLMRunner {
	return &RLMRunner{
		output:   os.Stdout,
		maxDepth: maxDepth,
	}
}

// Name returns the mode name
func (r *RLMRunner) Name() string {
	return "rlm"
}

// Initialize sets up RLM mode with REPL and LLM bridge
func (r *RLMRunner) Initialize(cfg Config) error {
	// Create provider for LLM bridge
	provider, err := NewProvider(cfg.Agent)
	if err != nil {
		return fmt.Errorf("failed to create provider for RLM: %w", err)
	}
	r.provider = provider

	// Create LM Bridge with depth limiting
	r.bridge = NewProviderBridge(provider, r.maxDepth)

	// Create Go REPL with bridge
	r.repl = NewGoREPL(r.bridge)
	if err := r.repl.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize REPL: %w", err)
	}

	// Create session
	r.session = &RLMSession{
		SessionID:  uuid.New().String(),
		Iteration:  0,
		MaxDepth:   r.maxDepth,
		IsComplete: false,
		StartedAt:  time.Now(),
	}

	// Create state directory for session persistence
	if err := os.MkdirAll(StateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Save initial session state
	if err := r.saveSession(); err != nil {
		return fmt.Errorf("failed to save initial session: %w", err)
	}

	return nil
}

// BuildPrompt constructs the prompt for the given iteration
func (r *RLMRunner) BuildPrompt(cfg Config, iteration int) ([]byte, error) {
	// Update session iteration
	r.session.Iteration = iteration
	if err := r.saveSession(); err != nil {
		fmt.Fprintf(r.output, "Warning: Failed to save session: %v\n", err)
	}

	// Read the prompt file as context
	contextContent, err := os.ReadFile(cfg.PromptFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt file: %w", err)
	}

	// Load context into REPL
	if err := r.repl.SetContext(string(contextContent)); err != nil {
		return nil, fmt.Errorf("failed to set REPL context: %w", err)
	}

	// Build RLM system prompt with metadata
	systemPrompt := BuildRLMSystemPrompt(iteration, r.maxDepth, len(contextContent))

	// Build iteration context
	var iterationContext string
	if cfg.MaxIterations > 0 {
		iterationContext = fmt.Sprintf("This is iteration %d of %d.\n\n", iteration, cfg.MaxIterations)
	} else {
		iterationContext = fmt.Sprintf("This is iteration %d.\n\n", iteration)
	}

	// Append user task
	prompt := systemPrompt + iterationContext + "# Task\n\n" + string(contextContent)

	return []byte(prompt), nil
}

// HandleResult processes the result from an agent iteration
func (r *RLMRunner) HandleResult(cfg Config, result *ResultMessage, iteration int) error {
	if result == nil {
		return nil
	}

	// Extract and execute ```repl code blocks from result
	codeBlocks := ExtractREPLCodeBlocks(result.Result)

	for i, code := range codeBlocks {
		fmt.Fprintf(r.output, "\n%s\n", dimStyle.Render(fmt.Sprintf("Executing REPL block %d...", i+1)))

		replResult, err := r.repl.Execute(code)
		if err != nil {
			fmt.Fprintf(r.output, "%s\n", errorStyle.Render(fmt.Sprintf("REPL Error: %v", err)))
			continue
		}

		// Print REPL output
		if replResult.Output != "" {
			fmt.Fprintf(r.output, "%s\n", replResult.Output)
		}

		// Report errors
		if replResult.Error != "" {
			fmt.Fprintf(r.output, "%s\n", errorStyle.Render(fmt.Sprintf("REPL Error: %s", replResult.Error)))
		}

		// Track LLM calls
		r.session.TotalLLMCalls += len(replResult.LLMCalls)
	}

	// Check for FINAL(answer) marker
	if answer := ExtractFinalAnswer(result.Result); answer != "" {
		r.session.FinalAnswer = answer
		r.session.IsComplete = true
		result.SessionComplete = true
		fmt.Fprintf(r.output, "\n%s\n", successStyle.Render("Final Answer: "+answer))
	}

	// Check for FINAL_VAR(variable_name) marker
	if varName := ExtractFinalVar(result.Result); varName != "" {
		value, err := r.repl.GetVariable(varName)
		if err != nil {
			fmt.Fprintf(r.output, "%s\n", errorStyle.Render(fmt.Sprintf("Failed to get variable %q: %v", varName, err)))
		} else {
			r.session.FinalAnswer = fmt.Sprintf("%v", value)
			r.session.IsComplete = true
			result.SessionComplete = true
			fmt.Fprintf(r.output, "\n%s\n", successStyle.Render(fmt.Sprintf("Final Answer (%s): %v", varName, value)))
		}
	}

	// Check for legacy <promise>COMPLETE</promise> marker
	if strings.Contains(result.Result, CompletionPromise) {
		r.session.IsComplete = true
		result.SessionComplete = true
	}

	// Save session state
	if err := r.saveSession(); err != nil {
		fmt.Fprintf(r.output, "Warning: Failed to save session: %v\n", err)
	}

	return nil
}

// GetBannerInfo returns information for rendering the loop banner
func (r *RLMRunner) GetBannerInfo() BannerInfo {
	// RLM mode doesn't use phases
	return BannerInfo{
		Phase: "REPL",
	}
}

// ShouldRunVerification determines if verification should run
func (r *RLMRunner) ShouldRunVerification(cfg Config, result *ResultMessage) bool {
	// Run verification only when session is complete
	return cfg.VerifyEnabled && r.session != nil && r.session.IsComplete
}

// StoreVerification stores the verification report
func (r *RLMRunner) StoreVerification(report VerificationReport) error {
	// Create verification directory
	verifyDir := filepath.Join(StateDir, "verification")
	if err := os.MkdirAll(verifyDir, 0755); err != nil {
		return fmt.Errorf("failed to create verification directory: %w", err)
	}

	// Write verification report
	report.Timestamp = time.Now()
	filename := fmt.Sprintf("verify_%04d_%d.json", report.Iteration, time.Now().UnixMilli())
	path := filepath.Join(verifyDir, filename)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal verification report: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write verification report: %w", err)
	}

	return nil
}

// Output returns the writer for mode output
func (r *RLMRunner) Output() io.Writer {
	return r.output
}

// SetOutput sets the writer for mode output
func (r *RLMRunner) SetOutput(w io.Writer) {
	r.output = w
}

// saveSession persists the session state to disk
func (r *RLMRunner) saveSession() error {
	path := filepath.Join(StateDir, "session.json")

	data, err := json.MarshalIndent(r.session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write session: %w", err)
	}

	return nil
}

// StateDir is the directory for RLM state files
const StateDir = ".ralph/state"
