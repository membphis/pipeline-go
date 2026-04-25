# AI Project Pipeline Orchestrator

AI 项目流水线编排器，将复杂项目分解为多个里程碑（Milestone），每个里程碑创建一个 git 分支、调用 AI 编码会话、执行验证命令、进行阶段审查，最后合并回主分支。

## 工作流程

```
project.yaml ──► 拓扑排序 ──► 逐个里程碑执行 ──► 完成
                    │               │
                    │               ├── 创建 git 分支
                    │               ├── 调用 AI 编码会话
                    │               ├── 运行验证命令
                    │               ├── 收集 HANDOFF.md
                    │               ├── 阶段审查
                    │               └── squash-merge 回主分支
                    │
                    └── 最终审查
```

## 安装

```bash
pip install pyyaml>=6.0
```

或使用 uv：

```bash
uv sync
```

## 使用方法

### 1. 编写 project.yaml

在项目根目录创建 `project.yaml`，定义里程碑和任务：

```yaml
project:
  name: my-project
  description: 项目描述

milestones:
  - name: setup
    spec: |
      初始化项目结构，配置依赖管理和开发环境。
    verify:
      - ls -la
      - python3 -c "import yaml; print('yaml available')"
    tasks:
      - name: init
        prompt: 创建 Python 项目结构，包含 pyproject.toml、src/ 和 tests/ 目录

  - name: core-logic
    depends_on: [setup]
    spec: |
      实现核心业务逻辑，遵循 setup 阶段定义的架构。
    verify:
      - python3 -m pytest tests/ -x -q
    tasks:
      - name: implement-core
        prompt: |
          实现主要应用逻辑：
          1. 创建数据模型
          2. 实现 API 端点
          3. 添加验证层

  - name: polish
    depends_on: [core-logic]
    spec: |
      最终打磨，包括文档、测试和性能优化。
    verify:
      - python3 -m pytest tests/ -x -q
    tasks:
      - name: docs
        prompt: 为所有公共 API 添加完善的文档和类型标注
      - name: tests
        prompt: 添加覆盖主要使用场景的集成测试
```

### 2. 运行流水线

```bash
# 基本用法
orchestrator

# 指定自定义 spec 文件
orchestrator --spec path/to/project.yaml

# 合并多个 spec 文件
orchestrator --spec base.yaml --extra-spec features.yaml

# 指定项目根目录
orchestrator --root /path/to/project

# 在现有分支上运行（不创建里程碑分支）
orchestrator --branch existing-branch
```

### 3. 理解执行过程

1. **拓扑排序**：根据 `depends_on` 计算里程碑执行顺序
2. **状态追踪**：每个里程碑的状态（pending → in_progress → completed/failed）保存在 `state.yaml`
3. **Git 分支**：每个里程碑在独立分支上工作（如 `setup-pipeline`），完成后 squash-merge 回主分支
4. **验证**：每个里程碑完成后自动运行 `verify` 中定义的命令
5. **交接笔记**：AI 会话生成的 `HANDOFF.md` 会被自动收集并传递给后续里程碑
6. **阶段审查**：每个里程碑完成后自动调用 AI 进行阶段审查
7. **最终审查**：所有里程碑完成后进行项目最终审查
8. **标签**：成功完成所有里程碑后自动打标签（`{project-name}-v1.0`）

## 配置项

### 里程碑（Milestone）

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 里程碑名称（唯一） |
| `spec` | string | 否 | 里程碑的详细规格说明 |
| `tasks` | array | 是 | 任务列表（至少一项） |
| `depends_on` | array | 否 | 依赖的里程碑名称列表 |
| `verify` | array/string | 否 | 验证命令（字符串或列表） |

### 任务（Task）

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 任务名称 |
| `prompt` | string | 是 | 发送给 AI 的提示词 |

### 验证命令（Verify）

支持单条命令（字符串）或多条命令（列表）。每条命令返回码为 0 表示通过，非零表示失败。

```yaml
verify: python3 -m pytest tests/ -x -q

# 或
verify:
  - python3 -m pytest tests/ -x -q
  - ruff check src/
  - mypy src/
```

## 前置条件

- Python 3.10+
- Git
- [opencode](https://opencode.ai)（AI 编码会话依赖，需在 `$PATH` 中）
- `pyyaml>=6.0`

## 开发

```bash
# 安装依赖
uv sync

# 运行测试
uv run pytest tests/ -v

# 运行单个测试
uv run pytest tests/test_FILE.py::test_NAME -v

# 代码检查
uv run ruff check orchestrator/ tests/

# 类型检查
uv run mypy orchestrator/
```

## 项目结构

```
orchestrator/
├── __init__.py
├── main.py        # CLI 入口和 Pipeline 编排逻辑
├── spec.py        # YAML spec 加载和校验
├── state.py       # 里程碑状态持久化
├── topo.py        # 依赖拓扑排序
├── git.py         # Git 操作封装
├── session.py     # AI 编码会话管理
├── handoff.py     # HANDOFF.md 收集
├── verify.py      # 验证命令执行
├── review.py      # 阶段和最终审查
├── context.py     # 上下文包管理
└── log.py         # 日志配置
```

## 常见问题

**Q: 如何跳过失败的里程碑？**

A: 失败的里程碑状态标记为 `failed`，流水线继续执行后续里程碑。所有里程碑完成后返回非零退出码。

**Q: 如何中断后继续？**

A: 状态保存在 `state.yaml` 中，重新运行时会从上次中断处继续。已完成的里程碑不会被重复执行。

**Q: 如何自定义 AI 模型？**

A: project.yaml 无需配置—流水线调用 `opencode` 命令，按 opencode 的配置选择模型。
