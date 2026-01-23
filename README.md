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

## Repository Overview

Go Ralph is a Go CLI that runs the Ralph Wiggum agentic loop (Claude Code or OpenAI Codex) with optional iteration limits and automatic git pushes. It reads prompts from `.ralph/`, streams agent output, writes JSONL logs, and updates an implementation plan between iterations.

Key locations:
- `cmd/` - CLI commands and flag parsing (build/plan/root).
- `internal/loop/` - Core loop execution, prompt handling, provider integration, logging, and git push logic.
- `internal/version/` - Version metadata for `goralph --version`.
- `assets/` - Project logo and release assets.
- `main.go` - CLI entrypoint.

## Features

- **Multi-Agent Support** - Choose between Claude Code and OpenAI Codex CLI as your agent provider
- **Build and Plan Modes** - Two execution modes with dedicated prompt files for different workflows
- **Implementation Plan Tracking** - Auto-manages `.ralph/IMPLEMENTATION_PLAN.md` for task tracking across iterations
- **Iteration-Aware Task Generation** - When using `-n`/`--max`, the agent breaks work into approximately N tasks
- **Configurable Iteration Limits** - Set maximum iterations with `-n`/`--max` flag or run unlimited
- **Automatic Git Pushes** - Pushes changes to remote after each iteration, auto-creates remote branches
- **Styled Terminal Output** - Simple terminal UI with lipgloss styling, colored status indicators, and boxed summaries
- **Iteration Summaries** - Displays duration, token usage, cost, and status after each iteration
- **JSON Logging** - Saves full agent output to timestamped JSONL files in `.ralph/logs/`
- **Stream JSON Parsing** - Parses streaming JSON output from agents in real-time

## Requirements

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code/getting-started) or [OpenAI Codex CLI](https://github.com/openai/codex) - at least one agent provider
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
# Run the agentic loop in build mode (uses Claude by default)
goralph build

# Run in build mode with max 20 iterations
goralph build --max 20
goralph build -n 20

# Run without pushing changes after each iteration
goralph build --no-push

# Run the agentic loop in plan mode
goralph plan

# Run in plan mode with max 5 iterations
goralph plan --max 5
goralph plan -n 5

# Use OpenAI Codex instead of Claude
goralph build --agent codex
goralph plan --agent codex

# Combine flags
goralph build -n 10 --no-push --agent codex
```

### Options

| Flag | Short | Description |
|------|-------|-------------|
| `--max` | `-n` | Maximum number of iterations (0 = unlimited) |
| `--no-push` | | Skip pushing changes after each iteration |
| `--agent` | | Agent provider to use: `claude` (default) or `codex` |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GORALPH_AGENT` | Default agent provider (`claude` or `codex`). Overridden by `--agent` flag. |

### Required Files

Before running, ensure these prompt files exist in your `.ralph/` directory:

- `.ralph/PROMPT_build.md` - Prompt used for build mode
- `.ralph/PROMPT_plan.md` - Prompt used for plan mode
- `.ralph/IMPLEMENTATION_PLAN.md` - Task tracking file (auto-created if missing)

### Warning

**Agents run in unattended mode with full system access:**

- **Claude Code** runs with `--dangerously-skip-permissions` mode enabled
- **Codex CLI** runs with `--dangerously-bypass-approvals-and-sandbox` mode enabled

These modes allow the agents to execute commands without confirmation prompts, which is required for unattended agentic loops.

**It is strongly recommended to run Go Ralph (or any fully autonomous AI agent) inside a sandbox or isolated environment.**

Running autonomous agents with unrestricted system access carries inherent risks. Sandboxing limits the potential impact of unintended actions.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
