# Go Ralph
## A Simple CLI for the Ralph Wiggum Technique

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Claude](https://img.shields.io/badge/Claude-D97757?logo=claude&logoColor=fff)](https://claude.ai/code)
[![Releases](https://img.shields.io/github/v/release/itsmostafa/goralph)](https://github.com/itsmostafa/goralph/releases)

<p align="center">
  <img src="assets/img/goralph-logo.png" alt="Go Ralph Logo" width="200">
</p>

An agentic loop that runs Claude Code iteratively with automatic git pushes. Letting your AI agents build while you sleep.
This project is intentionally built to be simple and easy to use as intended by the creator of Ralph, Geoffrey Huntley.
Reference: [Ralph Wiggum Technique](https://github.com/ghuntley/how-to-ralph-wiggum)

## Features

- **Build and Plan Modes** - Two execution modes with dedicated prompt files for different workflows
- **Implementation Plan Tracking** - Auto-manages `.ralph/IMPLEMENTATION_PLAN.md` for task tracking across iterations
- **Iteration-Aware Task Generation** - When using `-n`/`--max`, Claude breaks work into approximately N tasks
- **Configurable Iteration Limits** - Set maximum iterations with `-n`/`--max` flag or run unlimited
- **Automatic Git Pushes** - Pushes changes to remote after each iteration, auto-creates remote branches
- **Styled Terminal Output** - Simple terminal UI with lipgloss styling, colored status indicators, and boxed summaries
- **Iteration Summaries** - Displays duration, token usage, cost, and status after each iteration
- **JSON Logging** - Saves full Claude output to timestamped JSONL files in `.ralph/logs/`
- **Stream JSON Parsing** - Parses Claude's streaming JSON output in real-time

## Requirements

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code/getting-started)
- [Go 1.25](https://go.dev/doc/install)
- [Task](https://github.com/go-task/task) (optional) - for running task commands like `task run`

## Installation

### Option 1: Download Pre-built Binary

Download the latest release for your platform from the [Releases page](https://github.com/itsmostafa/goralph/releases).

### Option 2: Build from Source

1. Clone the repository:
   ```bash
   git clone https://github.com/itsmostafa/goralph.git
   cd goralph
   ```

2. Build and install the binary:
   ```bash
   go build -o goralph .
   go install
   ```

3. Create the required prompt files in your target project:
   ```bash
   mkdir -p .ralph
   touch .ralph/PROMPT_build.md
   touch .ralph/PROMPT_plan.md
   ```

4. Add your prompts to the files created above. These prompts will be used by Claude Code during the agentic loop.

## Usage

### Commands

```bash
# Run the agentic loop in build mode
goralph build

# Run in build mode with max 20 iterations
goralph build --max 20
goralph build -n 20

# Run the agentic loop in plan mode
goralph plan

# Run in plan mode with max 5 iterations
goralph plan --max 5
goralph plan -n 5
```

### Options

| Flag | Short | Description |
|------|-------|-------------|
| `--max` | `-n` | Maximum number of iterations (0 = unlimited) |

### Required Files

Before running, ensure these prompt files exist in your `.ralph/` directory:

- `.ralph/PROMPT_build.md` - Prompt used for build mode
- `.ralph/PROMPT_plan.md` - Prompt used for plan mode
- `.ralph/IMPLEMENTATION_PLAN.md` - Task tracking file (auto-created if missing)
