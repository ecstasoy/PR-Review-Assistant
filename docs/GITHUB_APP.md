# GitHub App 申请与配置指引

> 本文给项目维护者用。v3 主线④（Webhook + OAuth）上线前需要先有一个 GitHub App。
> 走完本文 6 步后你会拿到 4 个凭证：`App ID` / `Client ID` / `Client Secret` / `Webhook Secret` / `Private Key (PEM)`。

## 0. 准备

- 你的部署域名（如 `prra.fly.dev` 或自有域名）—— 决定回调 URL
- GitHub 个人账号 或 组织 owner 权限（用来托管 App）

## 1. 编辑 manifest

[`docs/github-app-manifest.yml`](./github-app-manifest.yml) 是 App 模板。**重要：申请前先改 3 处**：

```yaml
redirect_url: https://prra.example.com/auth/github/callback
hook_attributes:
  url:        https://prra.example.com/api/webhook/github
setup_url:    https://prra.example.com/setup
```

把 `prra.example.com` 替换成你的真实域名（暂用 `prra.fly.dev` 也行）。

## 2. 用 manifest 创建 App

两种方式任选：

### 方式 A：手动 portal 申请（最简）

打开 https://github.com/settings/apps/new 按 manifest 字段逐项填写。完成后跳第 3 步。

### 方式 B：manifest 一键创建（更准）

写一个临时 HTML 页面，POST 到 GitHub「Create App from Manifest」端点。
本项目暂未提供这种自动化脚本（v3-G1 后续 PR 补 `cmd/github-app-bootstrap`）；
现阶段走方式 A。

## 3. 拿到的凭证

App 创建后在 GitHub App 设置页（https://github.com/settings/apps/&lt;your-app&gt;）能看到：

| 凭证 | 在哪 | 对应 env |
|---|---|---|
| **App ID** | "About" 节，数字 ID | `GITHUB_APP_ID` |
| **Client ID** | "About" 节 | `GITHUB_OAUTH_CLIENT_ID` |
| **Client secret** | "About" 节 → Generate a new client secret（只显示一次！） | `GITHUB_OAUTH_CLIENT_SECRET` |
| **Webhook secret** | "Webhook" 节 → 自定义一串随机字符串填入 | `GITHUB_APP_WEBHOOK_SECRET` |
| **Private key** | 页面底部 "Private keys" → Generate a private key → 下载 `.pem` 文件 | `GITHUB_APP_PRIVATE_KEY`（PEM 内容或路径） |

## 4. 注入到部署

### Fly.io

```bash
flyctl secrets set \
  GITHUB_APP_ID=123456 \
  GITHUB_OAUTH_CLIENT_ID=Iv1.abcd1234 \
  GITHUB_OAUTH_CLIENT_SECRET=ghos_xxxxx \
  GITHUB_APP_WEBHOOK_SECRET=$(openssl rand -hex 32) \
  GITHUB_APP_PRIVATE_KEY="$(cat path/to/your-app.private-key.pem)"
```

`GITHUB_APP_PRIVATE_KEY` 也支持传文件路径（v3-G1 后续 PR 让 backend 自动判别）。

### 本地开发 (`backend/.env`)

```env
GITHUB_APP_ID=123456
GITHUB_APP_PRIVATE_KEY=/path/to/your-app.private-key.pem
GITHUB_APP_WEBHOOK_SECRET=replace-with-your-own
GITHUB_OAUTH_CLIENT_ID=Iv1.abcd1234
GITHUB_OAUTH_CLIENT_SECRET=ghos_xxxxx
```

## 5. 自己装到测试仓库

App 创建后在 `https://github.com/apps/&lt;your-app-slug&gt;` 点 Install → 选目标仓库（建议先装自己的测试仓库，不要装到生产仓库直到 v3-G3 G4 完整跑通）。

## 6. 验证 webhook 联通

部署 backend 后到 GitHub App 设置页 → "Advanced" → "Recent Deliveries"
能看到 GitHub 往你 `hook_attributes.url` 发的 ping payload。如果显示 ❌，检查：

- 域名是否能从公网访问
- 是否 HTTPS（GitHub 强制）
- 后端是否真的处理 `/api/webhook/github`（v3-G3 后才有）
- Webhook Secret 是否两边一致

---

## 权限说明（最小集理由）

| 权限 | 为啥需要 | 不要更多 |
|---|---|---|
| `contents: read` | 读 PR diff 必需 | write 不必 |
| `metadata: read` | GitHub 强制最低权限 | — |
| `pull_requests: write` | 写 review comment 必需 | 不需要 admin |
| `checks: read` | 读 CI 状态显示在顶栏 | write 不必 |
| `issues: read` | 评论里偶尔引用 issue 上下文 | 可去掉，到时再说 |

**永远不要要求 `repo` 范围的 write** —— 评审工具不应该能改用户代码。

## 撤回 / 重置

- 改 webhook URL：App 设置页直接改
- 重置 webhook secret：「Edit」面板生成新 secret，旧 secret 立即失效
- 重置 private key：「Generate a private key」生成新 .pem，旧 key 仍然 valid（GitHub 不立即吊销）→ 在「Private keys」里手动 delete 旧 key
- 重置 client secret：「Generate a new client secret」立即失效旧 secret
- 卸载 App：用户在仓库 Settings → Apps → Configure → Uninstall

## v3 后续 PR 怎么接

- **v3-G2 OAuth login flow**：前端跳 `https://github.com/login/oauth/authorize?client_id=$CLIENT_ID&redirect_uri=$REDIRECT`；后端 callback handler 拿 code 换 user-to-server token；用 token 调 `/user` 拿用户身份
- **v3-G3 Webhook receiver**：`POST /api/webhook/github` 验签（HMAC-SHA256 with WEBHOOK_SECRET）+ 解析 PR 事件 + 入队
- **v3-G4 Installation token**：用 App private key 签 JWT → 换 installation token → 用该 token 调 GitHub API（替代 PAT）
- **v3-G5 webhook 投递到 MQ**：webhook handler 只入队不评审，后台 worker 消费 → 评审完用 installation token 写 review comment
