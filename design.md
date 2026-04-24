# AI 项目流水线编排系统 — 设计方案

## 1. 问题

有 6~8 个里程碑，每个 10 个左右 spec，每个 spec 有 5~10 个 task，总计约 300~800 个 task。所有 spec 文件已准备完毕，希望一条命令触发 AI 工具（如 OpenCode / Claude Code）一次性执行完所有 task，产出完整工程代码。约束：模型最大上下文窗口 200k token。

## 2. 架构总览

```
project.yaml           state.yaml            handoff/
(spec 树定义)           (执行进度)            (上下文传递)
     │                      │                     │
     └──────────┬───────────┴──────────┬──────────┘
                │                      │
                ▼                      ▼
     ┌─────────────────────────────────────────┐
     │             Orchestrator                 │
     │                                         │
     │  for milestone in order:                │
     │    for task in topological_order():     │
     │      ├─ 校验前置依赖已完成               │
     │      ├─ 读取 task spec 片段             │
     │      ├─ 收集上游 handoff 笔记           │
     │      ├─ 组装 context bundle (≤200k)     │
     │      ├─ 启动 AI session (非交互模式)     │
     │      ├─ 验证 build + test + lint        │
     │      ├─ 成功 → git commit + handoff     │
     │      └─ 失败 → retry ≤ 3 次            │
     │    Phase Review (milestone 质量门)       │
     │  Final Review (全部完成后)              │
     └─────────────────────────────────────────┘
```

### 核心策略

每个 task 启动一个**独立的干净 session**，不共享历史上下文。上下文通过以下渠道传递：

- **双向**: 代码文件（git 提交历史）、结构化数据文件（project.yaml / state.yaml / handoff）
- **单向**: 编排器注入的 context bundle（上游 handoff 笔记 + 相关代码片段）

## 3. 数据文件设计

### 3.1 project.yaml — spec 树定义

```yaml
project:
  name: my-backend
  language: rust   # rust | go
  scratch_template: "cargo init --name my-backend"  # 脚手架命令

milestones:
  - id: m1-core
    name: Core Foundation
    order: 1
    specs:
      - id: m1-s1-data-model
        name: Data Model
        tasks:
          - id: m1-s1-t1
            name: Define structs
            depends_on: []
            expected_files:
              - src/models/mod.rs
            prompt: |
              定义核心数据结构：User、Session、Config。
              使用 serde Serialize/Deserialize derive。
              每个字段加文档注释。
            verify:
              - cargo build
              - cargo test
              - cargo clippy -- -D warnings

          - id: m1-s1-t2
            name: Add validation
            depends_on: [m1-s1-t1]
            expected_files:
              - src/models/mod.rs
              - src/models/validation.rs
            prompt: |
              为 User 和 Session 添加输入验证方法。
              email 验证用 regex crate。
              密码要求：8 位以上，含大小写字母和数字。
            verify:
              - cargo test
              - cargo clippy -- -D warnings

      - id: m1-s2-database
        name: Database Layer
        depends_on_specs: [m1-s1]   # 整个 spec 依赖
        tasks:
          - id: m1-s2-t1
            name: Connection pool
            depends_on: []
            expected_files:
              - src/db/mod.rs
              - src/db/pool.rs
            prompt: |
              基于 sqlx 实现连接池管理...
            verify:
              - cargo build
              - cargo test

  - id: m2-auth
    name: Authentication
    order: 2
    ...
```

**约束**:
- `depends_on` 可引用任何 task ID（跨 spec 跨 milestone）
- `verify` 中的命令全部成功才算 task 完成
- 每个 task 必须有明确的 `expected_files`，验证时检查文件是否存在

### 3.2 state.yaml — 执行状态

由编排器维护，允许断点续跑和人肉干预。

```yaml
version: 1
project_name: my-backend

status: running   # pending | running | completed | partially_failed | failed

current:
  milestone: m1-core
  spec: m1-s1-data-model
  task: m1-s1-t2

completed:
  - m1-s1-t1

failed:
  m1-s1-t5:
    attempts: 3
    last_error: "cargo test 报错：expected 2 tests, got 1"

started_at: "2026-04-24T10:00:00Z"
last_updated: "2026-04-24T12:30:00Z"
```

**支持命令**:
```
orchestrator run                  # 从头开始
orchestrator run --resume         # 从 state.yaml 的 current 继续
orchestrator run --retry m1-s1-t5 # 重跑失败的 task
orchestrator run --from m1-s2     # 从指定 milestone 开始
```

### 3.3 handoff/{task-id}-notes.md — 上下文传递

每个 task 完成后 AI 输出此文件。编排器在后续依赖 task 启动前收集并注入。

```markdown
## Handoff: m1-s1-t1 → m1-s1-t2

### Files Changed

- `src/models/mod.rs` (new): User 结构体，含 id(name, email, password_hash)、验证方法
- `src/models/user.rs` (new): User 的创建和查找逻辑

### Architectural Decisions

- User.id 类型为 uuid::Uuid，用 `Uuid::new_v4()` 生成
- email 字段存储前不做 normalize，查询时做 case-insensitive 匹配
- 密码用 argon2 哈希，这是 Cargo.toml 中已加的依赖

### For Next Task (m1-s1-t2: Add validation)

- email 验证逻辑需要在 `validation.rs` 中实现
- 当前对 email 格式无校验，t2 需要添加 regex 验证
- 密码强度验证建议放在 User 的构建函数中

### Known Limitations

- 尚无完整的 error type 定义，目前用 `anyhow::Error` 顶替
- 尚无数据库集成，User 目前只是结构体定义
```

**关键规则**:
- handoff 由 AI session 在 task prompt 要求下自动生成（通过 output template）
- 编排器不解析 handoff 内容（纯粹文本拼接）
- 如果 AI 写不出来，prompt 中应要求 AI 输出固定格式模板，填空即可

## 4. 单 Task Session 的 Context Bundle

每个 task 启动一个独立 session，编排器注入以下内容：

```
┌──────────────────────────────────────────────────┐
│ project-level CLAUDE.md (编码规范 / 架构约束 /    │
│   使用库 / 命名约定)       ~2k token             │
├──────────────────────────────────────────────────┤
│ system prompt head:                              │
│ "你正在执行 task [m1-s1-t2: Add validation]，     │
│  属于 milestone [m1-core], spec [m1-s1-data-model]│
│  语言: Rust。验证命令: cargo test, cargo clippy"  │
│                                       ~1k token  │
├──────────────────────────────────────────────────┤
│ handoff from dependencies:                       │
│ 从 m1-s1-t1 收集的 handoff 笔记                  │
│                                       ~2k token  │
├──────────────────────────────────────────────────┤
│ task spec (来自 project.yaml):                   │
│ prompt + expected_files + verify 命令            │
│                                       ~3k token  │
├──────────────────────────────────────────────────┤
│ current codebase snapshot:                       │
│ file tree (tree src/ --noreport)                 │
│ + 上游改动文件的完整内容                          │
│ + 所有相关已有模块的文件内容                       │
│ (≈已完成的 task 数 × 预计文件大小，                │
│  控制在 ~80~120k token)                          │
├──────────────────────────────────────────────────┤
│ output template (task prompt 追加):              │
│ "完成后你必须:                                    │
│  1. 新增/更新测试，覆盖主要路径和边缘情况           │
│  2. 所有 verify 命令通过                          │
│  3. handoff 文件写入 .opencode/handoff/{task-id}  │
│  4. 如果影响 API/使用方式，增量更新 README.md"    │
│                                       ~1k token  │
├──────────────────────────────────────────────────┤
│ 估算总量: ~90~130k token (≤200k)                │
└──────────────────────────────────────────────────┘
```

### context bundle 降级策略

如果注入代码 + handoff 超过 180k token，按如下顺序降级直到不超过 150k：

1. 只保留直接上游（immediate predecessors）的 handoff，丢弃更早的
2. 完整文件 → 仅函数签名 + 类型定义（用 `rg "^pub (fn|struct|enum|trait|impl)"` 提取）
3. handoff 发送 complete file content 仅注入摘要开头决策部分

降级编排器自动处理，不中断执行。

## 5. Preflight（执行前校验）

编排器在跑任何 task 之前执行：

| 检查项 | 命令 | 失败处理 |
|--------|------|---------|
| project.yaml 格式 | `python -c "import yaml; yaml.safe_load(open('project.yaml'))"` | 报错退出 |
| task ID 唯一性 | `yq eval '.milestones[].specs[].tasks[].id' project.yaml \| sort \| uniq -d` | 报错退出 |
| depends_on 引用有效 | 遍历所有 depends_on 确保引用的 ID 在 project.yaml 中存在 | 报错退出 |
| 循环依赖检查 | 拓扑排序前置检测 | 报错退出 |
| verify 命令前缀匹配 | check `cargo` / `go` 是否在 PATH 中 | 警告但继续 |
| 环境工具 | `cargo --version && cargo clippy --version && rustc --version` | 报错退出 |
| 脚手架初始化 | 如果是第一个 milestone，先跑 `cargo init` 或预设模板 | 自动执行 |

## 6. 整体执行流程

```
start
 │
 ├─ preflight (全部校验通过)
 │
 ├─ 解析 project.yaml，全量拓扑排序
 │
 ├─ for each milestone (按 order):
 │    │
 │    ├─ git checkout -b milestone/{milestone-id}
 │    │
 │    ├─ for each task (拓扑序):
 │    │    │
 │    │    ├─ 校验所有 depends_on 均在 completed 列表中
 │    │    ├─ 收集上游 handoff
 │    │    ├─ 组装 context bundle
 │    │    ├─ 启动 session
 │    │    │    claude -p "$(cat context_bundle.txt)" --output-format=json
 │    │    │    # 或
 │    │    │    opencode --prompt "$(cat context_bundle.txt)" --no-interactive
 │    │    │
 │    │    ├─ 读取 AI 输出，检测手写 handoff 文件
 │    │    ├─ 执行 verify 命令
 │    │    │
 │    │    ├─ [成功]
 │    │    │    ├─ git add .
 │    │    │    ├─ git commit -m "task: {task-id} - {task-name}"
 │    │    │    └─ 更新 state.yaml (task 加入 completed)
 │    │    │
 │    │    └─ [失败]
 │    │         ├─ retry_count < 3:
 │    │         │    ├─ 注入前次错误输出到 prompt
 │    │         │    ├─ git reset --hard HEAD (回滚)
 │    │         │    └─ retry_count++
 │    │         └─ retry_count >= 3:
 │    │              ├─ state.yaml 标记 FAILED
 │    │              ├─ 记录 last_error
 │    │              └─ continue (不阻塞后续无依赖的 task)
 │    │
 │    ├─ Phase Review (milestone 质量门)
 │    │    ├─ 收集本 milestone 全部 handoff
 │    │    ├─ 启动专用审查 session:
 │    │    │    prompt: "审查 milestone [m1-core] 的代码质量。
 │    │    │    输出架构评估报告和问题清单。更新 README。"
 │    │    ├─ 代码质量问题 → 直接修复 commit
 │    │    ├─ 重大架构改进 → 追加 task 到下一 milestone
 │    │    ├─ README 整体重新审阅更新
 │    │    └─ 打 tag: git tag v0.1.0-m1
 │    │
 │    └─ git checkout main && git merge milestone/{milestone-id}
 │
 ├─ Final Integration Review
 │    ├─ 启动最终审查 session
 │    ├─ 全量测试、端到端验证、模块交叉检查
 │    ├─ API 文档生成
 │    ├─ CHANGELOG 生成
 │    └─ README 终审
 │
 └─ 输出统计报告
```

## 7. Phase Review（里程碑质量门）

### 7.1 为何必要

300~800 个 task 单独执行，难免出现：
- task A 和 task C 的代码风格不一致
- 模块间接口不匹配（A 期望 `Result<T, Error>`，C 实际返回 `Option<T>`）
- 重复逻辑（两个 task 各自实现了同一种工具函数）
- 架构偏离了设计意图

### 7.2 审查 session 的 context bundle

```
┌──────────────────────────────────────────┐
│ 本 milestone 所有 handoff 合集   ~20k    │
│ git log --oneline                  ~2k   │
│ git diff main...HEAD               ~80k  │
│ 文件树概览                         ~1k   │
│ CLAUDE.md (项目规范)                ~2k   │
│                                          │
│ prompt:                                   │
│ "作为架构师审查 milestone 1 的代码质量。   │
│  检查项：                                 │
│  1. 模块边界是否清晰，接口定义是否一致    │
│  2. 错误处理是否统一                      │
│  3. 是否有重复代码/未使用代码             │
│  4. 命名、注释是否一致                    │
│  5. README 是否准确反映当前状态           │
│                                          │
│  输出到 review/architecture.md:           │
│   - 架构图描述                            │
│   - 风险评估                              │
│   - 改进建议                              │
│                                          │
│  小问题直接修复。                          │
│  大问题追加到下一 milestone。"             │
│                                   ~3k    │
│ 估算总量: ~110k token                    │
└──────────────────────────────────────────┘
```

### 7.3 审查产出

```
review/
├── architecture.md     # 架构一致性评估
├── quality.md          # 代码质量问题清单
└── TODO.md             # 追加到下一 milestone 的改进 task
```

### 7.4 README 更新策略

双层次:

- **Task 级（增量更新）**: 每个 task prompt 末尾模板要求：如果影响 API 或使用方式，更新 README 对应章节
- **Milestone 级（整体重审）**: Phase Review 统一审阅 README，补全遗漏，修复不一致，确保文档准确反映当前状态

README 应包含但不限于：
- 项目简介和目标
- 快速开始（构建、运行）
- 核心架构说明
- API 概览（外部暴露的接口）
- 开发指南（测试、lint、提交规范）
- 已知限制

## 8. 错误恢复机制

### 8.1 Task 级失败

```
task execution
    │
    ├─ retry_count < 3:
    │    ├─ prompt 追加: "上次执行失败了，输出如下：{stderr}"
    │    ├─ git reset --hard HEAD (撤销改动)
    │    ├─ retry_count++
    │    └─ 重新执行
    │
    └─ retry_count >= 3:
         ├─ state.yaml 标记 fail
         ├─ 记录 last_error
         ├─ git checkout . (清理工作目录)
         └─ 继续下一个 task
```

### 8.2 编排器进程中断

`state.yaml` 记录当前正在执行的 task。重启时 `--resume` 会：

1. 读取 `state.yaml` 找到 `current`
2. 检查该 task 的代码是否已提交（git log 检查 commit message）
3. 已提交 → 跳过该 task
4. 未提交 → 重新执行该 task

### 8.3 脚本层容错

```python
MAX_RETRY = 3
TIMEOUT_SECONDS = 300  # 单个 AI session 超时
for attempt in range(MAX_RETRY):
    try:
        result = run_subprocess(cmd, timeout=TIMEOUT_SECONDS)
    except TimeoutError:
        log.warning(f"Task {task_id} timed out, retry {attempt+1}")
        continue
```

## 9. 分支与版本策略

```
main
  │
  ├─ milestone/01-core-foundation  ← m1 所有 task 在此分支执行
  │    │
  │    ├─ commit: task: m1-s1-t1 - Define structs
  │    ├─ commit: task: m1-s1-t2 - Add validation
  │    └─ tag: v0.1.0-m1
  │
  ├─ milestone/02-auth              ← merge m1 后创建
  │    │
  │    ├─ commit: task: m2-s1-t1 - JWT generation
  │    └─ tag: v0.2.0-m2
  │
  └─ ...
```

- 每个 milestone 独立分支，隔离失败风险（全炸了可以删分支重来）
- 完成后 merge 到 main
- 每个里程碑打 tag

## 10. 最终集成审查（Final Review）

全部 milestone 完成后执行一次完整审查：

| 动作 | 内容 |
|------|------|
| 全量测试 | `cargo test --all-targets` / `go test ./...` |
| Lint | `cargo clippy -- -D warnings` / `golangci-lint run` |
| 交叉引用检查 | 所有模块公共 API 是否被正确调用 |
| 文档终审 | README + API 文档 + CHANGELOG 一致性 |
| CHANGELOG 生成 | 从所有 handoff + git log 聚合 |
| 最终 tag | `v1.0.0` |

## 11. 编排器实现概览

推荐语言：Python（最轻量，yaml 处理方便）或 Rust/Go（恰好是做后端的语言）

### 核心接口

```
orchestrator run                   # 从头开始
orchestrator run --resume          # 断点继续
orchestrator run --dry-run         # 只输出执行计划，不执行
orchestrator status                # 查看当前进度
orchestrator retry --task m1-s1-t5 # 重跑指定 task
orchestrator validate              # 只做 preflight 校验
```

### 核心模块

```
orchestrator/
├── main.py                # CLI 入口
├── spec.py                # project.yaml 解析 + 校验
├── topo.py                # 拓扑排序
├── session.py             # AI session 管理（cli 调用 + 超时）
├── verify.py              # 验证命令执行
├── state.py               # state.yaml 读写
├── handoff.py             # handoff 收集与注入
├── context.py             # context bundle 组装 + 降级
├── review.py              # Phase Review / Final Review
└── log.py                 # 结构化日志
```

## 12. 缺失点清单（最终汇总）

### P0 — 必须含

- [x] project.yaml spec 树定义格式
- [x] state.yaml 状态管理 + 断点续跑
- [x] handoff 上下文传递机制
- [x] 拓扑排序（跨 spec 跨 milestone 依赖解析）
- [x] context bundle 组装 + 200k 上限控制 + 降级策略
- [x] verify 验证 (build + test + lint)
- [x] 重试机制（3 次上限 + 错误信息注入）
- [x] 预检校验 (project.yaml 格式、依赖完整性、环境工具)
- [x] 项目脚手架初始化
- [x] Phase Review + README 更新
- [x] 分支策略 + merge 流程
- [x] Final Integration Review
- [x] 全局 CLAUDE.md（编码规范、架构约束）

### P1 — 建议第一版包含

- [ ] file tree 索引注入（帮助 AI 定位文件）
- [ ] task 级回滚（git reset --hard 回退）
- [ ] CHANGELOG 自动生成
- [ ] milestone tag 自动打
- [ ] 结构化日志 (logs/{task-id}/)
- [ ] 超时控制

### P2 — 后续版本

- [ ] dry-run 模式（估算任务数/耗时/token）
- [ ] 并行执行（无依赖的 spec 并发）
- [ ] 暂停/恢复信号（文件信号触发）
- [ ] 统计报告（总耗时、token 消耗、成功率）
- [ ] API 速率限制退避
- [ ] prompt 注入检查

## 13. 关键设计决策记录

| 决策 | 选择 | 理由 |
|------|------|------|
| 每个 task 独立 session vs 共享 session | 独立 session | 200k 窗口限制；错误隔离；上下文不膨胀 |
| handoff 格式 | 结构化 markdown | AI 生成容易，人可读，编排器无解析成本 |
| spec 格式 | YAML | 比 JSON 可读性好，支持注释，AI 输出友好 |
| 使用现有 AI 工具 vs 自建 LLM 调用 | 使用现有工具 (`claude` / `opencode`) | 成熟度更高，已处理文件编辑、工具调用等 |
| 串行 vs 并行 | 初版串行，后续加并行 | 简化错误处理，避免并行引发的竞态 |
| 是否在 task prompt 中输出 handoff 模板 | 是 | 减少 AI "不知道写什么" 的问题 |
