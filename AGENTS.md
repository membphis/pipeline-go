# orc — AI Project Pipeline Orchestrator

## Quick start
```bash
go build -o orc .              # build binary
./orc --help                   # verify
go test ./... -v               # run all tests
go test ./internal/topo/ -v    # single package
```

## Architecture
- **Entrypoint**: `orc/main.go` → CLI: `orc [--spec project.yaml] [--root .] [--branch X]`
- **Config**: `go.mod` module `orc`, single dep `gopkg.in/yaml.v3`, Go 1.22+
- **Core loop** (`pipeline.Run()`): load YAML → preflight → topo sort → for each milestone: create git branch → run opencode session → run verify → collect HANDOFF.md → phase review → squash-merge → tag
- **Modules**: `main.go` (CLI), `internal/pipeline/` (orchestration), `spec/` (YAML load+validate), `topo/` (sort+cycle detection), `state/` (persistence), `session/` (opencode subprocess), `git/` (branch/commit/tag/merge), `handoff/` (HANDOFF.md collector), `verify/` (shell commands), `review/` (phase/final review), `context/` (bundle+token budget), `log/` (slog wrapper)

## Key conventions
- `internal/*` packages, each in its own directory
- Functions > interfaces; interfaces only for testability (execCommand variables)
- Logging: `log.Get("name")` returns `*slog.Logger`
- Error handling: return errors to caller; Pipeline.Run() returns int exit code
- Tests: Go standard `testing` package

## Pipeline details
- YAML must have `project.name` and `milestones[]` with `name` and `tasks[]`
- `depends_on` is milestone-level string list
- `verify` can be string or `[]string`; exit code 0 = pass
- `--branch X` skips branch creation
- Each milestone = one opencode session
- Handoff notes: `**/HANDOFF.md` collected after each milestone
- State: `state.yaml` (gitignored)
- Tags: `{ms_name}-done` per milestone, `{project_name}-v1.0` on completion
- Default branch: detected from `git rev-parse --abbrev-ref HEAD`, fallback `main`

## Context bundle
- `context` package: ~4 chars/token, threshold 180k
- Degrade: `no_verify` → skip verify results; `no_handoff` → skip handoff; `minimal` → skip both
- Manual degrade in `review` package only

## Testing quirks
- Mock external commands via `var execCommand = exec.Command` pattern (git, session, verify)
- `session` tests patch `execCommand` to avoid real opencode calls
- `topo`, `spec`, `state`, `context` have no external deps, easily testable
- Run `go test ./... -v` from `orc/` after any change
