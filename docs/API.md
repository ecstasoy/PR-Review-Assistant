# API 接口文档

> 后端 base URL：`http://localhost:8080`（本地默认；通过 `PORT` env 改端口）
> 前端 dev server 把 `/api/*` 重写到后端，浏览器调 `http://localhost:3000/api/*` 等价。

---

## POST /api/review

提交 PR URL，跑总结阶段，**同步返 JSON**。SSE 流式响应是独立扩展。

### 请求

```http
POST /api/review
Content-Type: application/json

{
  "url": "https://github.com/owner/repo/pull/123"
}
```

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `url` | string | 是 | GitHub PR 链接；允许带 `/files` 后缀和末尾斜杠；前后空白自动 trim |

### 响应

**成功** `200 OK`

```json
{
  "id": "abc123def456789...",
  "owner": "golang",
  "repo": "go",
  "pr": 42,
  "url": "https://github.com/golang/go/pull/42",
  "head_sha": "abc123def456789...",
  "title": "fix race in scanner",
  "summary": "这个 PR 修复了 scanner 中的竞态条件...\n- 关键文件：scanner.go\n- ...",
  "risks": [
    {
      "file": "scanner.go",
      "line": 42,
      "severity": "high",
      "category": "bug",
      "confidence": 0.92,
      "reason": "并发访问 mutable 字段未加锁"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | string | 评审标识。v1 等于 `head_sha`；store 落地后改 ULID |
| `owner` | string | PR 仓库 owner |
| `repo` | string | PR 仓库名 |
| `pr` | int | PR 编号 |
| `url` | string | 原始输入 URL |
| `head_sha` | string | PR head 提交 SHA |
| `title` | string | PR 标题 |
| `summary` | string | LLM 生成的 markdown 总结（一段概述 + 3-5 条要点） |
| `risks` | array | LLM 识别的风险清单；解析失败 / mock 模式下为 `[]` |

**risks 项字段**：

| 字段 | 类型 | 说明 |
|---|---|---|
| `file` | string | 文件路径 |
| `line` | int | 行号（可选；不确定时响应中省略） |
| `severity` | string | `high` / `medium` / `low` |
| `category` | string | `bug` / `security` / `perf` / `style` / `other` |
| `confidence` | float | 0-1，LLM 自评把握度；前端 ≥ 0.9 默认展开 |
| `reason` | string | 中文说明，≤ 80 字 |

**错误**

| Status | 触发条件 | 响应体 |
|---|---|---|
| 400 | 请求 body 非合法 JSON | `{"error":"invalid request body"}` |
| 400 | `url` 字段为空 | `{"error":"url is required"}` |
| 400 | `url` 不是合法 GitHub PR 链接 | `{"error":"invalid GitHub PR URL"}` |
| 500 | 总结阶段失败（模板 / Stream 同步错 / 流中错） | `{"error":"summary failed","detail":"..."}` |
| 502 | GitHub API 调用失败（网络、404、速率限制） | `{"error":"fetch upstream failed","detail":"..."}` |

**注意**：风险识别阶段失败**不致命** —— 服务器记 warn 日志后照常返 200，`risks` 字段为空数组。这是为了让 mock 模式（无法产 JSON）能演示总结输出。生产部署应监控 warn 日志检测 risks 频繁失败。

### 性能与限制

- 单 PR 文件上限 **100**（一页拉到上限，超出由 prctx 层裁剪后续 PR 处理）
- 同步返回 = 等 LLM 全部生成完才响应。典型耗时 10-25s（取决于 PR 大小 / 模型）。SSE 升级后首字节 < 3s
- 默认 `LLM_PROVIDER=mock` 不调真实 LLM，无 key 也能跑（演示用）

### 示例

```bash
curl -X POST http://localhost:8080/api/review \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://github.com/golang/go/pull/12345"}'
```

---

## GET /api/health

存活探针。前端 / 监控用来确认后端已起。

### 请求

```http
GET /api/health
```

### 响应

**成功** `200 OK`

```json
{ "status": "ok" }
```

---

## GET /api/reviews

历史评审分页列表。**未实现**，后续 PR 落地（依赖 store 模块接通）。

预期形态：

```http
GET /api/reviews?limit=20&cursor=<ulid>
```

返回最近 `limit` 条 review 摘要（按 created_at desc）。

---

## GET /api/reviews/:id

按 id 取单条评审详情。**未实现**，后续 PR 落地。

预期形态：

```http
GET /api/reviews/abc123def456
```

返回完整的 `summary` + `risks[]` + `suggestions[]`。

`?live=1` 模式时改走 SSE 推送（等流式升级 PR）。

---

## SSE 事件协议（设计中）

将来 `/api/review` 升级到 SSE 后的事件 schema。`Content-Type: text/event-stream`，每条事件按 `event: <type>\ndata: <json>\n\n` 输出。

| event type | data schema | 含义 |
|---|---|---|
| `summary_delta` | `{"delta": "增量文本"}` | 总结阶段一帧 markdown 输出 |
| `risks_done` | `[{"file","line?","severity","category","reason"}]` | 风险识别阶段完成 |
| `suggestions_done` | `[{"file","line","type","suggestion"}]` | 行内建议阶段完成 |
| `error` | `{"stage":"summary\|risks\|suggestions","message":"..."}` | 某阶段中途错误 |
| `done` | `{"stage":"summary\|risks\|suggestions"}` | 某阶段完成 |

前端用原生 `EventSource` 订阅，按 type 路由到对应 UI 区域独立刷新。
