# Implementation Plan

## Tasks

(All tasks completed)

## Completed

- [x] **Task 3: Implement Codex output parsing**
  - Added Codex-specific message types to `types.go`:
    - `CodexThreadStartedEvent`, `CodexTurnCompletedEvent`, `CodexUsage`
    - `CodexItem` struct with fields for all item types
  - Implemented `CodexProvider.ParseOutput` with full event handling:
    - Tracks turn counts and aggregates usage statistics
    - Handles `turn.completed` for token usage extraction
    - Handles `item.started`/`item.completed` for tool tracking
    - Properly handles `error` and `turn.failed` events
  - Updated `FormatHeader` in `output.go` to display CLI provider name

- [x] **Task 2: Create provider-specific command builders**
  - Created `internal/loop/provider.go` with `Provider` interface
  - Implemented `ClaudeProvider` with `BuildCommand` and `ParseOutput`
  - Implemented `CodexProvider` with `BuildCommand` and `ParseOutput`
  - Added `NewProvider(cli CLIProvider) (Provider, error)` factory function
  - Refactored `runClaudeIteration` → `runIteration` to use `Provider` interface

- [x] **Task 1: Add CLI provider type and configuration**
  - Added `CLIProvider` type (`claude`, `codex`) in `internal/loop/types.go`
  - Added `CLI` field to `Config` struct in `internal/loop/loop.go`
  - Added `--cli` flag to both `cmd/build.go` and `cmd/plan.go` (default: `claude`)
  - Support `GORALPH_CLI` environment variable as fallback
  - Added `ValidateCLIProvider` function for validation
