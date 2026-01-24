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

- **Multi-Agent Support** - Choose between Claude Code and OpenAI Codex CLI as your agent provider
- **Session-Scoped Implementation Plans** - Creates timestamped plan files in `.ralph/plans/` for each session
- **Iteration-Aware Task Generation** - When using `-n`/`--max`, the agent breaks work into approximately N tasks
- **Configurable Iteration Limits** - Set maximum iterations with `-n`/`--max` flag or run unlimited
- **Automatic Git Pushes** - Pushes changes to remote after each iteration, auto-creates remote branches
- **Styled Terminal Output** - Simple terminal UI with lipgloss styling, colored status indicators, and boxed summaries
- **Iteration Summaries** - Displays duration, token usage, cost, and status after each iteration
- **JSON Logging** - Saves full agent output to timestamped JSONL files in `.ralph/logs/`
- **Stream JSON Parsing** - Parses streaming JSON output from agents in real-time
- **RLM Mode** - Recursive Language Model support for structured, stateful agent iterations

## RLM Mode

Go Ralph supports RLM (Recursive Language Model) mode, based on the research paper [arXiv:2512.24601](https://arxiv.org/abs/2512.24601). RLM mode uses a structured approach where state is persisted to files rather than relying on context window limits.

Each iteration follows the RLM cycle: **PLAN → SEARCH → NARROW → ACT → VERIFY**

```bash
# Enable RLM mode
goralph run --rlm

# RLM mode with verification (runs build/test before commit)
goralph run --rlm --verify

# Set max recursion depth (default: 3)
goralph run --rlm --max-depth 5
```

When RLM mode is enabled, state is persisted in `.ralph/state/` including session metadata, context manifests, and verification reports.

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

3. Create the required prompt file in your target project:
   ```bash
   mkdir -p .ralph
   touch .ralph/PROMPT.md
   ```

4. Add your prompt to the file created above. This prompt will be used by the agentic loop.

## Usage

### Commands

```bash
# Run the agentic loop (uses Claude by default)
goralph run

# Run with max 20 iterations
goralph run --max 20
goralph run -n 20

# Run without pushing changes after each iteration
goralph run --no-push

# Use OpenAI Codex instead of Claude
goralph run --agent codex

# Combine flags
goralph run -n 10 --no-push --agent codex
```

### Options

| Flag | Short | Description |
|------|-------|-------------|
| `--max` | `-n` | Maximum number of iterations (0 = unlimited) |
| `--no-push` | | Skip pushing changes after each iteration |
| `--agent` | | Agent provider to use: `claude` (default) or `codex` |
| `--rlm` | | Enable RLM (Recursive Language Model) mode |
| `--verify` | | Run build/test verification before commit |
| `--max-depth` | | Maximum recursion depth for RLM (default: 3) |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GORALPH_AGENT` | Default agent provider (`claude` or `codex`). Overridden by `--agent` flag. |

### Required Files

Before running, ensure this prompt file exists in your `.ralph/` directory:

- `.ralph/PROMPT.md` - Prompt file for the agentic loop
- `.ralph/plans/` - Directory for session-scoped implementation plans (auto-created)

### Warning

**Agents run in unattended mode with full system access:**

- **Claude Code** runs with `--dangerously-skip-permissions` mode enabled
- **Codex CLI** runs with `--dangerously-bypass-approvals-and-sandbox` mode enabled

These modes allow the agents to execute commands without confirmation prompts, which is required for unattended agentic loops.

**It is strongly recommended to run Go Ralph (or any fully autonomous AI agent) inside a sandbox or isolated environment.**

Running autonomous agents with unrestricted system access carries inherent risks. Sandboxing limits the potential impact of unintended actions.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
