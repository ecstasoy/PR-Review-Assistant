# API 接口文档

> 后端 base URL：`http://localhost:8080`（本地默认；通过 `PORT` env 改端口）
> 前端 dev server 把 `/api/*` 重写到后端，浏览器调 `http://localhost:3000/api/*` 等价。

---

## POST /api/review

提交 PR URL，**预检通过后切到 SSE 流**，按帧推送各 stage 事件。

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

**两段语义**：

- **预检失败**（URL 错 / body 错 / GitHub 拉取失败）→ 普通 JSON 响应（4xx / 5xx），无 SSE
- **预检通过**（GitHub 已返 PR 数据）→ `200 OK` + `Content-Type: text/event-stream`，按 SSE 帧推送

### SSE 帧格式

每帧形如：

```
event: <type>
data: <JSON>

```

（两个换行符结尾。）

### 事件类型

| event type | data schema | 出现时机 |
|---|---|---|
| `pr` | `{ id, owner, repo, pr, url, head_sha, title }` | 首帧，GitHub 拉取成功后立刻发，让前端先渲头部 |
| `summary_delta` | `{ "delta": "增量文本" }` | summary 阶段一帧 markdown 输出，多帧拼接成完整 markdown |
| `risks_done` | `[{ file, line?, severity, category, confidence, reason }]` | risks 阶段完成（要么有 risks 要么空数组） |
| `error` | `{ "stage": "summary\|risks", "message": "..." }` | 某 stage 中途失败；不中止整条流 |
| `done` | `{}` | 所有 stage 完成，连接即将关闭 |

**risks 项字段**：

| 字段 | 类型 | 说明 |
|---|---|---|
| `file` | string | 文件路径 |
| `line` | int | 行号（可选；不确定时省略） |
| `severity` | string | `high` / `medium` / `low` |
| `category` | string | `bug` / `security` / `perf` / `style` / `other` |
| `confidence` | float | 0-1，LLM 自评把握度；前端 ≥ 0.9 默认展开 |
| `reason` | string | 中文说明，≤ 80 字 |

### 预检错误（不发 SSE）

| Status | 触发条件 | 响应体 |
|---|---|---|
| 400 | 请求 body 非合法 JSON | `{"error":"invalid request body"}` |
| 400 | `url` 字段为空 | `{"error":"url is required"}` |
| 400 | `url` 不是合法 GitHub PR 链接 | `{"error":"invalid GitHub PR URL"}` |
| 502 | GitHub API 调用失败（网络、404、速率限制） | `{"error":"fetch upstream failed","detail":"..."}` |

### 性能与限制

- 单 PR 文件上限 **100**（一页拉到上限，超出由 prctx 层裁剪后续 PR 处理）
- **首字节延迟 < 200ms**（`pr` 元信息帧立刻发），summary 一字一字流出，UX 远好于同步等 25s
- 默认 `LLM_PROVIDER=mock` 不调真实 LLM，无 key 也能跑（演示用，risks 阶段会发 `error` event）

### 客户端示例

**curl**（看原始 SSE 输出）：

```bash
curl -N -X POST http://localhost:8080/api/review \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://github.com/golang/go/pull/12345"}'
```

`-N` 关掉 curl 的输出缓冲，让帧实时显示。

**JavaScript fetch + ReadableStream**（`EventSource` 只支持 GET，POST + SSE 必须手动解析）：

```javascript
const res = await fetch("/api/review", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ url }),
});
const reader = res.body.getReader();
const decoder = new TextDecoder();
let buf = "";
while (true) {
  const { done, value } = await reader.read();
  if (done) break;
  buf += decoder.decode(value, { stream: true });
  const frames = buf.split("\n\n");
  buf = frames.pop();
  for (const f of frames) {
    // 解析 event: / data: 行，按 type dispatch
  }
}
```

完整封装见 `frontend/lib/sse.ts` 的 `streamReview`。

---

## GET /api/health

存活探针。前端 / 监控用来确认后端已起。

```http
GET /api/health
```

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

返回最近 `limit` 条 review 摘要（按 created_at desc），每项含 risks 严重度计数 `{high, medium, low}`。

---

## GET /api/reviews/:id

按 id 取单条评审详情。**未实现**，后续 PR 落地。

预期形态：

```http
GET /api/reviews/abc123def456
```

返回完整的 `summary` + `risks[]` + `suggestions[]`。

`?live=1` 模式时改走 SSE 推送（与 `POST /api/review` 同一套事件协议）。
