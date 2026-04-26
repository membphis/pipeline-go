# orc

一个命令行工具，按 milestone 顺序在同一个 git 分支上执行 AI 编码任务。

你写一个 `project.yaml` 描述要做什么，它逐个 milestone 调用 opencode 去实现，并在每个 milestone 完成后跑你指定的验证命令。断点续传——中途挂了重新跑会自动跳过已完成 spec。

不切分支，所有工作都在当前分支上完成。

## 安装

```bash
cd orc && go build -o orc .
```

## 5 分钟体验

`sample-project/project.yaml` 定义了一个简单的计算器：

```bash
# -s 指定任务文件，--root 指定工作目录
orc -s sample-project/project.yaml --root /tmp/my-calc
```

`--root` 可以是一个空目录（orc 会自动 `git init` 和 `git commit --allow-empty`），也可以是一个已有仓库。

## 怎么写 project.yaml

```yaml
project:
  name: my-project

milestones:
  - id: setup
    name: 项目初始化
    specs:
      - id: s1
        description: 创建项目结构
        spec_file: design/setup.md    # 需求文档路径，相对于 orc 启动目录
        task_count: 3                  # 预估任务数
    verify:
      - go build ./...
      - go test ./...

  - id: core
    name: 核心功能
    depends_on: [setup]                # 等 setup 做完才轮到这个
    specs:
      - id: c1
        description: 实现核心逻辑
        spec_file: design/core.md
        task_count: 5
        test_count: 3                  # >0 时强制走 TDD 流程
    verify: make check                 # 单条命令可以不写列表
```

### 字段速查

| 字段 | 必填 | 说明 |
|------|------|------|
| `project.name` | 是 | 项目名 |
| `milestones[].id` | 是 | 唯一标识，`depends_on` 引用这个 |
| `milestones[].name` | 是 | 显示名，写给人看 |
| `milestones[].depends_on` | 否 | 依赖的 milestone id 列表 |
| `milestones[].specs[].spec_file` | 否 | 需求文档路径 |
| `milestones[].specs[].task_count` | 否 | 任务数量估计 |
| `milestones[].specs[].test_count` | 否 | 需要几个测试，填了就强制 TDD |
| `milestones[].verify` | 否 | 验证命令，string 或 string 数组，exit 0 算过 |

## 常用命令

```bash
# 基本用法
orc -s project.yaml --root ./my-project

# 指定 AI 模型（plan 阶段用推理模型，实现阶段用快的）
orc -s project.yaml --root . --plan-model anthropic/claude-opus-4-5 --exec-model openai/gpt-5.1-codex

```

如果重新执行时有些 milestone 已经完成了（记录在 `state.yaml` 里），它们会被自动跳过。想重来就删掉 `state.yaml`。

## 每个 milestone 内部做了什么

一个 milestone 会依次跑两轮 opencode：

1. **Plan** — 读需求文档，产出 `.orc_history/PLAN.md`（用 `--plan-model`）
2. **Exec** — 读 PLAN.md，写代码、跑测试、做 code review，最后 `git commit`（用 `--exec-model`）

Exec 结束后，orc 会执行 `verify` 里定义的命令，收集 `HANDOFF*.md`，然后对该 milestone 做一次审查。

## 环境要求

- Go 1.23+
- `git`
- `opencode` 命令在 $PATH 中，参见 [opencode.ai](https://opencode.ai)

## 开发

```bash
cd orc
go test ./... -v
go build -o orc .
```
