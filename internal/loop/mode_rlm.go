package loop

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// RLMRunner implements ModeRunner for RLM mode
type RLMRunner struct {
	output       io.Writer
	stateManager *StateManager
	maxDepth     int
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

// Initialize sets up RLM mode - creates state manager and initializes session
func (r *RLMRunner) Initialize(cfg Config) error {
	r.stateManager = NewStateManager(StateDir)
	if _, err := r.stateManager.InitSession(); err != nil {
		return fmt.Errorf("failed to initialize RLM session: %w", err)
	}
	return nil
}

// BuildPrompt constructs the prompt for the given iteration
func (r *RLMRunner) BuildPrompt(cfg Config, iteration int) ([]byte, error) {
	// Update session iteration
	session, err := r.stateManager.LoadSession()
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}
	session.Iteration = iteration
	if err := r.stateManager.SaveSession(session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return r.buildRLMPrompt(cfg, iteration)
}

// HandleResult processes the result from an agent iteration
func (r *RLMRunner) HandleResult(cfg Config, result *ResultMessage, iteration int) error {
	if result == nil {
		return nil
	}

	// Update phase if detected
	if result.ModePhase != "" {
		if err := r.stateManager.UpdateIteration(Phase(result.ModePhase)); err != nil {
			fmt.Fprintf(r.output, "Warning: Failed to update RLM phase: %v\n", err)
		}
	}

	// Record iteration in history
	historyEntry := HistoryEntry{
		Iteration: iteration,
		Role:      "assistant",
		Content:   fmt.Sprintf("Iteration %d completed", iteration),
		Phase:     Phase(result.ModePhase),
	}
	if err := r.stateManager.AppendHistory(historyEntry); err != nil {
		fmt.Fprintf(r.output, "Warning: Failed to append history: %v\n", err)
	}

	return nil
}

// GetBannerInfo returns information for rendering the loop banner
func (r *RLMRunner) GetBannerInfo() BannerInfo {
	if r.stateManager == nil {
		return BannerInfo{}
	}

	router := NewPhaseRouter(r.stateManager)
	phase, _ := router.InferPhase()
	return BannerInfo{
		Phase: PhaseDisplayName(phase),
	}
}

// ShouldRunVerification determines if verification should run
// In RLM mode, verification runs when agent signals <rlm:verified>
func (r *RLMRunner) ShouldRunVerification(cfg Config, result *ResultMessage) bool {
	return cfg.VerifyEnabled && result != nil && result.ModeVerified
}

// StoreVerification stores the verification report
func (r *RLMRunner) StoreVerification(report VerificationReport) error {
	if r.stateManager == nil {
		return nil
	}
	return r.stateManager.StoreVerification(report)
}

// Output returns the writer for mode output
func (r *RLMRunner) Output() io.Writer {
	return r.output
}

// SetOutput sets the writer for mode output
func (r *RLMRunner) SetOutput(w io.Writer) {
	r.output = w
}

// buildRLMPrompt builds a prompt with RLM context and phase-specific guidance
func (r *RLMRunner) buildRLMPrompt(cfg Config, iteration int) ([]byte, error) {
	// Read the prompt file
	promptContent, err := os.ReadFile(cfg.PromptFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt file: %w", err)
	}

	// Load session state
	session, err := r.stateManager.LoadSession()
	if err != nil {
		return nil, fmt.Errorf("failed to load session state: %w", err)
	}

	// Create phase router and infer current phase
	router := NewPhaseRouter(r.stateManager)
	phase, err := router.InferPhase()
	if err != nil {
		phase = PhasePlan // Default to plan on error
	}

	// Get phase-specific guidance
	phaseGuidance := router.GetPhaseGuidance(phase)

	// Load context manifest for inclusion
	context, err := r.stateManager.GetContext()
	if err != nil {
		context = &ContextManifest{} // Use empty context on error
	}

	// Get recent history for context
	recentHistory, _ := r.stateManager.GetRecentHistory(5)
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
		session.Depth, r.maxDepth,
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

	// Add state file conventions
	systemContext += stateFileInstructions + "\n\n---\n\n"

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

// stateFileInstructions provides guidance for writing state files
const stateFileInstructions = `## State File Conventions

**IMPORTANT:** You must write state files yourself using the Write tool. The loop cannot call your methods.

**File naming:**
- Search results: .ralph/state/search/search_NNNN_TIMESTAMP.json (NNNN = zero-padded iteration, e.g., 0001)
- Narrowed sets: .ralph/state/narrow/narrow_NNNN_TIMESTAMP.json
- Act results: .ralph/state/results/act_NNNN_TIMESTAMP.json
- Verification: .ralph/state/verification/verify_NNNN_TIMESTAMP.json

**Timestamps:** Use ISO 8601 format (e.g., "2024-01-23T10:30:00Z")

**Reading previous state:**
- Read .ralph/state/context.json for accumulated discoveries and focus
- Read files in .ralph/state/search/ to see previous searches
- Read files in .ralph/state/narrow/ to see previous narrowed sets

**Writing state:**
- Use the Write tool to create JSON files in the appropriate directories
- For context.json: Read first, merge your new data with existing, then write back
- Create directories if needed (they should already exist)

**Context.json structure:**
The context.json file accumulates information across iterations. Always preserve existing data when updating:
` + "```json" + `
{
  "task": {"summary": "...", "objectives": [...], "constraints": [...]},
  "codebase": {"root_dir": ".", "language": "go", "build_system": "taskfile"},
  "discoveries": [...],
  "focus": {"files": [...], "functions": [...], "tests": [...]},
  "last_updated": "..."
}
` + "```" + `
`

// =====================================
// RLM Types
// =====================================

// Phase represents the current phase in the RLM cycle
type Phase string

const (
	// PhasePlan is the planning phase - analyze task and create approach
	PhasePlan Phase = "PLAN"
	// PhaseSearch is the search phase - explore codebase to find relevant code
	PhaseSearch Phase = "SEARCH"
	// PhaseNarrow is the narrow phase - focus on specific files/functions
	PhaseNarrow Phase = "NARROW"
	// PhaseAct is the action phase - make changes to code
	PhaseAct Phase = "ACT"
	// PhaseVerify is the verify phase - run tests and validate changes
	PhaseVerify Phase = "VERIFY"
)

// StateDir is the directory for RLM state files
const StateDir = ".ralph/state"

// SessionState represents the current RLM session state
type SessionState struct {
	SessionID   string    `json:"session_id"`
	Iteration   int       `json:"iteration"`
	Depth       int       `json:"depth"`
	Phase       Phase     `json:"phase"`
	StartedAt   time.Time `json:"started_at"`
	LastUpdated time.Time `json:"last_updated"`
}

// ContextManifest tracks discovered context across iterations
type ContextManifest struct {
	Task        TaskInfo     `json:"task"`
	Codebase    CodebaseInfo `json:"codebase"`
	Discoveries []Discovery  `json:"discoveries"`
	Focus       FocusSet     `json:"focus"`
	LastUpdated time.Time    `json:"last_updated"`
}

// TaskInfo contains parsed task information
type TaskInfo struct {
	Summary     string   `json:"summary"`
	Objectives  []string `json:"objectives"`
	Constraints []string `json:"constraints"`
}

// CodebaseInfo contains discovered codebase structure
type CodebaseInfo struct {
	RootDir     string   `json:"root_dir"`
	Language    string   `json:"language"`
	BuildSystem string   `json:"build_system"`
	KeyFiles    []string `json:"key_files"`
	Patterns    []string `json:"patterns"`
}

// Discovery represents a finding during exploration
type Discovery struct {
	Iteration   int       `json:"iteration"`
	Phase       Phase     `json:"phase"`
	Type        string    `json:"type"`
	Path        string    `json:"path"`
	Description string    `json:"description"`
	Relevance   string    `json:"relevance"`
	Timestamp   time.Time `json:"timestamp"`
}

// FocusSet represents the current working context
type FocusSet struct {
	Files     []string `json:"files"`
	Functions []string `json:"functions"`
	Tests     []string `json:"tests"`
}

// HistoryEntry represents a single conversation entry in history
type HistoryEntry struct {
	Iteration int       `json:"iteration"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Phase     Phase     `json:"phase"`
	Timestamp time.Time `json:"timestamp"`
}

// SearchQuery represents a recorded search operation
type SearchQuery struct {
	Iteration int       `json:"iteration"`
	Query     string    `json:"query"`
	Type      string    `json:"type"`
	Results   []string  `json:"results"`
	Timestamp time.Time `json:"timestamp"`
}

// NarrowedSet represents a subset of context for focused work
type NarrowedSet struct {
	Iteration   int       `json:"iteration"`
	Description string    `json:"description"`
	Files       []string  `json:"files"`
	Functions   []string  `json:"functions"`
	Rationale   string    `json:"rationale"`
	Timestamp   time.Time `json:"timestamp"`
}

// PhaseDisplayName returns a human-readable name for the phase
func PhaseDisplayName(phase Phase) string {
	switch phase {
	case PhasePlan:
		return "Planning"
	case PhaseSearch:
		return "Searching"
	case PhaseNarrow:
		return "Narrowing"
	case PhaseAct:
		return "Implementing"
	case PhaseVerify:
		return "Verifying"
	default:
		return string(phase)
	}
}

// =====================================
// StateManager
// =====================================

// StateManager handles RLM state persistence
type StateManager struct {
	baseDir string
}

// NewStateManager creates a new StateManager
func NewStateManager(baseDir string) *StateManager {
	return &StateManager{baseDir: baseDir}
}

// InitSession initializes a new RLM session, cleaning up any previous state
func (sm *StateManager) InitSession() (*SessionState, error) {
	// Clean up previous state
	if err := os.RemoveAll(sm.baseDir); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to clean previous state: %w", err)
	}

	// Create state directories
	dirs := []string{
		sm.baseDir,
		filepath.Join(sm.baseDir, "search"),
		filepath.Join(sm.baseDir, "narrow"),
		filepath.Join(sm.baseDir, "results"),
		filepath.Join(sm.baseDir, "verification"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create state directory %s: %w", dir, err)
		}
	}

	// Create initial session state
	now := time.Now()
	state := &SessionState{
		SessionID:   uuid.New().String(),
		Iteration:   0,
		Depth:       0,
		Phase:       PhasePlan,
		StartedAt:   now,
		LastUpdated: now,
	}

	if err := sm.SaveSession(state); err != nil {
		return nil, err
	}

	// Create initial context manifest
	context := &ContextManifest{
		Task:        TaskInfo{},
		Codebase:    CodebaseInfo{},
		Discoveries: []Discovery{},
		Focus:       FocusSet{},
		LastUpdated: now,
	}
	if err := sm.saveContext(context); err != nil {
		return nil, err
	}

	return state, nil
}

// LoadSession loads the current session state
func (sm *StateManager) LoadSession() (*SessionState, error) {
	path := filepath.Join(sm.baseDir, "session.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read session state: %w", err)
	}

	var state SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse session state: %w", err)
	}

	return &state, nil
}

// SaveSession saves the session state
func (sm *StateManager) SaveSession(state *SessionState) error {
	state.LastUpdated = time.Now()
	path := filepath.Join(sm.baseDir, "session.json")

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session state: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write session state: %w", err)
	}

	return nil
}

// GetContext loads the context manifest
func (sm *StateManager) GetContext() (*ContextManifest, error) {
	path := filepath.Join(sm.baseDir, "context.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read context manifest: %w", err)
	}

	var context ContextManifest
	if err := json.Unmarshal(data, &context); err != nil {
		return nil, fmt.Errorf("failed to parse context manifest: %w", err)
	}

	return &context, nil
}

// UpdateContext updates the context manifest
func (sm *StateManager) UpdateContext(context *ContextManifest) error {
	return sm.saveContext(context)
}

func (sm *StateManager) saveContext(context *ContextManifest) error {
	context.LastUpdated = time.Now()
	path := filepath.Join(sm.baseDir, "context.json")

	data, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context manifest: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write context manifest: %w", err)
	}

	return nil
}

// AppendHistory appends a history entry to the history file
func (sm *StateManager) AppendHistory(entry HistoryEntry) error {
	entry.Timestamp = time.Now()
	path := filepath.Join(sm.baseDir, "history.jsonl")

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal history entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write history entry: %w", err)
	}

	return nil
}

// GetRecentHistory returns the last n history entries
func (sm *StateManager) GetRecentHistory(n int) ([]HistoryEntry, error) {
	path := filepath.Join(sm.baseDir, "history.jsonl")

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []HistoryEntry{}, nil
		}
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry HistoryEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read history file: %w", err)
	}

	if len(entries) <= n {
		return entries, nil
	}
	return entries[len(entries)-n:], nil
}

// StoreVerification stores a verification report
func (sm *StateManager) StoreVerification(report VerificationReport) error {
	report.Timestamp = time.Now()
	filename := fmt.Sprintf("verify_%04d_%d.json", report.Iteration, time.Now().UnixMilli())
	path := filepath.Join(sm.baseDir, "verification", filename)

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal verification report: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write verification report: %w", err)
	}

	return nil
}

// GetLatestVerification returns the most recent verification report
func (sm *StateManager) GetLatestVerification() (*VerificationReport, error) {
	dir := filepath.Join(sm.baseDir, "verification")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read verification directory: %w", err)
	}

	if len(entries) == 0 {
		return nil, nil
	}

	var latest string
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() > latest {
			latest = entry.Name()
		}
	}

	if latest == "" {
		return nil, nil
	}

	data, err := os.ReadFile(filepath.Join(dir, latest))
	if err != nil {
		return nil, fmt.Errorf("failed to read verification report: %w", err)
	}

	var report VerificationReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse verification report: %w", err)
	}

	return &report, nil
}

// UpdateIteration changes the session phase
func (sm *StateManager) UpdateIteration(phase Phase) error {
	state, err := sm.LoadSession()
	if err != nil {
		return err
	}

	state.Phase = phase
	return sm.SaveSession(state)
}

// =====================================
// PhaseRouter
// =====================================

// PhaseRouter handles phase inference and guidance generation
type PhaseRouter struct {
	state *StateManager
}

// NewPhaseRouter creates a new PhaseRouter
func NewPhaseRouter(state *StateManager) *PhaseRouter {
	return &PhaseRouter{state: state}
}

// InferPhase determines the current phase based on session state
func (pr *PhaseRouter) InferPhase() (Phase, error) {
	session, err := pr.state.LoadSession()
	if err != nil {
		return PhasePlan, err
	}

	context, err := pr.state.GetContext()
	if err != nil {
		return PhasePlan, err
	}

	// First iteration always starts with PLAN
	if session.Iteration <= 1 {
		return PhasePlan, nil
	}

	// If no discoveries yet, still in SEARCH/PLAN phase
	if len(context.Discoveries) == 0 {
		return PhaseSearch, nil
	}

	// If no focus files yet, need to NARROW
	if len(context.Focus.Files) == 0 {
		return PhaseNarrow, nil
	}

	// Check last verification result
	lastVerify, _ := pr.state.GetLatestVerification()
	if lastVerify != nil && !lastVerify.Passed {
		return PhaseAct, nil
	}

	// Default progression based on previous phase
	switch session.Phase {
	case PhasePlan:
		return PhaseSearch, nil
	case PhaseSearch:
		return PhaseNarrow, nil
	case PhaseNarrow:
		return PhaseAct, nil
	case PhaseAct:
		return PhaseVerify, nil
	case PhaseVerify:
		return PhaseSearch, nil
	default:
		return PhasePlan, nil
	}
}

// GetPhaseGuidance returns phase-specific instructions for the agent
func (pr *PhaseRouter) GetPhaseGuidance(phase Phase) string {
	switch phase {
	case PhasePlan:
		return planPhaseGuidance
	case PhaseSearch:
		return searchPhaseGuidance
	case PhaseNarrow:
		return narrowPhaseGuidance
	case PhaseAct:
		return actPhaseGuidance
	case PhaseVerify:
		return verifyPhaseGuidance
	default:
		return planPhaseGuidance
	}
}

// Phase guidance templates
const planPhaseGuidance = `## Phase: PLAN

**Objective:** Understand the task and create an implementation approach.

**Actions:**
1. Read and analyze the task from PROMPT.md
2. Identify key objectives and constraints
3. Create high-level implementation steps
4. Update context.json with task summary

**State File Updates:**

Update .ralph/state/context.json with task info:
` + "```json" + `
{
  "task": {
    "summary": "Brief description of the task",
    "objectives": ["objective1", "objective2"],
    "constraints": ["constraint1"]
  },
  "codebase": {
    "root_dir": ".",
    "language": "go",
    "build_system": "taskfile"
  },
  "discoveries": [],
  "focus": {"files": [], "functions": [], "tests": []},
  "last_updated": "2024-01-23T10:30:00Z"
}
` + "```" + `

**Exit criteria:**
- Task objectives are clear
- context.json updated with task summary
- Ready to search for relevant code

**Output:** Signal phase completion with:
` + "`" + `<rlm:phase>SEARCH</rlm:phase>` + "`"

const searchPhaseGuidance = `## Phase: SEARCH

**Objective:** Explore the codebase to find relevant code and patterns.

**Actions:**
1. Use grep/glob to find files related to the task
2. Read key files to understand existing patterns
3. Identify dependencies and related code
4. Record discoveries in state files

**Exit criteria:**
- Relevant files and functions identified
- Search results saved to .ralph/state/search/
- Discoveries added to context.json
- Ready to narrow focus

**Output:** Signal phase completion with:
` + "`" + `<rlm:phase>NARROW</rlm:phase>` + "`"

const narrowPhaseGuidance = `## Phase: NARROW

**Objective:** Focus on specific files and functions for modification.

**Actions:**
1. Review discoveries from context.json
2. Select specific files to modify
3. Identify exact functions/locations for changes
4. Save narrowed set to state

**Exit criteria:**
- Specific files and functions identified
- Narrowed set saved to .ralph/state/narrow/
- Focus updated in context.json
- Ready to implement

**Output:** Signal phase completion with:
` + "`" + `<rlm:phase>ACT</rlm:phase>` + "`"

const actPhaseGuidance = `## Phase: ACT

**CRITICAL:** You are in the IMPLEMENTATION phase. DO NOT plan. DO NOT analyze. IMPLEMENT NOW.

The planning phase is COMPLETE. Your focus set has been identified.

**Your ONLY job in this phase:** Make the actual code changes.

**DO NOT:**
- Create more plans or task lists
- Analyze or explore the codebase further
- Discuss what you "would" do - DO IT
- Ask questions or seek clarification
- Say you need to "plan" or "continue planning"

**DO:**
- Use the Edit tool to modify files
- Use the Write tool to create new files
- Make real, concrete code changes NOW

**Exit criteria:**
- Changes implemented
- Code follows existing patterns
- Results recorded in .ralph/state/results/
- Ready for verification

**Output:** Signal phase completion with:
` + "`" + `<rlm:phase>VERIFY</rlm:phase>` + "`"

const verifyPhaseGuidance = `## Phase: VERIFY

**Objective:** Validate changes work correctly.

**Actions:**
1. Run relevant build commands
2. Run relevant test commands
3. Check for any regressions
4. Fix issues if found

**If verification passes:**
1. Record success in verification file
2. Signal verification success with: ` + "`" + `<rlm:verified>true</rlm:verified>` + "`" + `
3. Commit your changes
4. If all tasks complete: ` + "`" + `<promise>COMPLETE</promise>` + "`" + `
5. Otherwise signal next search: ` + "`" + `<rlm:phase>SEARCH</rlm:phase>` + "`" + `

**If verification fails:**
1. Record failure in verification file
2. Analyze failure
3. Return to ACT phase to fix: ` + "`" + `<rlm:phase>ACT</rlm:phase>` + "`" + `
`
