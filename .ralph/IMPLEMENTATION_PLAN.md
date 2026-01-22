# Implementation Plan

## Overview

Add support for OpenAI Codex CLI as an alternative to Claude Code CLI. Users can choose which CLI to use via a `--cli` flag or environment variable.

## Key Differences Between CLIs

| Feature | Claude Code | Codex CLI |
|---------|-------------|-----------|
| Command | `claude` | `codex exec` |
| Piped input | `-p` flag + stdin | `-` argument reads from stdin |
| Skip permissions | `--dangerously-skip-permissions` | `--dangerously-bypass-approvals-and-sandbox` (alias: `--yolo`) |
| JSON output | `--output-format=stream-json` | `--json` (newline-delimited JSONL) |
| Verbose | `--verbose` | (streams progress to stderr by default) |

### Codex CLI JSON Event Types (from [OpenAI docs](https://developers.openai.com/codex/noninteractive/))

When using `--json`, Codex outputs newline-delimited JSON with these event types:
- `thread.started` - Session initialization
- `turn.started` / `turn.completed` / `turn.failed` - Turn lifecycle
- `item.*` - Agent messages, reasoning, command executions, file changes, MCP tool calls, web searches, plan updates
- `error` - Error events

## Tasks

- [ ] **Task 1: Add CLI provider type and configuration**
  - Add `CLIProvider` type (`claude`, `codex`) in `internal/loop/types.go`
  - Add `CLI` field to `Config` struct in `internal/loop/loop.go`
  - Add `--cli` flag to both `cmd/build.go` and `cmd/plan.go` (default: `claude`)
  - Support `GORALPH_CLI` environment variable as fallback using `cobra` flag binding

- [ ] **Task 2: Create provider-specific command builders**
  - Create `internal/loop/provider.go` with `Provider` interface:
    ```go
    type Provider interface {
        Name() string
        BuildCommand(prompt []byte) (*exec.Cmd, error)
    }
    ```
  - Implement `ClaudeProvider` that builds: `claude -p --dangerously-skip-permissions --output-format=stream-json --verbose`
  - Implement `CodexProvider` that builds: `codex exec --json --dangerously-bypass-approvals-and-sandbox -`
  - Refactor `runClaudeIteration` → `runIteration` to accept a `Provider`
  - Add `GetProvider(cfg Config) Provider` factory function

- [ ] **Task 3: Adapt output parsing for Codex JSON format**
  - Add Codex-specific message types to `types.go`:
    - `CodexEvent` with `type` field (`thread.started`, `turn.*`, `item.*`, `error`)
    - `CodexItemEvent` for item-specific payloads
  - Create `parseCodexOutput` function that:
    - Handles `turn.started`/`turn.completed` for iteration tracking
    - Extracts text content from `item.*` events
    - Handles `error` events gracefully
  - Update `parseOutput` to dispatch to correct parser based on provider
  - Update `FormatHeader` in `output.go` to display which CLI is being used

## Completed

<!-- Completed tasks will be moved here as: - [x] Task description -->
