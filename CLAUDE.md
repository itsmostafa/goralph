# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go Ralph is an implementation of the Ralph Wiggum Technique - an agentic loop pattern that runs Claude Code iteratively with automatic git pushes between iterations. Reference: https://github.com/ghuntley/how-to-ralph-wiggum

## Commands

### Run the agentic loop
```bash
./loop.sh              # Build mode, unlimited iterations
./loop.sh 20           # Build mode, max 20 iterations
./loop.sh plan         # Plan mode, unlimited iterations
./loop.sh plan 5       # Plan mode, max 5 iterations
```

### Run the Go application
```bash
task run               # Run main.go
```

## Architecture

The loop script (`loop.sh`) orchestrates Claude Code sessions:
1. Reads from `.ralph/PROMPT_build.md` (build mode) or `.ralph/PROMPT_plan.md` (plan mode)
2. Runs Claude in headless mode with `--dangerously-skip-permissions`
3. Pushes all changes to the current branch after each iteration
4. Repeats until max iterations reached or manually stopped

## Required Files

- `.ralph/PROMPT_build.md` - Build mode prompt for the agentic loop
- `.ralph/PROMPT_plan.md` - Plan mode prompt for the agentic loop
- `taskfile.yml` - Task runner configuration
