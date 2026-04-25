# Orchestrator Go 重写设计

## 动机

将 Python 版 orchestrator 重写为 Go，编译为独立二进制，
消除 Python 运行时依赖，便于分发和跨环境使用。

## 范围

1:1 功能复刻，不引入新功能。同步更新 README.md 和 AGENTS.md，
完成后删除 Python 版源码。

## 目录结构

```
orchestrator/
├── go.mod                     # module orchestrator
├── main.go                    # flag 解析 → Pipeline.Run()
├── internal/
│   ├── pipeline/              # Pipeline 核心编排
│   ├── spec/                  # YAML 加载 + preflight
│   ├── state/                 # state.yaml 持久化
│   ├── topo/                  # 拓扑排序 + CycleError
│   ├── git/                   # git 操作封装
│   ├── session/               # opencode 子进程
│   ├── handoff/               # HANDOFF.md 收集
│   ├── verify/                # shell 命令执行
│   ├── review/                # 阶段/最终审查
│   ├── context/               # context bundle + token 预算
│   └── log/                   # slog 日志配置
├── README.md
├── AGENTS.md
└── sample-project/
    └── project.yaml
```

## 模块职责

| 模块 | 职责 | Python 对照 |
|------|------|-------------|
| `main.go` | flag 解析，构造 Config，调 pipeline.Run | `main.py:parse_args()` |
| `internal/pipeline` | Pipeline.Run() 核心循环：加载→拓扑→逐里程碑执行 | `main.py:Pipeline.run()` |
| `internal/spec` | 读 YAML → Spec 结构体；preflight 校验（名称唯一性、depends_on 完整性） | `spec.py` |
| `internal/state` | 读/写 state.yaml；Status 类型（pending/in_progress/completed/failed） | `state.py` |
| `internal/topo` | DFS 拓扑排序；检测循环抛出 CycleError | `topo.py` |
| `internal/git` | current_branch、create_branch、commit、checkout、squash_merge、tag、is_clean | `git.py` |
| `internal/session` | exec opencode 子进程；超时处理 | `session.py` |
| `internal/handoff` | filepath.Walk 递归收集 HANDOFF.md → HandoffNote 结构体 | `handoff.py` |
| `internal/verify` | exec shell 命令 → VerifyResult（returncode/stdout/stderr） | `verify.py` |
| `internal/review` | 组装 context bundle → session.run → ReviewResult；含 degrade 降级 | `review.py` |
| `internal/context` | build_bundle、exceeds_threshold、degrade（no_verify/no_handoff/minimal） | `context.py` |
| `internal/log` | slog 配置，INFO 级别，时间戳+级别+包名 | `log.py` |

## CLI 接口

```bash
orchestrator [--spec project.yaml] [--root .] [--branch X] [--extra-spec Y]
```

- `--spec` / `-s` — 默认 `project.yaml`
- `--extra-spec` — 可多次指定，合并到 milestones 列表
- `--root` — 默认 `.`
- `--branch` — 跳过分支创建，在现有分支上工作

退出码：全完成 0，有失败 1。

## Pipeline.Run() 核心流程

```
load spec → preflight → topo sort
for each milestone in ordered:
  state.set(milestone, in_progress) → save
  if branch_override: use existing branch
  else: checkout base → create_branch(name-pipeline)
  prompt = compute_milestone_spec(...)  # 含里程碑信息、state、handoff notes
  session.run(prompt)
  if success: state.set(milestone, completed)
  else: state.set(milestone, failed); has_failures = true
  verify.run(ms.verify) → collect VerifyResult
  handoff.collect(root) → dedup → collect HandoffNote
  review.review_phase(...)
  if not branch_override:
    git.commit → git.tag → checkout base → squash_merge
review.review_final(...)
if all_completed: git.tag(project-v1.0)
```

失败处理：session 失败不阻断后续里程碑。所有里程碑完成后返回非零退出码。

## 依赖

- `gopkg.in/yaml.v3` — YAML 解析
- 其余全标准库：`os/exec`、`log/slog`、`flag`、`path/filepath`、`time`、`math`、`strings`

## 关键数据流（代码中的类型）

```
Spec struct:
  Project: { Name string, Description string }
  Milestones: []Milestone

Milestone struct:
  Name string
  Spec string
  DependsOn []string
  Tasks []TaskSpec
  Verify []string

TaskSpec struct:
  Name string
  Prompt string

StateData: map[MilestoneName] → { Status (pending|in_progress|completed|failed), Timestamp }

HandoffNote: { Source string, Content string }
VerifyResult: { Returncode int, Stdout string, Stderr string }
SessionResult: { Returncode int, Stdout string, Stderr string }
ReviewResult: { Returncode int, Stdout string, Stderr string }
```

## 测试

- 使用 Go 标准 `testing` 包
- `internal/*` 每个包有 `*_test.go`
- 对外部命令（git、opencode、shell）用接口/函数变量注入 mock
- `spec`、`topo`、`state`、`context` 无外部依赖，可直接测试

## 删除清单

完成后删除：
- `orchestrator/__init__.py`、`main.py`、`spec.py`、`state.py`、`topo.py`、`git.py`、`session.py`、`handoff.py`、`verify.py`、`review.py`、`context.py`、`log.py`
- `tests/` 整个目录
- `pyproject.toml`
- `uv.lock`
- `.python-version`（如有）
- `.github/`（如有 Python 相关 CI）
- `orchestrator.egg-info/`
- `.mypy_cache/`、`.pytest_cache/`、`.ruff_cache/`
- `.venv/`

## 更新文档

- `README.md` — 替换为 Go 版构建/运行说明
- `AGENTS.md` — 替换为 Go 版开发命令和结构说明
