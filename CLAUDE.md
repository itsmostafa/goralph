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
goralph build -n 5     # Run with max 5 iterations
goralph plan --max 10  # Run with max 10 iterations
```

## Project Structure

- `cmd/` - Cobra CLI commands (root, build, plan)
- `internal/loop/` - Core loop logic, output formatting, and types
- `.ralph/` - Prompt files for build and plan modes

## Required Files

- `.ralph/PROMPT_build.md` - Build mode prompt for the agentic loop
- `.ralph/PROMPT_plan.md` - Plan mode prompt for the agentic loop
- `taskfile.yml` - Task runner configuration

## Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
