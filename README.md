

# Go Ralph - Ralph Wiggum Technique in Go

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Claude](https://img.shields.io/badge/Claude-D97757?logo=claude&logoColor=fff)](https://claude.ai/code)

<p align="center">
  <img src="assets/img/goralph-logo.png" alt="Go Ralph Logo" width="200">
</p>

An agentic loop that runs Claude Code iteratively with automatic git pushes. Letting your AI agents build while you sleep.
This project is intentionally built to be simple and easy to use as intended by the creator of Ralph, Geoffrey Huntley.
Reference: [Ralph Wiggum Technique](https://github.com/ghuntley/how-to-ralph-wiggum)

## Requirements

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code/getting-started)
- [Go 1.25](https://go.dev/doc/install)
- [Task](https://github.com/go-task/task) (optional) - for running task commands like `task run`

## Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/goralph.git
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
