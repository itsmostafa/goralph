# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go Ralph is an implementation of the Ralph Wiggum Technique - an agentic loop pattern that runs Claude Code iteratively with automatic git pushes between iterations. Reference: https://github.com/ghuntley/how-to-ralph-wiggum

## Commands

### Run the Go application
```bash
task run               # Run main.go
```

## Required Files

- `.ralph/PROMPT_build.md` - Build mode prompt for the agentic loop
- `.ralph/PROMPT_plan.md` - Plan mode prompt for the agentic loop
- `taskfile.yml` - Task runner configuration
