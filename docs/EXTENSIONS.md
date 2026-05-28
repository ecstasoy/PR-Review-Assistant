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

- `backend/internal/prctx/builder.go` 加 `L4 References` 字段
- 新增 `backend/internal/index/`：负责构建/查询语义索引或符号图
- `prctx.Builder.Build` 在 budget 允许时拼入 L4

## 5. 多语言 UI

- `frontend/` 引入 `next-intl`
- App Router 加 `[locale]` 段：`app/[locale]/page.tsx` 等
- 文案集中到 `frontend/messages/{zh,en}.json`

## 6. Docker / CI

- 根目录加 `Dockerfile`（多阶段：Go builder + Node builder + 运行镜像）
- `.github/workflows/ci.yml`：`go vet`、`go test`、`pnpm build`
- 不改业务代码；只增打包/校验流程

## 7. 私有部署

- 根目录加 `docker-compose.yml`：app + sqlite 数据卷
- README 新增"自部署"章节

## 8. CLI 模式（GitHub Actions 集成）

- `backend/cmd/cli/main.go`：复用 `review.Orchestrator`，结果输出到 stdout (Markdown / JSON)
- `.github/workflows/review.yml` 模板：在 PR 上跑 CLI 并把结果以评论发回

---

每个扩展位的具体实现交由后续 PR；本文档仅声明"接口位已留"的承诺，便于评委理解架构延展性。
