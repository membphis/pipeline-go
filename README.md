# AI Project Pipeline Orchestrator

AI 项目流水线编排器，将复杂项目分解为多个里程碑（Milestone），
每个里程碑创建 git 分支、调用 AI 编码会话、执行验证命令、
进行阶段审查，最后合并回主分支。

## Quick Start

```bash
# Build
cd orc && go build -o orc .

# Run
./orc/orc [--spec project.yaml] [--root .] [--branch X]
```

## Usage

Write a `project.yaml`:

```yaml
project:
  name: my-project

milestones:
  - name: setup
    spec: Initialize project
    tasks:
      - name: init
        prompt: Create project structure
    verify:
      - ls -la
```

Run:

```bash
# Basic
orc

# Custom spec
orc --spec path/to/project.yaml

# Combine specs
orc --spec base.yaml --extra-spec features.yaml

# Existing branch (skip git branch creation)
orc --branch existing-branch
```

## Requirements

- Go 1.22+
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
│   ├── git/                   # Git 操作
│   ├── session/               # AI 会话管理
│   ├── handoff/               # HANDOFF.md 收集
│   ├── verify/                # 验证命令执行
│   ├── review/                # 阶段/最终审查
│   ├── context/               # 上下文包管理
│   └── log/                   # 日志配置
├── go.mod
└── sample-project/
```

## How It Works

1. **Load YAML** — 解析 project.yaml
2. **Preflight** — 校验格式和依赖完整性
3. **Topo Sort** — 根据 depends_on 计算执行顺序
4. **Per Milestone** — 创建分支 → 运行 opencode → 验证 → 收集 HANDOFF.md → 阶段审查 → 合并
5. **Final Review** — 全部完成后最终审查
6. **Tag** — 成功完成后打标签
