# Implementation Plan: Add Codex CLI Support

## Overview

Add support for OpenAI's Codex CLI as an alternative to Claude Code. Users will be able to choose between `claude` and `codex` as the AI backend for the agentic loop.

### Key Differences Between CLIs

| Feature | Claude | Codex |
|---------|--------|-------|
| Command | `claude` | `codex exec` |
| Prompt flag | `-p` (stdin) | positional or stdin |
| Skip permissions | `--dangerously-skip-permissions` | `--yolo` or `--dangerously-bypass-approvals-and-sandbox` |
| JSON output | `--output-format=stream-json` | `--json` (newline-delimited JSON events) |
| Verbose | `--verbose` | (not needed for JSON mode) |

## Tasks

- [ ] **2. Implement ClaudeRunner** - Move Claude-specific logic from `loop.go` into `internal/runner/claude.go`. Implement the `Runner` interface for Claude. Extract `runClaudeIteration` logic into the runner, including command construction and output parsing.

- [ ] **3. Implement CodexRunner** - Create `internal/runner/codex.go` implementing the `Runner` interface for Codex CLI. Use `codex exec` with `--json` for structured output. Parse Codex JSON events and map to common `Result` struct. Handle Codex-specific flags (`--yolo`, `--full-auto`).

- [ ] **4. Add runner selection to Config** - Add `Runner` field to `loop.Config` struct (type `runner.Runner`). Update `loop.Run()` to use the configured runner instead of hardcoded Claude. Keep backward compatibility by defaulting to Claude if no runner specified.

- [ ] **5. Add --runner flag to CLI commands** - Add `--runner` or `-r` flag to `build` and `plan` commands accepting "claude" or "codex". Create runner factory function in `cmd/` package that instantiates the correct runner. Update both command files to pass the runner to `loop.Config`.

- [ ] **6. Add runner display to header output** - Update `FormatHeader()` to show which runner is being used. Add runner name to the `Config` struct or pass it separately.

- [ ] **7. Create runner-specific output types** - Define Codex-specific JSON message types in `internal/runner/codex_types.go`. Map Codex events to the existing streaming display format where possible. Handle differences in tool invocation reporting.

- [ ] **8. Add configuration file support for default runner** - Create optional config file support (e.g., `.ralph/config.toml` or environment variable `GORALPH_RUNNER`). Allow users to set their preferred default runner without CLI flags.

- [ ] **9. Update documentation and help text** - Update root command long description to mention both CLI options. Add runner information to `--help` output for build and plan commands. Update CLAUDE.md with new CLI usage examples.

- [ ] **10. Add runner validation** - Verify the selected runner CLI is installed and accessible before starting the loop. Provide helpful error messages if the CLI is not found (e.g., "codex not found, install with: npm i -g @openai/codex").

## Completed

- [x] **1. Define Runner interface** - Created `internal/runner/runner.go` with a `Runner` interface that abstracts the AI CLI execution. Defined methods: `Name() string`, `Command() *exec.Cmd`, `ParseOutput(io.Reader, io.Writer, io.Writer) (*Result, error)`. Created `Result` and `Usage` structs with common fields.
