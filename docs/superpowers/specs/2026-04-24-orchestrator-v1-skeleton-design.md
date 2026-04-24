# Orchestrator v1 Skeleton — 设计规格

## 1. 概述

基于 design.md 实现的 AI 项目流水线编排系统第一版骨架。核心目标：跑通 milestone → spec → task 三级粒度的完整执行链路，支持 `orchestrator run` 和 `orchestrator run --resume`。

暂缓能力：retry + 错误恢复机制（v2）。

## 2. 架构

```
orchestrator/
├── __init__.py
├── main.py                # CLI 入口 (argparse)
├── spec.py                # project.yaml 解析 + preflight 校验
├── topo.py                # 拓扑排序 + 循环依赖检测
├── state.py               # state.yaml 读写
├── session.py             # 调 opencode CLI 执行 task
├── verify.py              # 执行 verify 命令
├── handoff.py             # handoff 收集、写入、注入
├── context.py             # context bundle 组装 + 200k 降级
├── review.py              # Phase Review + Final Review
├── git.py                 # Git 操作封装 (branch/commit/merge/tag)
└── log.py                 # 结构化日志
```

### 模块依赖关系（实现顺序）

| 模块 | 职责 | 依赖 |
|------|------|------|
| `log.py` | 统一日志输出（INFO/WARN/ERROR） | 无 |
| `spec.py` | 解析 project.yaml → Py 数据结构；preflight 校验 | log |
| `topo.py` | 拓扑排序，检测循环依赖和无效 depends_on | log |
| `state.py` | state.yaml 初始化/读取/更新 | log |
| `git.py` | git init、branch、commit、merge(squash)、tag | log |
| `verify.py` | 执行 verify 命令，返回 pass/fail | log |
| `session.py` | 组装 prompt → `opencode --prompt-file` → 收集输出 | log |
| `handoff.py` | 写入模板 handoff、收集上游 handoff 内容 | log |
| `context.py` | 组装 context bundle，token 估算，降级策略 | log |
| `review.py` | Phase/Final Review session 启动 | session, handoff, git |
| `main.py` | CLI 解析 → 编排执行流程 | 全部模块 |

## 3. 内部数据结构

### project.yaml 解析（spec.py）

project.yaml 过大时可拆分为多个文件（如 `project.yaml` + `project-m2.yaml`），spec.py 支持读取目录下的 `project*.yaml` 并合并解析。

```python
@dataclass
class Task:
    id: str              # "m1-s1-t1"
    name: str            # "Define structs"
    depends_on: list[str]
    expected_files: list[str]
    prompt: str
    verify: list[str]    # ["cargo build", "cargo test", ...]
    milestone_id: str    # 反向引用
    spec_id: str

@dataclass
class Spec:
    id: str
    name: str
    depends_on_specs: list[str]
    tasks: list[Task]

@dataclass
class Milestone:
    id: str
    name: str
    order: int
    specs: list[Spec]

@dataclass
class Project:
    name: str
    language: str
    scratch_template: str
    milestones: list[Milestone]
```

### state.yaml 状态（state.py）

```python
@dataclass
class State:
    version: int
    project_name: str
    status: str           # pending|running|completed|partially_failed|failed
    current: Optional[TaskRef]
    completed: list[str]
    failed: dict[str, FailedInfo]
    started_at: str
    last_updated: str

@dataclass
class TaskRef:
    milestone: str
    spec: str
    task: str

@dataclass
class FailedInfo:
    attempts: int
    last_error: str
```

### 状态转换

```
pending ──(run)──▶ running ──(全部完成)──▶ completed
                       │
                       ├──(部分失败但继续)──▶ partially_failed
                       │
                       └──(中断)──▶ 下次 --resume 继续 running
```

## 4. CLI 命令

- `orchestrator run` — 从头执行全部 task
- `orchestrator run --resume` — 从 state.yaml 断点继续

实现：Python argparse，`orchestrator/` 目录作为可执行包（`python -m orchestrator`）。

## 5. 核心执行流程

```python
def run(resume=False):
    project = spec.parse("project.yaml")        # 1. 解析
    spec.preflight(project)                     # 2. 校验
    state = state.load_or_init(project, resume) # 3. 状态初始化
    tasks = topo.sort(project)                  # 4. 全量拓扑排序

    start_idx = find_resume_point(tasks, state) if resume else 0

    for milestone in project.milestones:
        git.checkout_branch(f"milestone/{milestone.id}")

        for task in milestone_tasks(milestone, tasks):
            if task.id in state.completed:
                continue
            if not deps_satisfied(task, state):
                log.error(f"依赖未满足: {task.id}")
                state.mark_failed(task, "依赖的前置 task 未完成")
                continue

            bundle = context.assemble(project, task, state)
            result = session.execute(task, bundle)

            ok = verify.run(task.verify)
            if ok:
                handoff.write(task, result)
                git.commit(task)
                state.mark_complete(task)
            else:
                state.mark_failed(task, result.error)
                continue

        if milestone_has_failures(milestone, state):
            state.status = "partially_failed"

        review.phase(milestone, project, state)
        git.create_tag(f"v0.{milestone.order}.0-{milestone.id}")
        git.merge_to_main(milestone)

    if state.status != "partially_failed":
        state.status = "completed"

    review.final(project, state)
```

关键规则：
- **deps_satisfied**：检查依赖 task 是否在 completed 列表中
- **失败不阻塞**：失败的 task 不阻塞无依赖的后续 task
- **milestone git**：独立分支 → Phase Review → squash merge → tag

## 6. Context Bundle 组装（context.py）

按 design.md §4 结构组装，总量 ≤180k token：

| 章节 | 内容 | 预估 |
|------|------|------|
| CLAUDE.md | 项目编码规范 | ~2k |
| System head | task 描述 + milestone/spec 上下文 | ~1k |
| Handoff | 上游依赖的 handoff 笔记拼接 | ~2k |
| Task prompt | 原始 prompt | ~3k |
| Code snapshot | 文件树 + 相关模块代码 | ~80-120k |
| Output template | 输出要求 | ~1k |

降级策略（触发条件：>180k → 降至 ≤150k）：

| 优先级 | 动作 | 节省 |
|--------|------|------|
| 1 | 只保留直接上游 handoff | ~3-5k |
| 2 | 完整文件 → 仅函数签名+类型定义 | ~40-60k |
| 3 | Handoff 仅注入摘要开头 | ~1k |

## 7. Session 执行（session.py）

```python
def execute(task: Task, context_bundle: str) -> SessionResult:
    prompt_file = f"/tmp/orchestrator-{task.id}-prompt.txt"
    write_file(prompt_file, context_bundle)

    # opencode 的 CLI 参数以实际 opencode --help 为准
    proc = subprocess.run(
        ["opencode", "--prompt-file", prompt_file, "--no-interactive"],
        timeout=600,
        capture_output=True, text=True
    )

    return SessionResult(task_id=task.id, success=proc.returncode == 0,
                         stdout=proc.stdout, stderr=proc.stderr)
```

超时：默认 600s。

## 8. 支撑模块

### verify.py

依次执行 verify 命令列表，全部 0 返回码才 pass。不通过时返回具体失败命令和 stderr。

### handoff.py

- `collect(task)`: 收集 task 所有依赖的 handoff 文件并拼接
- `write(task, result)`: 写入手写 handoff 模板到 `.opencode/handoff/{task-id}-notes.md`

### git.py

- `checkout_branch(name)`: 创建或切换到 `milestone/{name}` 分支
- `commit(task)`: `git add . && git commit -m "task: {task.id} - {task.name}"`
- `create_tag(tag)`: `git tag {tag}`
- `merge_to_main(branch)`: `git checkout main && git merge --squash {branch} && git commit -m "merge milestone/{branch}"`

### review.py

- `phase_review(milestone, project, state)`: milestone 完成后审查 session，context 包含该 milestone 所有 handoff + git diff + 文件树。产出 `review/architecture.md`、`review/quality.md`、`review/TODO.md`。小问题直接修复。
- `final_review(project, state)`: 全量测试 + 文档终审 + CHANGELOG。

## 9. 项目文件布局

```
project-root/
├── orchestrator/          # Python 包
│   ├── __init__.py
│   ├── main.py
│   ├── spec.py
│   ├── topo.py
│   ├── state.py
│   ├── session.py
│   ├── verify.py
│   ├── handoff.py
│   ├── context.py
│   ├── review.py
│   ├── git.py
│   └── log.py
├── project.yaml           # spec 树（可拆为 project*.yaml 多文件）
├── claude.md              # 编码规范（随 context 注入）
├── state.yaml             # 自动生成
├── .opencode/
│   ├── handoff/           # AI 生成
│   ├── logs/              # 结构化日志
│   └── .tmp/              # 临时 prompt
└── review/                # Phase Review 产出
    ├── architecture.md
    ├── quality.md
    └── TODO.md
```

## 10. v1 范围边界

| 能力 | v1 | v2+ |
|------|:--:|:---:|
| spec 树解析 + preflight | ✓ | |
| 拓扑排序 + 循环检测 | ✓ | |
| state.yaml 状态管理 | ✓ | |
| `run` / `run --resume` | ✓ | |
| opencode session 执行 | ✓ | |
| context bundle 组装 + 降级 | ✓ | |
| verify 命令执行 | ✓ | |
| handoff 收集与注入 | ✓ | |
| git branch/commit/merge/tag | ✓ | |
| Phase Review + Final Review | ✓ | |
| Retry + 错误恢复 | | ✓ |
| `--dry-run` | | ✓ |
| 并行执行 | | ✓ |
| 统计报告 | | ✓ |
