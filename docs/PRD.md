# AI PR Review 助手 — 产品需求文档（PRD）

> 单人 · 72h · Web · Go 后端 + Next.js 前端 · 本地运行优先
> 配套排期见 [PLAN.md](./PLAN.md)

---

## 1. 产品概述

一个**站在 reviewer 视角**的 AI 辅助评审工具。用户粘贴任意 GitHub PR 链接（无需仓库权限），系统自动拉取 diff、扩展上下文、调用 LLM 输出三类信息：

1. **PR 变更总结** — 一句话讲清这个 PR 在做什么
2. **风险代码识别** — 按严重度标记可疑文件/片段
3. **行内 Review 建议** — 给出可执行的修改建议

用户拿到产出后可直接照搬到 GitHub PR 评论里，或作为自己评审的辅助材料。

## 2. 目标用户与典型场景

| 用户 | 痛点 | 本工具解决方式 |
|---|---|---|
| Reviewer / Maintainer | 接到不熟悉模块的 PR | 结构化摘要 + 风险点 |
| Tech Lead | 抽查 PR 但没时间通读 | 风险清单优先看高分项 |
| 开源贡献者 | 提 PR 前自检 | 自己粘自己 PR 链接预审 |
| 学习者 | 想读懂著名开源项目的 PR | 粘 URL 看 AI 解读 |

**与 PR-Agent / CodeRabbit 的关键差异**：它们是"装到仓库里的 bot"，需要仓库写权限；本工具是"独立 Web 工具"，**任意 PR URL 即用**，特别适合 reviewer 视角和学习场景。

## 3. 核心功能（MVP）

### F1 PR 拉取
- 输入：GitHub PR URL（公开仓库）
- 解析 `owner/repo/pr#`，调 GitHub REST 拉 PR meta + files diff
- 可选配置 PAT 以提高 rate limit / 支持私有仓
- 边界：单 PR 文件数 > 50 或 diff > 100KB 时给出"超大 PR"提示并降级处理

### F2 三层上下文构建
| 层级 | 来源 | 用途 |
|---|---|---|
| L1 | diff hunk | 永远包含，token 预算最低保证 |
| L2 | 变更文件全文 | 仅对中小文件拉取；超大文件抽取受影响函数 |
| L3 | 项目约定（README / CONTRIBUTING / CLAUDE.md / AGENTS.md） | 注入系统提示，提升风格契合度 |

Token 预算按 **L1:L2:L3 = 4:5:1** 分配；超出时按 L3 → L2 → L1 顺序压缩。

### F3 三阶段 AI 分析（结构化输出）
| 阶段 | Prompt 角色 | 输出格式 |
|---|---|---|
| 总结 | "用一段中文讲清这个 PR 在做什么" | Markdown 文本 |
| 风险识别 | "列出风险文件及原因，分级 high/medium/low" | JSON `[{file, severity, reason}]` |
| 行内建议 | "对每个 hunk 给出可执行修改建议" | JSON `[{file, line, type, suggestion}]` |

三阶段**并行调用**（不是 PR-Agent 那种串行），降低端到端延迟。

### F4 前端展示
- 顶部：总结卡片（markdown 渲染）
- 中部：风险清单（按 severity 排序，点击跳到对应文件）
- 底部：文件级 diff 视图 + 行内 AI 注释气泡
- 加载态：SSE 流式推进，先到先显（总结最快到，建议最慢到）

### F5 缓存
- 按 `repo:pr_number:head_sha` 作为缓存 key，本地 SQLite 存
- 同一 PR 在 head_sha 不变时秒回，节省费用 + 演示加分

## 4. 与参考项目的关系

| 项目 | 借鉴点 | 我的不同点 |
|---|---|---|
| [PR-Agent](https://github.com/qodo-ai/pr-agent) | 多阶段 prompt 拆分思路、token 自适应裁剪 | 用 Go 重写、三阶段并行、独立 Web UI 而非 bot |
| CodeRabbit（商业） | 严重度分级、行内 comment 排版 | 不依赖 GitHub App 权限，本地优先 |
| [Greptile](https://www.greptile.com/) | 跨文件上下文检索 | 未来扩展方向，MVP 不做 RAG |
| Anthropic Claude Code Review | 多 agent 验证 → 降低误报 | 写进"未来扩展"，MVP 不做 |

**原创承诺**：所有 Go 代码、前端组件、prompt 模板均为本项目原创；架构思想公开致谢。

## 5. 技术设计

### 5.1 模型选型
- **MVP**：单一 Provider，使用 OpenAI 兼容协议（OpenAI / DeepSeek / 通义千问 / Moonshot 任选）
- **代码层抽象**：`pkg/llm.Provider` 接口（`Complete` / `Stream`），便于切换
- **选型理由**（README 详写）：
  - 代码理解能力（HumanEval / SWE-bench）
  - 上下文窗口（≥ 32K 才能容纳中等 PR）
  - 中文输出质量
  - 价格（演示成本）
- **降级**：key 未配时自动 mock 一段示例输出，保证本地可跑通

### 5.2 架构

```
┌──────────────┐        ┌──────────────────────────────────────┐
│ Next.js UI   │ ──SSE→ │ Go Backend (gin)                     │
│ (frontend)   │        │  ├─ internal/github   PR 抓取        │
└──────────────┘        │  ├─ internal/prctx    三层上下文构建 │
                        │  ├─ internal/llm      Provider 抽象  │
                        │  ├─ internal/review   三阶段并行调度 │
                        │  └─ internal/store    SQLite 缓存+历史│
                        └──────────────────────────────────────┘
                                  │
                            ┌─────┴──────┐
                            ↓            ↓
                       GitHub API   LLM Provider
```

### 5.3 数据流
1. 前端 POST `/api/review` { url, options }
2. Backend 校验 URL → 查缓存 → 命中直接 SSE 推回
3. 未命中：拉 PR → 构建三层上下文 → fan-out 三阶段 LLM 调用
4. 每个阶段完成立即 SSE 推一帧，前端边收边渲染
5. 全部完成后写缓存

## 6. 非功能性需求

| 维度 | 目标 |
|---|---|
| 响应速度 | 首字节 < 3s，总结到达 < 8s，全部完成 < 25s（中等 PR） |
| 误报控制 | 风险识别仅输出 high/medium 给前端，low 折叠 |
| 漏报控制 | 三阶段独立调用，不共享上下文以减少相互污染 |
| 可观测 | 后端日志记录 prompt 大小 / 调用耗时 / token 用量 |
| 可移植 | 单二进制 + 静态前端，`make dev` 一键起 |

## 7. 未来扩展（README 详写章节）

1. **多模型 A/B**：同一 PR 跑不同模型，前端并排对比
2. **GitHub App 化**：把 Web 工具改造成 PR 自动评论 bot
3. **代码库 RAG**：超出当前 PR 文件范围拉相关定义（参考 Greptile）
4. **多 agent 自验证**：风险识别后再起一个 agent 验证，过滤误报（参考 Anthropic Claude Code Review）
5. **自定义规则**：用户上传 review 规范，注入 prompt
6. **CI 集成**：暴露 CLI 模式，在 GitHub Actions 里产出报告

## 8. 验收标准

- [ ] 粘贴任意公开 GitHub PR URL，25 秒内出齐总结 + 风险 + 建议
- [ ] 至少在 3 个真实公开 PR 上跑通（不同语言、不同体量）
- [ ] 关 key 时降级 mock 模式仍能演示完整 UI 流程
- [ ] README 覆盖：依赖列表、运行步骤、模型选择、上下文策略、未来扩展、致谢
- [ ] Demo 视频 ≤ 5 分钟，语音讲解三大能力 + 一个失败案例的处理
- [ ] 全部 commit 时间戳落在 72h 窗口内，PR 描述符合题目规范

## 9. 不做的事（YAGNI）

- ❌ 用户系统 / 登录
- ❌ 团队协作 / 评论保存
- ❌ Webhook / GitHub App 集成（写进未来扩展）
- ❌ 多语言 UI（中文为主，英文 README）
- ❌ RAG / 向量库（写进未来扩展）
- ❌ 私有部署文档（本地 demo 优先）
