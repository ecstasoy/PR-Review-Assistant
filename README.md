# PR-Review-Assistant

> 一个 **站在 reviewer 视角** 的 AI 辅助评审工具。粘贴任意 GitHub PR 链接，30 秒拿到结构化评审：**变更总结 / 风险识别 / 行内建议**。

## ✨ 核心能力

- **F1 PR 拉取**：粘 URL 即用，无需仓库写权限；自动带回 meta（作者 / 角色 / state / labels / refs / stats）+ CI 状态 + 完整文件 diff
- **F2 三层上下文**：diff hunk → 变更文件全文 / 受影响函数 → 项目约定文件（README / CONTRIBUTING / CLAUDE.md），按预算自动裁剪
- **F3 三阶段并行 AI 分析**：总结 / 风险（severity × category 两维分级）/ 行内建议（按 hunk 出可应用 patch），三阶段独立调用、SSE 流式回吐
- **F4 流式响应**：SSE 推送，summary 边生成边渲染；首字节 < 3s（mock 模式 < 100ms）
- **F5 SHA 级缓存**：cache key = `owner/repo:pr:head_sha`；head_sha 不变时秒回

## 🎬 Demo 视频

> 60 秒演示从粘 URL 到看到三阶段评审结果的完整流程；评审历史 + 缓存秒回 + 会话视图 5 步时间线。
> 链接将在 demo 视频 PR 上线后回填。

## 🚀 快速开始

> 需要：**Go 1.22+** · **Node 20+** · **pnpm**（或 npm）

```bash
# 1. 拉代码
git clone <repo-url> && cd PR-Review-Assistant

# 2. 装依赖
make install
# 等同于：cd backend && go mod tidy; cd frontend && pnpm install

# 3. 配置环境（可选；不配走 mock 模式可直接演示）
cp backend/.env.example backend/.env
# 编辑 backend/.env：至少填 GITHUB_TOKEN；想用真实 LLM 还要填 OPENAI_API_KEY

# 4. 一键启动前后端
make dev
# 后端: http://localhost:8080  ·  前端: http://localhost:3000

# 5. 验证健康
curl http://localhost:8080/health   # → {"status":"ok"}
```

试跑：打开 http://localhost:3000，粘贴任意公开 PR 链接（如 `https://github.com/golang/go/pull/12345`）→ 落地页提交后跳 `/review/streaming?url=…`，SSE 逐步出总结 / 风险 / 建议。

## ⚙️ 环境配置

后端启动时由 `godotenv` 自动从 `backend/.env` 加载；生产环境直接用 process env，无需 `.env` 文件。

| 变量 | 默认 | 说明 |
|---|---|---|
| `PORT` | `8080` | 后端 HTTP 监听端口 |
| `GITHUB_TOKEN` | _空_ | GitHub PAT；不配走匿名（60 req/h + 无法读私库） |
| `LLM_PROVIDER` | `mock` | `mock`（无 key 演示）或 `openai`（任意 OpenAI 兼容 endpoint） |
| `OPENAI_API_KEY` | _空_ | `LLM_PROVIDER=openai` 时必填；缺失 → 自动降级 `mock` + WARN 日志 |
| `OPENAI_BASE_URL` | `https://api.deepseek.com` | DeepSeek / OpenAI / Kimi / 通义 任选 |
| `LLM_MODEL` | `deepseek-chat` | 与 BASE_URL 匹配的模型名 |
| `SQLITE_PATH` | `./data/reviews.db` | SQLite 文件路径；父目录不存在自动建 |

**部署环境凭证管理**：

| 平台 | 注入方式 |
|---|---|
| Fly.io | `fly secrets set OPENAI_API_KEY=...` |
| GitHub Actions | Repo Secrets → workflow `env: ... ${{ secrets.X }}` |
| Docker | `docker run -e OPENAI_API_KEY=...` 或 `--env-file` |
| Vercel | 项目设置 → 环境变量 |

## 📡 API 路由

后端 Gin router，路径不带前缀（前端 `NEXT_PUBLIC_BACKEND_URL` 拼）。

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/health` | 健康检查；返回 `{"status":"ok"}` |
| `POST` | `/review` | 提交 PR URL，SSE 流式回吐 `pr / files / summary_delta / risks_done / suggestions_done / stage_done / info / error / done` 事件 |
| `GET` | `/reviews?limit=N` | 历史评审列表（最近 N 条，默认 100），含 owner / repo / pr / head_sha / title / created_at / ci / risk_counts / lang |
| `GET` | `/reviews/:id` | 单条评审详情；含完整 PR meta + files + summary + risks + suggestions |

SSE 事件协议详见 `frontend/lib/sse.ts` 与 `backend/internal/api/review.go`。

## 🧩 架构概览

```
┌──────────────────────────┐         ┌────────────────────────────────────────┐
│ Next.js 15 App Router    │ ──SSE→  │ Go Backend (Gin)                       │
│ • app/(main)/ landing+   │         │  ├─ internal/api       HTTP + SSE       │
│   history (chrome wrapped) │         │  ├─ internal/github   PR meta+diff+CI │
│ • app/review/[id]/        │         │  ├─ internal/prctx    三层上下文构建    │
│   report / diff / session │         │  ├─ internal/prompts  go:embed 模板    │
│ • lib/sse.ts SSE 客户端    │         │  ├─ internal/llm      Provider 接口    │
│ • Tailwind v4 + tokens    │         │  ├─ internal/review   3 阶段并行调度    │
│                          │         │  ├─ internal/store    SQLite 缓存       │
└──────────────────────────┘         │  ├─ internal/index    L4 索引占位（v3） │
                                     │  └─ internal/agent    agent 循环占位     │
                                     └────────────────────────────────────────┘
                                                  │
                                          ┌───────┴────────┐
                                          ↓                ↓
                                    GitHub REST API   LLM Provider
                                                       (DeepSeek/OpenAI/Mock)
```

## 🧠 模型选择

- **抽象层**：`internal/llm.Provider` 接口（`Complete` / `Stream`），运行时按 env 选；切换 Provider / 模型不改业务代码
- **两个实现**：
  - `mock` — 写死回复，不发网络请求；用于本地开发、CI、演示与 `OPENAI_API_KEY` 缺失时的兜底
  - `openai` — 任意 OpenAI 兼容 endpoint（DeepSeek / OpenAI / Kimi / 通义），SSE 流式拉取
- **默认 Provider**：DeepSeek `deepseek-chat`，理由：OpenAI 协议兼容 / 国内可达 / 单价低、上下文窗口 64k 足够覆盖三层裁剪
- **按阶段差异化**（设计预留，v2 落地）：summary / suggestions 用便宜档（`deepseek-chat`）匹配生成型任务；risks 用 reasoner 档（`deepseek-reasoner`）匹配判断型任务。当前 v1 统一用 `deepseek-chat` 节省成本
- **降级**：`LLM_PROVIDER=openai` 但 `OPENAI_API_KEY` 缺失时，启动期日志告警并自动切回 `mock`，演示永不开天窗

## 🔍 上下文获取策略

题目要求"说明上下文获取方式"，核心思路：**评审质量 ≈ 上下文质量**，diff hunk 远远不够。本项目按三层裁剪 + 显式预算管理，避免 prompt 爆炸又留住信号。

| 层级 | 来源 | 用途 | 默认预算 |
|---|---|---|---|
| **L1** | PR meta + diff hunk | 最低保证，永远完整保留 | 40% |
| **L2** | 变更文件全文 / 受影响函数 | 让模型理解"改的这块在整体里干嘛" | 50% |
| **L3** | 项目约定文件（README / CONTRIBUTING / CLAUDE.md） | 风格契合 / 命名规范 / 提交习惯 | 10% |
| L4 *(v3)* | 跨文件 def/ref RAG 检索 | 解决"我改的函数在别处被怎么调用" | 异步索引后启用 |

**算法**：`internal/prctx.LayeredBuilder`（`backend/internal/prctx/layered.go`）

1. 先估 L1 token 数 (`estimateTokens(s) = len(s) / 3`，按 OpenAI tokenizer 实际比近似)
2. L3 按 `TokenLimit / 10` 分配，超额按字符截断
3. L2 拿到 `TokenLimit - L1 - L3`，按文件大小排序后逐文件填，直到预算耗尽；放不下的文件路径进 `BudgetReport.Dropped`
4. **floor 保护**：L2 至少留 `floorL2Tokens`（默认 1500），避免 L1 / L3 把 L2 挤光
5. 返回 `Context{L1Str, L2Files, L3Str, BudgetReport}`，三阶段 LLM 调用复用同一份

**压缩顺序**：L3 → L2 → L1。L1 是最后被砍的（diff hunk 一旦丢失评审就无意义）。

## 🛣️ 未来扩展

1. **多模型 A/B 对比**：同 PR 跑多模型，结果并排展示
2. **GitHub App 化**：从 Web 工具升级为 PR 自动评论 bot（webhook + Installation token + sticky comment 回灌）
3. **代码库 RAG（异步索引）**：跨文件检索定义补全 L4 上下文，借鉴 Greptile；首次评审同步跑 L1-L3 即返，索引任务投递消息队列由后台 worker 处理（embedding 全仓 ~分钟级），第二次评审起命中 RAG；MQ 选型 v3 用 Redis Streams 或嵌入式 NATS，不破坏 SSE 流式体验
4. **多 agent 自验证**：风险识别二次校验过滤误报，借鉴 Anthropic Claude Code Review
5. **自定义规则注入**：用户上传 review 规范文档，挂到 L3 与 risks/suggestions prompt 之间
6. **真部署**：v1 SQLite/内存 → PG/Redis；PAT → OAuth 登录；store seam 已设计成 Interface 可切换
7. **agent 工具调用版**：从一次 fan-out 升级为多轮 tool-using（grep / read_file / list_dir），可观察、可引导

## 📦 依赖与原创说明

> 题目合规要求

### 第三方依赖

**后端（Go）**

- `github.com/gin-gonic/gin` — HTTP 路由 + 中间件
- `github.com/google/go-github/v66` — GitHub REST 客户端
- `github.com/mattn/go-sqlite3` — 评审记录缓存
- `github.com/oklog/ulid/v2` — 评审记录 ID（时间排序，比 UUID 短 10 字符）
- `github.com/caarlos0/env/v11` — 环境变量加载（带类型 / tag 校验）
- `github.com/joho/godotenv` — `.env` 文件加载

**前端（TypeScript / pnpm）**

- `next` 15 + `react` 19 — App Router + Server / Client Components
- `tailwindcss` v4 + `@tailwindcss/postcss` — 样式（`@theme` 直接定义 tokens）
- `lucide-react` — 图标（替代手写 SVG）
- `class-variance-authority` + `clsx` + `tailwind-merge` — 组件变体系统
- `highlight.js` — 50+ 语言语法高亮（diff 视图）
- `react-markdown` + `remark-gfm` — Markdown 渲染（summary / risks reason）
- `react-diff-viewer-continued` — diff 视图基础

### 原创部分

以下均为本项目原创：

- 全部 Go 业务代码（GitHub 抓取、三层上下文构建、并行调度、SSE 协议）
- 三阶段 prompt 模板（`backend/internal/prompts/*.tmpl`，go:embed 进二进制）
- 全部前端组件树（landing / history / review 三视图 + AgentSession 5 步时间线 + 设计 tokens）
- token 预算裁剪策略（`LayeredBuilder` + `BudgetReport`）
- 主语言检测（`detectPrimaryLang`，按文件数多数派 + lockfile 黑名单 + tie-break 字母序）

### 参考与致谢

- [qodo-ai/pr-agent](https://github.com/qodo-ai/pr-agent) — 多阶段 prompt 设计思路
- CodeRabbit — 严重度分级与行内 comment 排版灵感
- [Greptile](https://www.greptile.com/) — 跨文件上下文检索（未来扩展方向）
- Anthropic Claude Code Review — 多 agent 验证模式（未来扩展方向）

> **没有任何代码片段是从上述项目直接拷贝的**；仅在架构和 prompt 拆分思路层面借鉴。

## 👥 团队与分工

- 单人开发

## 📄 License

[MIT](./LICENSE)
