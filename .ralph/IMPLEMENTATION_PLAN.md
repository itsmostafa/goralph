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
  - Validate CLI provider value and return error for unknown providers

- [ ] **Task 2: Create provider-specific command builders**
  - Create `internal/loop/provider.go` with `Provider` interface:
    ```go
    type Provider interface {
        Name() string
        BuildCommand(prompt []byte) (*exec.Cmd, error)
        ParseOutput(r io.Reader, w io.Writer, logFile io.Writer) (*ResultMessage, error)
    }
    ```
  - Implement `ClaudeProvider`:
    - `BuildCommand`: `claude -p --dangerously-skip-permissions --output-format=stream-json --verbose`
    - `ParseOutput`: existing `parseClaudeOutput` logic
  - Implement `CodexProvider`:
    - `BuildCommand`: `codex exec --json --dangerously-bypass-approvals-and-sandbox -`
    - `ParseOutput`: new Codex-specific parser (see Task 3)
  - Add `NewProvider(cli CLIProvider) (Provider, error)` factory function
  - Refactor `runClaudeIteration` → `runIteration` to use `Provider` interface
  - Update `Run()` to instantiate provider once at start

- [ ] **Task 3: Implement Codex output parsing**
  - Add Codex-specific message types to `types.go`:
    ```go
    type CodexEvent struct {
        Type string `json:"type"`  // thread.started, turn.*, item.*, error
    }
    type CodexItemEvent struct {
        Type    string `json:"type"`
        Content any    `json:"content,omitempty"`
        // Additional fields based on item type
    }
    ```
  - Implement `CodexProvider.ParseOutput` that:
    - Handles `turn.started`/`turn.completed`/`turn.failed` for turn tracking
    - Extracts text from `item.message` and `item.reasoning` events
    - Shows tool invocations from `item.command_*` and `item.mcp_tool_*` events
    - Handles `error` events gracefully
    - Returns a `ResultMessage` equivalent for summary display
  - Update `FormatHeader` in `output.go` to display CLI provider name
  - Add `FormatIterationSummary` variant or adapter for Codex stats

## Implementation Notes

### File Changes Summary

| File | Changes |
|------|---------|
| `internal/loop/types.go` | Add `CLIProvider` type, Codex event types |
| `internal/loop/loop.go` | Add `CLI` to Config, refactor to use Provider |
| `internal/loop/provider.go` | New file with Provider interface and implementations |
| `internal/loop/output.go` | Update `FormatHeader` to show CLI |
| `cmd/build.go` | Add `--cli` flag |
| `cmd/plan.go` | Add `--cli` flag |

### Testing Strategy

- Test with both `claude` and `codex` CLI values
- Verify JSON parsing with sample output from each CLI
- Test environment variable fallback (`GORALPH_CLI`)
- Ensure graceful error handling for missing CLI executables

## Completed

<!-- Completed tasks will be moved here as: - [x] Task description -->
