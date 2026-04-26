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
- **Entrypoint**: `orc/main.go` → CLI: `orc [--spec project.yaml] [--root .] [--plan-model MODEL] [--exec-model MODEL]`; also supports `-s` shorthand and positional `--extra-spec <path>` for combining YAMLs
- **Config**: `go.mod` module `orc`, single dep `gopkg.in/yaml.v3`, Go 1.23
- **Core loop** (`pipeline.Run()`): load YAML → preflight → topo sort → for each milestone: plan session (Phase 1, plan-model) → exec session (Phase 2-5, exec-model) → verify → collect HANDOFF.md → phase review (per-milestone opencode session, exec-model) → all-completed log
- **Modules**: `main.go` (CLI), `internal/pipeline/` (orchestration), `spec/` (YAML load+validate), `topo/` (sort+cycle detection), `state/` (persistence), `session/` (opencode subprocess), `git/` (repo init, scaffold, tag), `handoff/` (HANDOFF.md collector), `verify/` (shell commands), `review/` (phase review), `context/` (bundle+token budget), `log/` (slog wrapper)

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
- Each milestone = two opencode sessions + one review session: plan (Phase 1, plan-model), exec (Phase 2-5, exec-model), phase review (exec-model); no branch creation
- Handoff notes: `**/HANDOFF*.md` collected after each milestone (written to `.orc_history/HANDOFF-{id}.md` by sessions)
- State: `state.yaml` (gitignored)
- Init scaffold: creates `README.md`, `.gitignore` (with `.orc_history/` entry), `.orc_history/.gitignore` (with `*`)

## Context bundle
- `context` package: ~4 chars/token, threshold 180k
- Degrade: `no_verify` → skip verify results; `no_handoff` → skip handoff; `minimal` → skip both
- Degrade triggered automatically in `review` package only

## Testing quirks
- Mock external commands via `var execCommand = exec.Command` pattern (git, verify)
- `topo`, `spec`, `state`, `context` have no external deps, easily testable
- Run `go test ./... -v` from `orc/` after any change
