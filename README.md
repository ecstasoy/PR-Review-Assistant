# PR-Review-Assistant

> 一个 **站在 reviewer 视角** 的 AI 辅助评审工具。粘贴任意 GitHub PR 链接，30 秒拿到结构化评审：**变更总结 / 风险识别 / 行内建议**。

<!-- TODO(PR #17): 在此插入 demo GIF 或截图 -->

## ✨ 核心能力

- **F1 PR 拉取**：粘 URL 即用，无需仓库写权限
- **F2 三层上下文**：diff hunk → 变更文件全文 → 项目约定文件（README/CONTRIBUTING/CLAUDE.md）
- **F3 三阶段并行 AI 分析**：总结 / 风险（severity 分级）/ 行内建议（按 hunk）
- **F4 流式响应**：SSE 推送，先到先显，首字节 < 3s
- **F5 SHA 级缓存**：同一 PR head_sha 不变时秒回

详见 [docs/PRD.md](./docs/PRD.md) · 排期见 [docs/PLAN.md](./docs/PLAN.md)

## 🎬 Demo 视频

<!-- TODO(PR #17): 上传至 B 站后替换链接 -->
*将在 Day 3 补充*

## 🚀 快速开始

> 需要 Go 1.22+、Node 20+、pnpm（或 npm）

```bash
# 1. 拉代码
git clone <repo-url> && cd PR-Review-Assistant

# 2. 配置环境变量（可选；不配走 mock 模式）
cp backend/.env.example backend/.env
# 编辑 backend/.env 填入 OPENAI_API_KEY / OPENAI_BASE_URL / GITHUB_TOKEN

# 3. 一键启动
make dev
# 前端: http://localhost:3000
# 后端: http://localhost:8080
```

<!-- TODO(PR #6): 补充 make dev 实现细节、端口可配置说明 -->

## 🧩 架构概览

```
┌──────────────┐        ┌──────────────────────────────────┐
│ Next.js UI   │ ──SSE→ │ Go Backend (chi)                 │
│ (frontend)   │        │  ├─ internal/github   PR 抓取    │
└──────────────┘        │  ├─ internal/context  三层上下文 │
                        │  ├─ internal/llm      Provider   │
                        │  ├─ internal/review   并行调度   │
                        │  └─ internal/cache    SQLite     │
                        └──────────────────────────────────┘
                                  │
                            ┌─────┴──────┐
                            ↓            ↓
                       GitHub API   LLM Provider
```

## 🧠 模型选择

<!-- TODO(PR #16): 详细对比 DeepSeek / OpenAI / Claude / 通义 等候选模型 -->

- **默认 Provider**：DeepSeek（OpenAI 兼容、便宜、国内可达）
- **抽象层**：`internal/llm.Provider` 接口（`Complete` / `Stream`），切换只改环境变量
- **降级**：未配 key 时自动使用 mock provider，保证演示可复现
- **评估维度**（待补全表格）：代码理解 / 上下文窗口 / 中文质量 / 价格

## 🔍 上下文获取策略

<!-- TODO(PR #16): 详述三层裁剪算法与 token 预算分配 -->

| 层级 | 来源 | 用途 |
|---|---|---|
| L1 | diff hunk | 最低保证，永远包含 |
| L2 | 变更文件全文 / 受影响函数 | 提升代码理解 |
| L3 | 项目约定文件 | 风格契合 |

Token 预算默认按 **L1:L2:L3 = 4:5:1** 分配；超限时按 L3 → L2 → L1 顺序压缩。

## 🛣️ 未来扩展

<!-- TODO(PR #16): 展开每一项的实现路径 -->

1. **多模型 A/B 对比**：同 PR 跑多模型，结果并排展示
2. **GitHub App 化**：从 Web 工具升级为 PR 自动评论 bot
3. **代码库 RAG**：跨文件检索定义，借鉴 Greptile
4. **多 agent 自验证**：风险识别二次校验过滤误报，借鉴 Anthropic Claude Code Review
5. **自定义规则注入**：用户上传 review 规范
6. **CI 集成**：CLI 模式 + GitHub Actions 产出报告

## 📦 依赖与原创说明

> 题目合规要求

### 第三方依赖

**后端（Go）**
- `github.com/go-chi/chi/v5` — HTTP 路由
- `github.com/google/go-github/v66` — GitHub REST 客户端
- `github.com/mattn/go-sqlite3` — 缓存存储
- `github.com/caarlos0/env/v11` — 环境变量加载

**前端（TypeScript）**
- `next` / `react` — 应用框架
- `tailwindcss` + `shadcn/ui` — 样式与组件
- `react-markdown` + `remark-gfm` — Markdown 渲染
- `react-diff-viewer-continued` — diff 视图

<!-- TODO(PR #16): 每次新增依赖时同步更新 -->

### 原创部分

以下均为本项目原创：

- 全部 Go 业务代码（GitHub 抓取、三层上下文构建、并行调度、SSE 流）
- 三阶段 prompt 模板（`backend/internal/prompts/*.tmpl`，go:embed 进二进制）
- 前端 UI 组件与 SSE 集成
- token 预算裁剪策略

### 参考与致谢

- [qodo-ai/pr-agent](https://github.com/qodo-ai/pr-agent) — 多阶段 prompt 设计思路
- CodeRabbit — 严重度分级与行内 comment 排版灵感
- [Greptile](https://www.greptile.com/) — 跨文件上下文检索（未来扩展方向）
- Anthropic Claude Code Review — 多 agent 验证模式（未来扩展方向）

> **没有任何代码片段是从上述项目直接拷贝的**；仅在架构和 prompt 拆分思路层面借鉴。

## 👥 团队与分工

<!-- TODO(PR #16): 单人项目可简写；多人项目按队员账号列出 PR 范围 -->

- 单人开发

## 📄 License

[MIT](./LICENSE)
