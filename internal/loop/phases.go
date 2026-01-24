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
4. Update context manifest with task summary

**State updates:**
- Write task summary to context.json
- Record initial analysis in history

**Exit criteria:**
- Task objectives are clear
- Implementation approach is documented
- Ready to search for relevant code

**Output:** Signal phase completion with:
` + "`" + `<rlm:phase>SEARCH</rlm:phase>` + "`"

const searchPhaseGuidance = `## Phase: SEARCH

**Objective:** Explore the codebase to find relevant code and patterns.

**Actions:**
1. Use grep/glob to find files related to the task
2. Read key files to understand existing patterns
3. Identify dependencies and related code
4. Record discoveries for future reference

**State updates:**
- Store search results in .ralph/state/search/
- Add discoveries to context.json
- Update history with findings

**Exit criteria:**
- Relevant files and functions identified
- Understanding of existing patterns
- Ready to narrow focus

**Output:** Signal phase completion with:
` + "`" + `<rlm:phase>NARROW</rlm:phase>` + "`"

const narrowPhaseGuidance = `## Phase: NARROW

**Objective:** Focus on specific files and functions for modification.

**Actions:**
1. Review discoveries from search phase
2. Select specific files to modify
3. Identify exact functions/locations for changes
4. Validate approach against existing patterns

**State updates:**
- Store narrowed set in .ralph/state/narrow/
- Update focus in context.json
- Record rationale in history

**Exit criteria:**
- Specific files and functions identified
- Clear understanding of required changes
- Ready to implement

**Output:** Signal phase completion with:
` + "`" + `<rlm:phase>ACT</rlm:phase>` + "`"

const actPhaseGuidance = `## Phase: ACT

**Objective:** Implement the planned changes.

**Actions:**
1. Review focus set from narrow phase
2. Implement changes in identified files
3. Follow existing code patterns and conventions
4. Write minimal, focused changes

**State updates:**
- Record changes made in history
- Update any affected discoveries

**Exit criteria:**
- Changes implemented
- Code follows existing patterns
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

**State updates:**
- Store verification results in .ralph/state/verification/
- Update history with verification outcome

**If verification passes:**
1. Signal verification success with: ` + "`" + `<rlm:verified>true</rlm:verified>` + "`" + `
2. Commit your changes
3. If all tasks complete: ` + "`" + `<promise>COMPLETE</promise>` + "`" + `
4. Otherwise signal next search: ` + "`" + `<rlm:phase>SEARCH</rlm:phase>` + "`" + `

**If verification fails:**
1. Analyze failure
2. Return to ACT phase to fix: ` + "`" + `<rlm:phase>ACT</rlm:phase>` + "`" + `
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
