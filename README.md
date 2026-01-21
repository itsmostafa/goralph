# Go Ralph - Ralph Wiggum Technique in Go

Reference, https://github.com/ghuntley/how-to-ralph-wiggum, to understand how to implement the Ralph Wiggum Technique

## Requirements

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code/getting-started)
- [Go 1.25](https://go.dev/doc/install)

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
