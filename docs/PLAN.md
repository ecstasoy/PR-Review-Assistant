# AI PR Review 助手 — 72h 开发排期

> 单人开发 · Web 形态 · Go 后端 + Next.js 前端 · 本地运行优先

## 总体策略

每天结束都要有"可演示"的版本。

- **Day 1**：端到端骨架打通（PR URL 进 → AI 总结出）
- **Day 2**：差异化（上下文扩展、风险识别、行内建议）
- **Day 3**：打磨 + 录像 + 文档

**模型选型**：先用单一 Provider（OpenAI 兼容协议，DeepSeek/OpenAI/Claude 任选最易拿 key 的），代码层抽象成 `LLMProvider` 接口，README 里写"未来支持多模型切换"。

**仓库结构**：monorepo，`/backend`（Go）+ `/frontend`（Next.js）+ `/docs`。每个 PR 只动一处。

---

## Day 1（0–24h）：端到端骨架打通

目标：PR URL 进 → 一段 AI 总结出。丑没关系，必须能跑。

| # | PR 标题 | 预估 | 内容 |
|---|---|---|---|
| 1 | chore: scaffold monorepo with Go backend and Next.js frontend | 2h | `/backend` go mod init + 一个 `/health` 路由；`/frontend` `create-next-app`；根 README 占位 |
| 2 | feat(backend): parse GitHub PR URL and fetch diff via GitHub API | 3h | `pkg/github`：解析 `owner/repo/pr#`，调 GitHub REST 拿 PR meta + files diff；先支持 PAT |
| 3 | feat(backend): add LLM provider abstraction with one implementation | 3h | `pkg/llm`：`Provider` 接口（`Complete`/`Stream`），实现 OpenAI 兼容客户端，env 读 key |
| 4 | feat(backend): /api/review endpoint that returns PR summary | 3h | 串起来：URL → diff → prompt → LLM → 返回 markdown 总结 |
| 5 | feat(frontend): minimal UI — paste URL, show summary | 3h | 单页面：输入框 + 按钮 + 渲染 markdown（用 `react-markdown`） |
| 6 | chore: add .env.example, run scripts, dev README | 1h | `make dev` 或 `pnpm dev` 一键起前后端 |

**Day 1 收尾自检**：随便贴一个公开 PR 链接，30 秒内能看到一段中文总结。

---

## Day 2（24–48h）：做出差异化

这是项目能不能拿名次的关键。多数同赛作品只读 diff hunk，你要读"上下文"。

| # | PR 标题 | 预估 | 内容 |
|---|---|---|---|
| 7 | feat(backend): expand context by fetching full file content for changed files | 3h | 不止读 diff hunk，关键文件拉全文（控制 token 预算，超大文件只取相关函数） |
| 8 | feat(backend): risk identification pass tagging files by severity | 3h | 第二轮 prompt：输出结构化 JSON（`[{file, severity, reason}]`），不和总结混 |
| 9 | feat(backend): per-hunk review comments as structured output | 3h | 第三轮 prompt：每个 hunk 给出 inline 建议（行号 + 类型 + 建议） |
| 10 | feat(frontend): split-view diff viewer with AI annotations | 4h | 左 diff 右注释；用 `react-diff-viewer` 或类似库；点文件跳转 |
| 11 | feat(backend): inject project conventions from README/CONTRIBUTING into prompt | 2h | 拉 repo 根的 README/CONTRIBUTING/CLAUDE.md，作为系统提示注入，提升"上下文理解" |
| 12 | feat(backend): stream LLM response via SSE | 2h | 改造 `/api/review` 用 SSE 推送，前端边收边渲染（"响应速度"加分） |

**Day 2 收尾自检**：找一个有真实问题的 PR（比如自己仓库里故意埋 bug），能看到风险标记 + 行内建议。

---

## Day 3（48–72h）：打磨 + Demo + 文档

最后一天**不要再加新功能**，只修 bug + 写文档 + 录像。

| # | PR 标题 | 预估 | 内容 |
|---|---|---|---|
| 13 | feat(backend): cache PR analyses by commit SHA | 2h | 本地文件或 SQLite 缓存，同一 PR 不重复花钱；演示秒回 |
| 14 | fix: error handling, rate limit, empty states | 3h | 错的 URL、私有仓、超长 diff、key 没配——都要给体面提示 |
| 15 | polish(frontend): visual cleanup, loading states, responsive | 3h | 配色、字号、骨架屏；不求美但求不丢分 |
| 16 | docs: write comprehensive README with architecture, model choice, context strategy | 3h | 见下面"README 必写章节" |
| 17 | docs: add demo PR examples and screenshots | 1h | 准备 2–3 个真实公开 PR 作为 demo 素材 |
| — | 录 demo 视频 | 3h | 脚本 → 试录 → 正式录 → 上传 B 站 → README 加链接 |
| — | 最终回归测试 + 缓冲 | 2h | 清状态、重跑、检查 commit 时间戳 |

---

## 风控（如果落后于排期）

按这个顺序砍：

1. **先砍** PR 10 的精美 diff 视图 → 退化成简单列表
2. 再砍 PR 12 流式 → 改成普通请求 + loading
3. 再砍 PR 11 项目约定注入
4. **绝对不能砍**：PR 7（上下文扩展）、PR 8（风险识别）、PR 16（README）、Demo 视频

---

## README 必写章节（评分点）

题目原话："**说明系统在模型选择、上下文获取方式及未来扩展方向上的设计思路**"。这三个必须各占一节：

- **模型选择**：为什么选这个 Provider，对比了什么，为什么抽象成接口
- **上下文获取**：diff hunk → 全文件 → 项目约定的三层策略，token 预算怎么分
- **未来扩展**：多模型对比、GitHub App、本地知识库 RAG、自定义 review 规则

---

## 提交节奏红线

题目明说"仅在最后一天一次性导入所有代码的作品，将直接视为无效作品"。所以：

- **Day 1 必须有 ≥ 4 个 PR 合入主分支**
- **Day 2 必须有 ≥ 4 个新 PR**
- **commit 时间戳全部落在 72h 窗口内**
- 每个 PR 描述照题目要求写齐：**标题 / 功能描述 / 实现思路 / 测试方式**
