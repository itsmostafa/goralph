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
goralph run                 # Run the agentic loop (uses Claude by default)
goralph run -n 5            # Run with max 5 iterations (tasks broken into ~5 pieces)
goralph run --max 10        # Run with max 10 iterations (tasks broken into ~10 pieces)
goralph run --no-push       # Run without committing or pushing changes
goralph run --agent codex   # Use OpenAI Codex instead of Claude
goralph run --mode=rlm      # Enable RLM (Recursive Language Model) mode
goralph run --verify        # Run build/test verification before commit
goralph run --mode=rlm --verify  # RLM mode with verification
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

RLM (Recursive Language Model) mode implements the academic RLM approach (arXiv:2512.24601) with a REPL environment for programmatic context manipulation. The LLM can write and execute code to explore context and make recursive LLM calls.

### REPL Environment

In RLM mode, agents have access to a Tengo scripting environment. They write code in ` + "```repl" + ` blocks:

```repl
// Access the task context
println(slice(context, 0, 200))

// Search for patterns
matches := find_all("TODO:.*", context)
for m in matches {
    println(m)
}

// Make recursive LLM calls
summary := llm_query("Summarize: " + slice(context, 0, 1000))
println(summary)
```

### Built-in REPL Functions

- `context` - The full task/document as a string
- `slice(s, start, end)` - Get substring
- `len(s)` - Length of string or array
- `contains(s, substr)` - Check if string contains substring
- `split(s, sep)` / `join(arr, sep)` - Split/join strings
- `find_all(pattern, text)` - Regex search
- `replace(s, old, new)` - String replacement
- `llm_query(prompt)` - Recursive LLM call
- `llm_batch(prompts)` - Batched LLM calls
- `println(args...)` / `print(args...)` - Output

### RLM Completion Markers

Agents signal completion using:
- `FINAL(answer text)` - Direct final answer
- `FINAL_VAR(variable_name)` - Return value of a REPL variable
- `<promise>COMPLETE</promise>` - Legacy completion marker

### RLM State Directory

Session state is persisted in `.ralph/state/`:
- `session.json` - Session metadata (iteration, depth, LLM call count)
- `verification/` - Verification reports

### Recursion Depth

The `--max-depth` flag (default: 3) limits how deep recursive `llm_query()` calls can go to prevent runaway token usage.

### Verification

When `--verify` is enabled:
- Project type is auto-detected (Go, Node.js, Rust, Python)
- Build and test commands run before commit
- Failed verification skips push and continues to next iteration

## Project Structure

- `cmd/` - Cobra CLI commands (root, run)
- `internal/loop/` - Core loop logic
  - `loop.go` - Main loop execution and agent iteration
  - `mode.go` - Mode type, ModeRunner interface, and validation
  - `mode_ralph.go` - Ralph mode runner and prompt building
  - `mode_rlm.go` - RLM mode runner with REPL environment
  - `rlm_types.go` - RLM-specific types (REPLResult, LLMCallRecord, RLMSession)
  - `rlm_repl.go` - REPLExecutor interface
  - `rlm_go_repl.go` - GoREPL implementation using Tengo scripting
  - `rlm_bridge.go` - LMBridge for routing recursive LLM calls
  - `rlm_parser.go` - Parser for REPL code blocks and FINAL markers
  - `rlm_prompts.go` - System prompt templates for RLM mode
  - `providers.go` - Agent provider implementations (Claude, Codex) with Query method
  - `types.go` - Message types, Config, and shared constants
  - `verify.go` - Verification runner for build/test checks
  - `output.go` - Terminal output formatting with lipgloss
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
- `github.com/d5/tengo/v2` - Tengo scripting language for RLM REPL
