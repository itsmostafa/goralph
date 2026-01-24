# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go Ralph is an implementation of the Ralph Wiggum Technique - an agentic loop pattern that runs AI coding agents (Claude Code or OpenAI Codex) iteratively with automatic git pushes between iterations. Reference: https://github.com/ghuntley/how-to-ralph-wiggum

## Commands

### Task runner commands
```bash
task build             # Build the goralph binary
task run               # Run main.go
task install           # Install goralph to ~/.local/bin/
```

### CLI usage
```bash
goralph run              # Run the agentic loop (uses Claude by default)
goralph run -n 5         # Run with max 5 iterations (tasks broken into ~5 pieces)
goralph run --max 10     # Run with max 10 iterations (tasks broken into ~10 pieces)
goralph run --no-push    # Run without pushing changes after each iteration
goralph run --agent codex  # Use OpenAI Codex instead of Claude
goralph run --rlm        # Enable RLM (Recursive Language Model) mode
goralph run --verify     # Run build/test verification before commit
goralph run --rlm --verify  # RLM mode with verification
goralph run --max-depth 5   # Set max recursion depth for RLM (default: 3)
```

### Environment variables
```bash
GORALPH_AGENT=codex  # Set default agent provider (claude or codex)
```

## How It Works

1. **Reads prompt file** from `.ralph/PROMPT.md`
2. **Creates session-scoped plan** in `.ralph/plans/implementation_plan_{timestamp}.md`
3. **Runs the selected agent** (Claude Code or Codex) with the combined prompt, streaming output in real-time
4. **Agent completes one task**, updates the implementation plan, and commits
5. **Pushes changes** to the remote branch (unless `--no-push` is set)
6. **Loops** until max iterations reached, all tasks complete, or agent signals completion

### Completion Promise

Agents can signal that all tasks are complete by outputting the exact string:
```
<promise>COMPLETE</promise>
```

When detected, the loop exits gracefully with a "Session Complete" message instead of continuing to the next iteration. This saves tokens by avoiding unnecessary iterations when work is done.

### Iteration-Aware Task Generation

When using `--max`/`-n` flag:
- The agent is instructed to break work into approximately N tasks (one per iteration)
- Without the flag, the agent creates a comprehensive task list

## RLM Mode

RLM (Recursive Language Model) mode uses a structured approach where state is persisted to files rather than relying on context window. Each iteration follows the RLM cycle: PLAN → SEARCH → NARROW → ACT → VERIFY.

### RLM State Directory

When RLM mode is enabled, state is persisted in `.ralph/state/`:
- `session.json` - Session metadata (iteration, depth, phase)
- `context.json` - Context manifest (discoveries, key files, focus)
- `history.jsonl` - Conversation history across iterations
- `search/` - Search operation results
- `narrow/` - Narrowed context subsets
- `verification/` - Verification reports

### RLM Markers

Agents signal state transitions using markers:
- `<rlm:phase>PHASE</rlm:phase>` - Signal phase transition (PLAN, SEARCH, NARROW, ACT, VERIFY)
- `<rlm:verified>true</rlm:verified>` - Signal verification passed
- `<promise>COMPLETE</promise>` - Signal all tasks complete

### Verification

When `--verify` is enabled:
- Project type is auto-detected (Go, Node.js, Rust, Python)
- Build and test commands run before commit
- Failed verification skips push and continues to next iteration

## Project Structure

- `cmd/` - Cobra CLI commands (root, run)
- `internal/loop/` - Core loop logic
  - `loop.go` - Main loop execution and agent iteration
  - `providers.go` - Agent provider implementations (Claude, Codex)
  - `types.go` - Message types and agent provider constants
  - `rlm_types.go` - RLM-specific type definitions
  - `state.go` - StateManager for RLM state persistence
  - `phases.go` - Phase routing and inference logic
  - `verify.go` - Verification runner for build/test checks
  - `output.go` - Terminal output formatting with lipgloss
  - `prompt.go` - Prompt file reading and construction
  - `git.go` - Git operations (push, branch management)
- `internal/version/` - Version info (populated via ldflags at build time)
- `.ralph/` - Prompt files and session data
- `.ralph/plans/` - Session-scoped implementation plans (timestamped)
- `.ralph/logs/` - Timestamped JSONL logs of each agent session
- `.ralph/state/` - RLM state files (when RLM mode is enabled)

## Required Files

- `.ralph/PROMPT.md` - Prompt file for the agentic loop
- `.ralph/plans/` - Directory for session-scoped implementation plans (auto-created)
- `taskfile.yml` - Task runner configuration

## Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/google/uuid` - UUID generation for RLM sessions
