# Implementation Plan

## Overview

Add support for OpenAI Codex CLI as an alternative to Claude Code CLI. Users can choose which CLI to use via a `--cli` flag or environment variable.

## Key Differences Between CLIs

| Feature | Claude Code | Codex CLI |
|---------|-------------|-----------|
| Command | `claude` | `codex exec` |
| Piped input | `-p` flag + stdin | stdin with `-` |
| Skip permissions | `--dangerously-skip-permissions` | `--dangerously-bypass-approvals-and-sandbox` or `--yolo` |
| JSON output | `--output-format=stream-json` | `--json` (newline-delimited) |
| Verbose | `--verbose` | (default behavior) |

## Tasks

- [ ] **Task 1: Add CLI provider type and configuration**
  - Add `CLIProvider` type (`claude`, `codex`) in `internal/loop/types.go`
  - Add `CLI` field to `Config` struct in `internal/loop/loop.go`
  - Add `--cli` flag to both `build.go` and `plan.go` commands (default: `claude`)
  - Support `GORALPH_CLI` environment variable as fallback

- [ ] **Task 2: Create provider-specific command builders**
  - Create `internal/loop/provider.go` with `Provider` interface
  - Implement `ClaudeProvider` that builds Claude CLI command
  - Implement `CodexProvider` that builds Codex CLI command
  - Refactor `runClaudeIteration` to use provider interface
  - Handle different JSON output formats between providers

- [ ] **Task 3: Adapt output parsing for Codex JSON format**
  - Research Codex CLI's newline-delimited JSON event format
  - Add Codex-specific message types to `types.go` if needed
  - Create `parseCodexOutput` function or extend `parseClaudeOutput` to handle both formats
  - Update `FormatHeader` to show which CLI is being used
  - Test with both CLIs and ensure consistent output formatting

## Completed

<!-- Completed tasks will be moved here as: - [x] Task description -->
