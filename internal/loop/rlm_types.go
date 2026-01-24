package loop

import "time"

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

// RLMConfig holds RLM-specific configuration
type RLMConfig struct {
	Enabled      bool
	MaxDepth     int
	CurrentDepth int
	Verify       bool
}

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
	// Task information from PROMPT.md
	Task TaskInfo `json:"task"`

	// Codebase information
	Codebase CodebaseInfo `json:"codebase"`

	// Discoveries made during exploration
	Discoveries []Discovery `json:"discoveries"`

	// Current focus - files/functions being worked on
	Focus FocusSet `json:"focus"`

	// Last updated timestamp
	LastUpdated time.Time `json:"last_updated"`
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
	Type        string    `json:"type"` // file, function, pattern, dependency
	Path        string    `json:"path"`
	Description string    `json:"description"`
	Relevance   string    `json:"relevance"` // high, medium, low
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
	Role      string    `json:"role"` // system, assistant, user
	Content   string    `json:"content"`
	Phase     Phase     `json:"phase"`
	Timestamp time.Time `json:"timestamp"`
}

// SearchQuery represents a recorded search operation
type SearchQuery struct {
	Iteration int       `json:"iteration"`
	Query     string    `json:"query"`
	Type      string    `json:"type"` // grep, glob, semantic
	Results   []string  `json:"results"`
	Timestamp time.Time `json:"timestamp"`
}

// SearchResults aggregates search results for the session
type SearchResults struct {
	Queries []SearchQuery `json:"queries"`
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

// VerificationReport contains the results of verification checks
type VerificationReport struct {
	Iteration int                 `json:"iteration"`
	Passed    bool                `json:"passed"`
	Checks    []VerificationCheck `json:"checks"`
	Timestamp time.Time           `json:"timestamp"`
}

// VerificationCheck represents a single verification check
type VerificationCheck struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Passed  bool   `json:"passed"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// RLM marker patterns for output detection
const (
	// RLMPhaseMarkerStart is the start of a phase marker
	RLMPhaseMarkerStart = "<rlm:phase>"
	// RLMPhaseMarkerEnd is the end of a phase marker
	RLMPhaseMarkerEnd = "</rlm:phase>"
	// RLMVerifiedMarkerStart is the start of a verified marker
	RLMVerifiedMarkerStart = "<rlm:verified>"
	// RLMVerifiedMarkerEnd is the end of a verified marker
	RLMVerifiedMarkerEnd = "</rlm:verified>"
	// StateDir is the directory for RLM state files
	StateDir = ".ralph/state"
)
