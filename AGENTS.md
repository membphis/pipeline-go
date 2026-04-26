# orc — AI Project Pipeline Orchestrator

## 首要原则（First Principles）

**不创建独立分支，所有操作在当前分支完成。** 不要做任何分支创建、切换、合并操作。Pipeline 直接在当前工作分支上依次执行 milestone 的 opencode session，不涉及任何 git 分支管理。

## Quick start
```bash
cd orc
go build -o orc .              # build binary
./orc --help                   # verify
go test ./... -v               # run all tests
go test ./internal/topo/ -v    # single package
```

## Architecture
- **Entrypoint**: `orc/main.go` → CLI: `orc [--spec project.yaml] [--root .]`; also supports `-s` shorthand and positional `--extra-spec <path>` for combining YAMLs
- **Config**: `go.mod` module `orc`, single dep `gopkg.in/yaml.v3`, Go 1.23
- **Core loop** (`pipeline.Run()`): load YAML → preflight → topo sort → for each milestone: build prompt (spec content + TDD instructions + state + handoff) → run opencode session → verify → collect HANDOFF.md → phase review → final review → tag on all-completed
- **Modules**: `main.go` (CLI), `internal/pipeline/` (orchestration), `spec/` (YAML load+validate), `topo/` (sort+cycle detection), `state/` (persistence), `session/` (opencode subprocess), `git/` (tag only; most functions are dead code from removed branch logic), `handoff/` (HANDOFF.md collector), `verify/` (shell commands), `review/` (phase/final review), `context/` (bundle+token budget), `log/` (slog wrapper)

## Key conventions
- `internal/*` packages, each in its own directory
- Functions > interfaces; interfaces only for testability (`var execCommand = exec.Command` pattern in git, session, verify)
- Logging: `log.Get("name")` returns `*slog.Logger`
- Error handling: return errors to caller; Pipeline.Run() returns int exit code
- Tests: Go standard `testing` package

## Pipeline details
- YAML uses `project.name`, `milestones[].{id,name,depends_on,specs}` — see `sample-project/project.yaml`
- `depends_on` is milestone-level string list referencing `id` (not `name`)
- `specs[].spec_file` is resolved relative to `--root`
- `specs[].test_count` > 0 triggers TDD instructions in the prompt
- `verify` can be `string` or `[]string`; exit code 0 = pass
- Each milestone = one opencode session, no branch creation
- Handoff notes: `**/HANDOFF.md` collected after each milestone
- State: `state.yaml` (gitignored)
- Tags: `{project_name}-v1.0` on all-completed

## Context bundle
- `context` package: ~4 chars/token, threshold 180k
- Degrade: `no_verify` → skip verify results; `no_handoff` → skip handoff; `minimal` → skip both
- Degrade triggered automatically in `review` package only

## Testing quirks
- Mock external commands via `var execCommand = exec.Command` pattern (git, verify)
- `topo`, `spec`, `state`, `context` have no external deps, easily testable
- Run `go test ./... -v` from `orc/` after any change
