# 扩展位 (Extension Slots)

> v1 把以下功能"留好接口位但不实现"。本文档列出每条延后功能对应的代码扩展点，v2 改造时**不需要破坏性改动**已稳定的接口。

## 1. OAuth 登录 / 用户系统

| 扩展位 | 当前状态 |
|---|---|
| `backend/internal/api/middleware/auth.go` | v1 no-op，把 `userID = nil` 塞进 `gin.Context` |
| `backend/internal/review/orchestrator.go` `Run(ctx, pr, userID *string)` | 签名已含 `userID`，v1 传 nil |
| `backend/internal/store/store.go` `Store.List(ctx, userID *string, ...)` | 接口已按 `userID` 过滤 |
| `backend/internal/store/schema.sql` `reviews.user_id` 列 | nullable，已建 `(user_id, created_at)` 索引 |
| `frontend/components/NavBar.tsx` "Sign in" 链 | 灰显，tooltip `Coming in v2` |

**v2 接入方式**：替换 `AuthCtx` 实现为 OAuth/session 校验；handler 自动拿到 userID；前端解禁 Sign in；schema 不需要迁移。

## 2. 团队协作 / 评论

- 新增 `backend/internal/store/schema.sql` 表 `comments`（外键到 `reviews.id`）
- 新增 `backend/internal/api/comments.go`
- 前端 `app/review/[id]/page.tsx` 末尾挂 `CommentsSection` 组件

## 3. GitHub App / Webhook 自动评审

- 新增 `backend/internal/api/webhook.go`（GitHub webhook 签名验证 + 事件路由）
- 复用 `internal/github.Fetcher` 与 `internal/review.Orchestrator`，无需新增业务模块
- README 加 GitHub App 安装指引

## 4. RAG / 代码图谱（L4 上下文）

### 已留接口
- `backend/internal/index/`：`Retriever` / `Embedder` / `Reference` / `NoopRetriever`
- `backend/internal/prctx/builder.go`：`Context.L4References` 字段，v1 永远 nil
- `prctx.Builder` 构造时接 `index.Retriever`；v1 注入 `NoopRetriever{}`

### v2 落地

**embedding 时机**：lazy。第一次评审某仓库时 embed 本 PR 涉及文件 + 相邻定义。比"提交时全量索引"省钱，可增量。

**存储选型**

| 候选 | 适用 |
|---|---|
| sqlite-vec（推荐） | 单二进制部署、< 1M chunk；与现有 SQLite 共存零新依赖 |
| Qdrant / pgvector | 多租户、共享后端 |
| 内存 FAISS-like | 演示快、重启丢 |

**token 预算**：当前 L1:L2:L3 = 4:5:1，引入 L4 改 3:4:1:2；`prctx/budget.go` 压缩顺序 L3 → L4 → L2 → L1。

**chunking**：v2 决定。候选：tree-sitter 按函数 / 类切；超长函数二次按行段。

## 5. Agent 循环 / 工具调用

### 已留接口
- `backend/internal/agent/`：`ToolSpec` / `Tool` / `Registry` / `Agent.Run`
- `backend/internal/review/orchestrator.go`：`Stage` 接口；`Orchestrator.Stages []Stage`
- `llm.Request` 未来加 `Tools []ToolSpec` 是新字段、向后兼容

### v2 工具目录草稿

| Tool | 用途 |
|---|---|
| `fetch_file(path)` | 拉仓库任意文件原文 |
| `search_symbol(name)` | 搜符号定义 / 引用 |
| `get_definition(file, line)` | 拿某符号的定义 |
| `query_index(query, k)` | 调 `index.Retriever`，与 RAG 共底 |
| `read_diff_hunk(file)` | 重读某文件的 diff hunk |

### 循环参数
- `MaxSteps = 5`（默认）防失控
- 单步预算：单次 chat completion ≤ 2k token
- 超限策略：返回部分结果 + 截断说明

### 哪个 stage 换 agent
- **risks** —— 风险识别天然分步（定位 → 验证 → 评级），agent 比单次 prompt 误报低
- **suggestions** —— 建议生成需要回看上下文，agent 也合适
- **summary** —— 一次过即可，不需要 agent

### 接入示例（v2 代码）

```go
reg := agent.NewRegistry()
reg.Register(&FetchFileTool{...})
reg.Register(&SearchSymbolTool{...})
risksAgent := agent.NewStage(reg, risksAgentPrompt, llmProvider)

o.Stages = []review.Stage{
    review.SummaryStage{},      // 不动
    risksAgent,                 // 换成 agent
    review.SuggestionsStage{},  // 不动
}
```

Orchestrator 一行不改。

## 6. 多语言 UI

- `frontend/` 引入 `next-intl`
- App Router 加 `[locale]` 段：`app/[locale]/page.tsx` 等
- 文案集中到 `frontend/messages/{zh,en}.json`

## 7. Docker / CI

- 根目录加 `Dockerfile`（多阶段：Go builder + Node builder + 运行镜像）
- `.github/workflows/ci.yml`：`go vet`、`go test`、`pnpm build`
- 不改业务代码；只增打包/校验流程

## 8. 私有部署

- 根目录加 `docker-compose.yml`：app + sqlite 数据卷
- README 新增"自部署"章节

## 9. CLI 模式（GitHub Actions 集成）

- `backend/cmd/cli/main.go`：复用 `review.Orchestrator`，结果输出到 stdout (Markdown / JSON)
- `.github/workflows/review.yml` 模板：在 PR 上跑 CLI 并把结果以评论发回

---

每个扩展位的具体实现交由后续 PR；本文档仅声明"接口位已留"的承诺，便于评委理解架构延展性。
