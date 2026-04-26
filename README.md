# Orc — AI Project Pipeline Orchestrator

AI 项目流水线编排器，将复杂项目分解为多个里程碑（Milestone），
在当前分支上依次调用 AI 编码会话，执行验证命令，进行阶段审查。

> **核心原则：不创建 git 分支。** 所有 milestone 直接在当前分支执行。

## Quick Start

```bash
# Build
cd orc && go build -o orc .

# Run
orc -s ./sample-project/project.yaml
```

## Usage

Write a `project.yaml`:

```yaml
project:
  name: my-project

milestones:
  - id: m1-setup
    name: Project Setup
    specs:
      - id: s1
        description: Initialize project structure
        task_count: 2
        est_minutes: 10
        spec_file: design/setup.md
    verify:
      - go build ./...
```

Run:

```bash
# Basic (default: project.yaml)
orc

# Custom spec
orc --spec path/to/project.yaml

# Shorthand
orc -s path/to/project.yaml

# Custom root
orc --root /path/to/project

# Combine multiple spec files
orc --spec base.yaml --extra-spec features.yaml
```

## Requirements

- Go 1.23+
- Git
- [opencode](https://opencode.ai) (on `$PATH`)

## Development

```bash
cd orc
go test ./... -v
go build -o orc .
```

## Project Structure

```
orc/
├── main.go                    # CLI 入口
├── internal/
│   ├── pipeline/              # Pipeline 核心编排
│   ├── spec/                  # YAML 加载和校验
│   ├── state/                 # 状态持久化
│   ├── topo/                  # 拓扑排序
│   ├── git/                   # Git 操作（仅打标签）
│   ├── session/               # opencode 子进程管理
│   ├── handoff/               # HANDOFF.md 收集
│   ├── verify/                # 验证命令执行
│   ├── review/                # 阶段/最终审查
│   ├── context/               # Token 预算管理
│   └── log/                   # 日志配置
├── go.mod
└── sample-project/
```

## How It Works

1. **Load YAML** — 解析 project.yaml
2. **Preflight** — 校验格式和依赖完整性
3. **Topo Sort** — 根据 depends_on 计算执行顺序
4. **Per Milestone** — 运行 opencode → 验证 → 收集 HANDOFF.md → 阶段审查
5. **Final Review** — 全部完成后最终审查
6. **Tag** — 成功完成后打 `{name}-v1.0` 标签
