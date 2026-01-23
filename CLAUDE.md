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
goralph build            # Run agentic loop in build mode (uses Claude by default)
goralph plan             # Run agentic loop in plan mode
goralph build -n 5       # Run with max 5 iterations (tasks broken into ~5 pieces)
goralph plan --max 10    # Run with max 10 iterations (tasks broken into ~10 pieces)
goralph build --no-push  # Run without pushing changes after each iteration
goralph build --agent codex  # Use OpenAI Codex instead of Claude
goralph build --agent rlm    # Use RLM (Recursive Language Model) for large codebase analysis
```

### Environment variables
```bash
GORALPH_AGENT=codex      # Set default agent provider (claude, codex, or rlm)
ANTHROPIC_API_KEY=...    # Required for RLM provider
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

## Project Structure

- `cmd/` - Cobra CLI commands (root, build, plan)
- `internal/loop/` - Core loop logic
  - `loop.go` - Main loop execution and agent iteration
  - `providers.go` - Agent provider implementations (Claude, Codex, RLM)
  - `types.go` - Message types and agent provider constants
  - `output.go` - Terminal output formatting with lipgloss
  - `prompt.go` - Prompt file reading and construction
  - `git.go` - Git operations (push, branch management)
- `internal/rlm/` - Recursive Language Model implementation
  - `types.go` - Core types (Config, Environment, Message types)
  - `runner.go` - RLM session orchestration and LLM API calls
  - `repl.go` - JavaScript REPL executor using goja
  - `parser.go` - FINAL/FINAL_VAR statement parsing
  - `prompts.go` - System prompt builder for RLM sessions
  - `fs.go` - Filesystem module for codebase exploration
- `internal/version/` - Version info (populated via ldflags at build time)
- `.ralph/` - Prompt files and session data
- `.ralph/plans/` - Session-scoped implementation plans (timestamped)
- `.ralph/logs/` - Timestamped JSONL logs of each agent session

## Required Files

- `.ralph/PROMPT.md` - Prompt file for the agentic loop (used by both build and plan modes)
- `.ralph/plans/` - Directory for session-scoped implementation plans (auto-created)
- `taskfile.yml` - Task runner configuration

## Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
- `github.com/dop251/goja` - JavaScript interpreter for RLM REPL

## RLM Provider

The RLM (Recursive Language Model) provider is an alternative to Claude Code and Codex that implements the RLM pattern for handling large codebases. It uses a REPL-based approach where the model explores context programmatically rather than consuming it all in a single call.

### When to Use RLM

- **Large codebases**: When the codebase is too large to fit in a single context window
- **Targeted analysis**: When you need to analyze specific files or patterns rather than entire repositories
- **Reduced token costs**: RLM can be more cost-effective for exploration-heavy tasks

### How RLM Works

1. **Context as variable**: Instead of embedding all code in the prompt, the codebase is available through `fs.*` functions
2. **REPL-based exploration**: The model writes JavaScript code to explore files and analyze code
3. **Iterative refinement**: The model receives execution results and iterates until it has enough information
4. **Recursive sub-calls**: Complex analysis can be delegated to recursive LLM calls with focused context

### RLM Environment

The model has access to:
- `context` - The main document/prompt content
- `query` - The user's question or task
- `fs.list(path)` - List files in directory
- `fs.read(path)` - Read file contents
- `fs.glob(pattern)` - Find files matching glob pattern (e.g., `*.go`, `**/*.ts`)
- `fs.exists(path)` - Check if path exists
- `fs.tree(path, depth)` - Get directory tree structure
- `re.findAll(pattern, text)` - Find regex matches
- `re.search(pattern, text)` - Find first match
- `recursiveLLM(subQuery, subContext)` - Spawn recursive analysis

### RLM Configuration

Default configuration (in `internal/rlm/types.go`):
- Model: `claude-sonnet-4-20250514`
- Max depth: 5 (recursion limit)
- Max iterations: 30 (REPL turns per call)
- Timeout: 120 seconds per LLM call
- Max output chars: 2000 per REPL execution

### Example RLM Session

When you run `goralph build --agent rlm`, the agent might:

1. Explore the project structure: `fs.tree('.', 3)`
2. Find relevant files: `fs.glob('**/*.go')`
3. Read specific files: `let code = fs.read('main.go')`
4. Analyze content: `code.split('\n').filter(l => l.includes('func'))`
5. Signal completion: `FINAL("The main entry point is in cmd/root.go...")`
