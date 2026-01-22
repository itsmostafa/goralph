# Implementation Plan

## Tasks

- [ ] **Task 2: Create provider-specific command builders**
  - Create `internal/loop/provider.go` with `Provider` interface
  - Implement `ClaudeProvider` with `BuildCommand` and `ParseOutput`
  - Implement `CodexProvider` with `BuildCommand` and `ParseOutput`
  - Add `NewProvider(cli CLIProvider) (Provider, error)` factory function
  - Refactor `runClaudeIteration` → `runIteration` to use `Provider` interface

- [ ] **Task 3: Implement Codex output parsing**
  - Add Codex-specific message types to `types.go`
  - Implement `CodexProvider.ParseOutput` for Codex JSON events
  - Update `FormatHeader` in `output.go` to display CLI provider name

## Completed

- [x] **Task 1: Add CLI provider type and configuration**
  - Added `CLIProvider` type (`claude`, `codex`) in `internal/loop/types.go`
  - Added `CLI` field to `Config` struct in `internal/loop/loop.go`
  - Added `--cli` flag to both `cmd/build.go` and `cmd/plan.go` (default: `claude`)
  - Support `GORALPH_CLI` environment variable as fallback
  - Added `ValidateCLIProvider` function for validation
