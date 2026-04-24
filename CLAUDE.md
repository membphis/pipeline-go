# AI Project Pipeline Orchestrator

## Build/Test/Lint Commands
- Run all tests: `uv run pytest tests/ -v`
- Run single test: `uv run pytest tests/test_FILE.py::test_NAME -v`
- Run tests with coverage: `uv run pytest tests/ --cov=orchestrator`
- Lint: `uv run ruff check orchestrator/ tests/`
- Type check: `uv run mypy orchestrator/`

## Code Style
- Python 3.10+ with type hints for all public functions
- Use `uv` for package management (not pip directly)
- Tests use pytest with unittest.mock
- All tests must be isolated (use tmp_path, patches, fixtures)
- Follow existing patterns in the codebase

## Project Structure
- `orchestrator/` — package source code
- `tests/` — pytest tests mirroring orchestrator/ structure
- `sample-project/` — example project.yaml for testing
- `CLAUDE.md` — this file

## Key Patterns
- Module-level functions (not classes) unless state is needed
- Type hints via `from __future__ import annotations` or `from typing import Any`
- Logging via `orchestrator.log.get(__name__)`
- Error handling: raise domain-specific exceptions with clear messages
- Dataclasses for structured data objects

## Pipeline Architecture
The orchestrator runs project pipelines defined in YAML:
- `project.yaml` defines milestones with specs, dependencies, tasks, and verify commands
- Each milestone creates a git branch, runs AI session, runs verify, phase review
- At completion: squash-merge to main, tag, final review
