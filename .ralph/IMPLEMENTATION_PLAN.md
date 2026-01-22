# Implementation Plan

## Tasks

- [ ] Align Go version requirements by updating `go.mod` to a supported Go release and reflecting the same version in `README.md`.
- [ ] Make implementation plan handling non-destructive (only create if missing or add a flag to reset), and document the behavior.
- [ ] Improve git branch handling for detached HEAD or missing remotes so pushes fail gracefully with clear messaging.
- [ ] Document CLI provider selection (`--cli`, `GORALPH_CLI`) and Codex requirements/behavior in `README.md` and `CLAUDE.md`.
- [ ] Add focused tests for stream parsing and prompt/plan assembly to catch JSON parsing or plan injection regressions.
- [ ] Add preflight checks for required external binaries (git, claude/codex) to emit actionable errors before running.

## Completed

<!-- Completed tasks will be moved here as: - [x] Task description -->
