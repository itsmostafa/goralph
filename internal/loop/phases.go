package loop

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
		// Last verification failed, need to fix and re-verify
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
		// After verify, loop back to search for next task or complete
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

**State File Updates:**

After searching, create a search result file:
` + "```" + `
.ralph/state/search/search_NNNN_TIMESTAMP.json
` + "```" + `

Schema (create this file using the Write tool):
` + "```json" + `
{
  "iteration": 1,
  "query": "description of what you searched for",
  "type": "grep|glob|semantic",
  "results": ["file1.go", "file2.go"],
  "timestamp": "2024-01-23T10:30:00Z"
}
` + "```" + `

Also update context.json with discoveries (read existing file first, merge in new discoveries, then write back):
` + "```json" + `
{
  "discoveries": [
    {
      "iteration": 1,
      "phase": "SEARCH",
      "type": "file|function|pattern|dependency",
      "path": "path/to/file.go",
      "description": "What you found",
      "relevance": "high|medium|low"
    }
  ]
}
` + "```" + `

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

**State File Updates:**

Create a narrowed set file:
` + "```" + `
.ralph/state/narrow/narrow_NNNN_TIMESTAMP.json
` + "```" + `

Schema (create this file using the Write tool):
` + "```json" + `
{
  "iteration": 2,
  "description": "What this narrowed set covers",
  "files": ["path/to/file1.go", "path/to/file2.go"],
  "functions": ["FunctionA", "FunctionB"],
  "rationale": "Why these files/functions were selected"
}
` + "```" + `

Update context.json focus set (read existing file first, update focus, then write back):
` + "```json" + `
{
  "focus": {
    "files": ["path/to/file1.go"],
    "functions": ["FunctionA"],
    "tests": ["path/to/file_test.go"]
  }
}
` + "```" + `

**Exit criteria:**
- Specific files and functions identified
- Narrowed set saved to .ralph/state/narrow/
- Focus updated in context.json
- Ready to implement

**Output:** Signal phase completion with:
` + "`" + `<rlm:phase>ACT</rlm:phase>` + "`"

const actPhaseGuidance = `## Phase: ACT

**Objective:** Implement the planned changes.

**Actions:**
1. Review focus set from context.json
2. Implement changes in identified files
3. Follow existing code patterns and conventions
4. Write minimal, focused changes

**State File Updates:**

After implementing, record results in a results file:
` + "```" + `
.ralph/state/results/act_NNNN_TIMESTAMP.json
` + "```" + `

Schema (create this file using the Write tool):
` + "```json" + `
{
  "iteration": 3,
  "phase": "ACT",
  "changes": [
    {
      "file": "path/to/file.go",
      "type": "modified|created|deleted",
      "description": "What was changed"
    }
  ],
  "timestamp": "2024-01-23T10:30:00Z"
}
` + "```" + `

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

**State File Updates:**

Record verification results:
` + "```" + `
.ralph/state/verification/verify_NNNN_TIMESTAMP.json
` + "```" + `

Schema (create this file using the Write tool):
` + "```json" + `
{
  "iteration": 3,
  "passed": true,
  "build": {
    "command": "task build",
    "success": true,
    "output": "Build succeeded"
  },
  "tests": {
    "command": "go test ./...",
    "success": true,
    "output": "All tests passed"
  },
  "timestamp": "2024-01-23T10:30:00Z"
}
` + "```" + `

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
