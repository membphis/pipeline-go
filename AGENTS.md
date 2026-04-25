# orchestrator — AI Project Pipeline Orchestrator

## Quick start
```bash
uv sync                          # install deps
uv run pytest tests/ -v          # all tests
uv run pytest tests/test_X.py::test_Y -v  # single test
uv run ruff check orchestrator/ tests/   # lint
uv run mypy orchestrator/                 # typecheck
```

## Architecture
- **Entrypoint**: `orchestrator/main.py:main` → CLI: `orchestrator [--spec project.yaml] [--root .] [--branch X] [--extra-spec Y]`
- **Config via `pyproject.toml`**: single dep `pyyaml>=6.0`, Python >=3.10, entry point `orchestrator = "orchestrator.main:main"`
- **Core loop** (`Pipeline.run()`): load YAML → preflight → topo sort → for each milestone: create git branch → run opencode session → run verify → collect HANDOFF.md → phase review → squash-merge → tag
- **Modules**: `main.py` (orchestration), `spec.py` (load/validate yaml), `topo.py` (dependency sort + CycleError), `state.py` (state.yaml persistence), `session.py` (opencode subprocess), `git.py` (branch/commit/tag/merge), `handoff.py` (HANDOFF.md collector), `verify.py` (shell command runner), `review.py` (phase/final review via opencode), `context.py` (context bundle + token budget)
- **Design doc** at `design.md` describes an aspirational v1 with spec→task hierarchy; actual impl is milestone-level only (no nested spec/task level). Keep this in mind when reading design.md.

## Key conventions
- Module-level functions > classes (only `Pipeline`, `State` use classes for state)
- Dataclasses for data objects (`HandoffNote`, `SessionResult`, `VerifyResult`, `ReviewResult`)
- Logging: `from orchestrator import log; logger = log.get(__name__)`
- Error handling: raise domain-specific exceptions (`CycleError`, `RuntimeError` for git failures, `FileNotFoundError` for missing opencode)
- Tests use `unittest.mock` + pytest, **no pytest-mock**; isolation via `tmp_path`, `patch`, fixtures
- **All test modules need `_setup_log` fixture**: `@pytest.fixture(autouse=True) def _setup_log(): log.setup()`

## Pipeline execution details
- YAML must have `project.name` and `milestones[]` with `name` and `tasks[]`
- `depends_on` is milestone-level string list, validated during preflight + runtime via topo sort
- `verify` can be string or list; return code 0 = pass
- `--branch X` skips branch creation (work on existing branch); otherwise creates `{name}-pipeline`
- Each milestone runs ONE opencode session (collective prompt for all tasks)
- Handoff notes collected from `**/HANDOFF.md` after each milestone
- State persisted to `state.yaml` (gitignored)
- Tags: `{ms_name}-done` per milestone, `{project_name}-v1.0` on full completion
- Default branch auto-detected from `refs/remotes/origin/HEAD`, falls back to `main`

## Context bundle
- `context.py` estimates ~4 chars/token, threshold at 180k tokens
- Degrade strategies: `no_verify` → skip verify results; `no_handoff` → skip handoff notes; `minimal` → skip both
- Manual degrade in `review.py` only (auto-degrade not implemented in main loop)

## Testing quirks
- `test_session.py` needs `_mock_opencode_path` fixture (patches `shutil.which` to return `/usr/bin/opencode`)
- `test_main.py` integration-style tests mock all 7 modules; add new mocks for any new module dependency
- `test_context.py` imports `HandoffNote` and `VerifyResult` from `context` module (these are duplicated in `verify.py` and `handoff.py`)
- Run `pytest tests/ -v` from project root after any change to verify
