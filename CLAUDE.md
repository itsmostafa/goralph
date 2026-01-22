# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go Ralph is an implementation of the Ralph Wiggum Technique - an agentic loop pattern that runs Claude Code iteratively with automatic git pushes between iterations. Reference: https://github.com/ghuntley/how-to-ralph-wiggum

## Commands

### Task runner commands
```bash
task build             # Build the goralph binary
task run               # Run main.go
task install           # Install goralph to ~/.local/bin/
```

### CLI usage
```bash
goralph build          # Run agentic loop in build mode
goralph plan           # Run agentic loop in plan mode
goralph build -n 5     # Run with max 5 iterations (tasks broken into ~5 pieces)
goralph plan --max 10  # Run with max 10 iterations (tasks broken into ~10 pieces)
goralph build --no-push  # Run without pushing changes after each iteration
```

## How It Works

1. **Reads prompt file** from `.ralph/PROMPT_build.md` or `.ralph/PROMPT_plan.md`
2. **Appends implementation plan** from `.ralph/IMPLEMENTATION_PLAN.md` with instructions
3. **Runs Claude Code** with the combined prompt, streaming output in real-time
4. **Claude completes one task**, updates the implementation plan, and commits
5. **Pushes changes** to the remote branch (unless `--no-push` is set)
6. **Loops** until max iterations reached or all tasks complete

### Iteration-Aware Task Generation

When using `--max`/`-n` flag:
- Claude is instructed to break work into approximately N tasks (one per iteration)
- Without the flag, Claude creates a comprehensive task list

## Project Structure

- `cmd/` - Cobra CLI commands (root, build, plan)
- `internal/loop/` - Core loop logic
  - `loop.go` - Main loop execution and Claude iteration
  - `output.go` - Terminal output formatting with lipgloss
  - `types.go` - Message types for JSON stream parsing
- `internal/version/` - Version info (populated via ldflags at build time)
- `.ralph/` - Prompt files and implementation plan
- `.ralph/logs/` - Timestamped JSONL logs of each Claude session

## Required Files

- `.ralph/PROMPT_build.md` - Build mode prompt for the agentic loop
- `.ralph/PROMPT_plan.md` - Plan mode prompt for the agentic loop
- `.ralph/IMPLEMENTATION_PLAN.md` - Task tracking file (auto-created if missing)
- `taskfile.yml` - Task runner configuration

## Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
