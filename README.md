# OpenCode Pool Gateway

当前版本：`0.6.0`

## API 转发网关

第三方客户端使用“令牌管理”页面生成的本站令牌作为 Bearer Token，请求本机或服务器上的 OpenCode Pool Gateway 地址。OpenCode Pool Gateway 完整保留请求路径、查询参数、请求体和业务请求头，仅将上游鉴权替换为选中凭证的 OpenCode API Key。

支持路由：

- `/zen/go/v1/chat/completions`
- `/zen/go/v1/responses`
- `/zen/go/v1/messages`
- `/zen/go/v1/models`
- `/zen/v1/responses`
- `/zen/v1/messages`
- `/zen/v1/chat/completions`
- `/zen/v1/models`
- `/zen/v1/models/<model-id>`（Gemini 原生协议）

Zen 模型协议对应关系：

| 协议 | 模型类型 | 路由 |
| --- | --- | --- |
| OpenAI Responses | GPT、Grok | `/zen/v1/responses` |
| Anthropic Messages | Claude、Qwen | `/zen/v1/messages` |
| Google Generative Language | Gemini | `/zen/v1/models/<model-id>` |
| OpenAI-compatible Chat Completions | DeepSeek、MiniMax、GLM、Kimi 等 | `/zen/v1/chat/completions` |

Go 模型协议对应关系：

| 协议 | 模型类型 | 路由 |
| --- | --- | --- |
| OpenAI Responses | GPT 5.6 Luna | `/zen/go/v1/responses` |
| Anthropic Messages | MiniMax M3/M2.7/M2.5、Qwen 3.8/3.7/3.6 | `/zen/go/v1/messages` |
| OpenAI-compatible Chat Completions | Grok、GLM、Kimi、DeepSeek、MiMo、Hy3 等 | `/zen/go/v1/chat/completions` |
| Models | Go 完整模型目录 | `/zen/go/v1/models` |

设置页可切换轮询负载均衡与优先级故障转移。最大尝试次数为 `0` 时，每次请求最多尝试所有可用凭证一次。代理支持 `http://`、`https://` 与带用户名密码认证的 `socks5://`，账号独立代理优先于全局代理。

SOCKS5 示例：`socks5://username:password@127.0.0.1:1080`。用户名或密码包含特殊字符时需要进行 URL 编码。

### 客户端接入

1. 在“令牌管理”页面创建本站访问令牌。
2. 将第三方客户端的 Base URL 指向 OpenCode Pool Gateway，例如 `http://127.0.0.1:8787/zen/v1`。
3. 将本站令牌填写为客户端 API Key。网关验证本站令牌后选择账号池中的 OpenCode API Key 请求上游，第三方令牌不会发送给 OpenCode。

Responses 示例：

```bash
curl http://127.0.0.1:8787/zen/v1/responses \
  -H "Authorization: Bearer gq_your_gateway_token" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-5.2","input":"Hello"}'
```

Anthropic Messages 示例：

```bash
curl http://127.0.0.1:8787/zen/v1/messages \
  -H "Authorization: Bearer gq_your_gateway_token" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet-4","max_tokens":256,"messages":[{"role":"user","content":"Hello"}]}'
```

Gemini 客户端应保留模型路径和动作后缀，例如 `/zen/v1/models/gemini-3.6-flash:generateContent`。查询参数、请求体及除鉴权外的业务请求头均原样透传。

OpenCode Pool Gateway 是一个使用 Go 标准库构建的 OpenCode Go / Zen 账号池管理与 API 转发网关。它不依赖 OpenCode CLI，可集中管理多个 Workspace，查询额度与模型，并向第三方客户端提供带鉴权、调度、故障转移和代理支持的兼容接口。

## 功能

- 批量管理多个 OpenCode Workspace。
- 分别识别 Go 订阅额度与 Zen 按量付费余额。
- 展示 Go 的 5 小时、每周、每月额度窗口和重置时间。
- 分别查询 Go 与 Zen 可用模型，不混用模型目录。
- 标记 Zen 已弃用模型并显示官方弃用日期。
- 账号卡片支持搜索、状态筛选、编辑、刷新和删除。
- Windows 控制台支持打开网页、刷新、帮助和安全退出。
- 数据与可执行文件放在同一目录，便于迁移和备份。
- 服务端登录验证、会话管理和登录失败限流。
- 使用 Workspace ID 与 Cookie 自动识别账号邮箱和本人 API Key。
- 一个账号存在多个 API Key 时可在添加页面选择使用的 Key。
- 支持轮询负载均衡和按数字升序的凭证优先级模式。
- 上游失败时自动切换其他凭证，每个凭证在单次请求中最多尝试一次。
- 支持全局代理与凭证独立代理，独立代理优先。
- 本站访问令牌仅保存摘要，第三方令牌不会透传给 OpenCode。
- 请求日志记录模型、凭证、Token、缓存、耗时和脱敏错误报文。
- 请求日志支持普通 JSON、gzip 和 SSE 流式响应的用量解析，耗时自动显示为毫秒、秒或分秒。
- 自动区分 Free、Go、Zen 和 Go + Zen 账号类型，并按模型权益筛选转发凭证。

## 模型权限

- Free 账号不能用于 Go 路由。
- Free 账号可以使用官方定价页标记为免费的 Zen 模型。
- Zen 非免费模型只会调度已开启 Zen 计费的凭证。
- Go 路由只会调度已开通 Go 的凭证。
- “模型”窗口会显示每个模型对当前凭证是否可用，以及不可用原因。

Zen 免费模型清单依据 OpenCode 官方定价页维护。官方可能调整免费期限或模型列表，升级版本时应同步检查。

## 快速开始

### Windows

下载 `opencode-pool-gateway-0.6.0-windows-amd64.exe`，双击运行，然后访问：

```text
http://localhost:8787
```

控制台命令：

```text
O  打开网页
R  刷新全部账号
Q  安全退出
H  显示帮助
```

首次启动时程序会创建 `data/auth.json`，控制台显示一次随机初始密码。登录后请在“设置 → 修改登录凭证”中修改用户名和密码。

### Linux

```bash
chmod +x opencode-pool-gateway-0.6.0-linux-amd64
./opencode-pool-gateway-0.6.0-linux-amd64
```

ARM64 Ubuntu 使用 `opencode-pool-gateway-0.6.0-linux-arm64`。服务器部署及 systemd 配置见 [docs/ubuntu.md](docs/ubuntu.md)。

## 登录安全

登录配置保存在可执行文件旁的 `data/auth.json`，优先级高于其他来源。密码使用随机盐和 PBKDF2-HMAC-SHA256 哈希保存，不写入明文。

文件不存在时，程序会创建用户名 `admin` 和随机密码，并在控制台或 systemd 日志中显示一次初始密码。登录后可在网页设置中修改；保存会立即热更新配置，并注销全部旧会话。

HTTPS 部署可设置 `OPG_COOKIE_SECURE=1`，强制浏览器只通过 HTTPS 发送会话 Cookie。

认证保护覆盖网页、静态资源和全部 `/api/` 接口。会话有效期为 24 小时；Cookie 使用 `HttpOnly` 和 `SameSite=Strict`。

## 添加账号

每个 Workspace 只添加一次，程序会同时检测 Go 与 Zen：

添加页面支持三种方式：

- `Workspace + Cookie`：可查询账号邮箱、API Key、Go 额度、Zen 余额和完整账号类型。
- `账号密码登录`：后台执行内置 GitHub OAuth 协议脚本，自动取得 Workspace ID、auth Cookie、账号邮箱和本人 API Key。需要 Python 3 与 `requests`。
- `仅 API Key`：适合只拿到 API Key 的场景。程序会使用 `deepseek-v4-flash` 分别请求 `/zen/go/v1/chat/completions` 与 `/zen/v1/chat/completions`；若响应返回 `CreditsError`，会从付款方式链接自动提取 Workspace ID，并将其作为默认显示名称。

账号密码登录会按脚本定义的 `账号----密码----邮箱密码` 参数协议运行。程序在可执行文件同级的 `temp/opencode-auth-*` 目录读取脚本生成的 Account、Workspace ID 和 Auth Cookie。每次运行的脚本、标准输出和结果文件会保留，便于排查提取失败；登录密码不会写入账号配置。

安装脚本依赖：

```bash
python3 -m pip install requests
```

Windows 也可以使用：

```powershell
py -3 -m pip install requests
```

每次协议登录的调试文件保存在：

```text
temp/opencode-auth-随机字符/
├── opencode_auth_extractor.py
├── opencode_账号.json
├── stdout.log
└── stderr.log
```

这些目录不会自动删除，确认不再需要后可手动清理。`temp/` 已被 Git 忽略。

API Key-only 新增界面只要求填写 API Key、优先级和独立代理。探测请求固定使用 `deepseek-v4-flash` 和 `max_tokens: 1`，可能产生极少量 Token 消耗。公开模型目录不会用于验证 API Key。若上游成功响应没有返回 Workspace ID，凭证仍会保存为 API Key-only，额度信息保持不可查询，且后续刷新不会重复执行权益探测。

仅 API Key 可以确认部分 Go/Zen 请求权益，但无法读取 Workspace 页面中的 Go 5 小时、每周、每月额度窗口或 Zen 余额。卡片会将这种情况标记为“需 Cookie 查询”，不会误判为“未开通”。

探测模型返回 `RegionError` 表示该模型需要在 Workspace 中明确选择托管区域，并不代表 API Key 无效或整个 Go/Zen 服务不可用。程序会从错误链接提取 Workspace ID，并将对应服务视为已通过鉴权。只有 HTTP 401 或明确的认证错误才会拒绝 API Key。

程序启动时只读取账号配置，不会按名称、Workspace ID、Cookie、账号类型或字段完整度自动删除或合并账号。账号仅能由管理页面的删除操作移除。

| 字段 | 用途 | 是否必填 |
| --- | --- | --- |
| 显示名称 | 本地识别账号；留空自动使用邮箱 | 否 |
| Workspace ID | 形如 `wrk_...` | 是 |
| OpenCode API Key | 查询 Go/Zen 模型目录；留空自动获取 | 否 |
| `auth` Cookie | 查询 Workspace Go 额度和 Zen 余额 | 是 |

点击“自动识别账号”后，程序会读取当前账号邮箱和该用户本人拥有的完整 API Key。存在多个 Key 时需要选择一个；其他成员的掩码 Key 不会作为候选项。

API Key 与网站 Cookie 用途不同。仅有 API Key 不能查询 Workspace 页面中的 Go 额度和 Zen 余额。自动识别失败时可检查 Cookie 后重试，也可手动填写名称和 API Key。

### 获取 Workspace ID 和 Cookie

1. 使用 GitHub 或 Google 登录 [OpenCode Zen](https://opencode.ai/zen)。
2. 进入工作区后，从地址栏复制 `workspace/wrk_...` 中的 Workspace ID。
3. 从浏览器开发者工具的 Cookie 列表复制 `opencode.ai` 的完整 `auth` Cookie。

OpenCode 登录采用 GitHub/Google OAuth，最终会话 Cookie 为 HttpOnly。OpenCode Pool Gateway 不保存 GitHub/Google 密码，也无法在纯服务端或 Ubuntu 部署中直接读取浏览器 Cookie，因此目前仍需手动提供 Workspace ID 和 Cookie。

## 数据目录

配置固定保存在可执行文件旁：

```text
data/accounts.json
data/auth.json
data/gateway.json
data/tokens.json
data/requests.jsonl
```

`accounts.json` 包含 OpenCode 敏感凭证；`auth.json` 包含管理账号和密码哈希；`tokens.json` 只保存本站令牌摘要；`requests.jsonl` 保存脱敏请求日志。请勿上传、分享或提交 `data/`；仓库已默认排除该目录。

建议权限：

```bash
chmod 700 data
chmod 600 data/accounts.json
```

## 构建

要求 Go 1.22 或更高版本。

Windows PowerShell：

```powershell
.\scripts\build.ps1
```

Linux/macOS：

```bash
chmod +x scripts/build.sh
./scripts/build.sh
```

产物生成在 `dist/`，包含 Windows AMD64、Linux AMD64、Linux ARM64 和 `SHA256SUMS.txt`。

版本号由根目录 `VERSION` 管理，并通过构建参数写入程序：

```bash
./opencode-pool-gateway-0.6.0-linux-amd64 --version
```

## 接口

| 接口 | 方法 | 说明 |
| --- | --- | --- |
| `/api/accounts` | GET/POST | 查询或添加账号（名称和 API Key 可自动获取） |
| `/api/accounts/discover` | POST | 使用 Workspace ID 与 auth Cookie 自动读取邮箱和本人 API Key |
| `/api/accounts/login-extract` | POST | 运行账号密码登录协议并自动读取 Workspace、Cookie、邮箱和 API Key |
| `/api/accounts/{id}` | PUT/DELETE | 编辑或删除账号 |
| `/api/accounts/{id}/refresh` | POST | 刷新单个账号 |
| `/api/accounts/{id}/models` | GET | 分别查询 Go/Zen 模型 |
| `/api/refresh` | POST | 刷新全部账号 |
| `/api/settings` | GET/PUT | 查询或热更新调度、重试和全局代理配置 |
| `/api/tokens` | GET/POST | 查询或创建本站访问令牌 |
| `/api/tokens/{id}` | PUT/DELETE | 启停、重命名或删除令牌 |
| `/api/logs` | GET/DELETE | 查询或清空请求日志 |
| `/api/version` | GET | 查询版本信息 |
| `/api/shutdown` | POST | 安全退出程序 |

## 安全说明

- 即使已有应用登录，也建议通过 HTTPS 反向代理访问，不要通过公网明文 HTTP 输入密码。
- Ubuntu 部署建议配合防火墙或 VPN，避免无关来源访问登录入口。
- Cookie 和 API Key 均视为账号凭证；泄漏后应立即撤销或重新登录。
- OpenCode Pool Gateway 是非官方项目，OpenCode 页面或接口结构变化可能导致采集暂时失效。
- 第三方客户端应将本站令牌放在 `Authorization: Bearer <token>` 中，切勿直接暴露 OpenCode API Key。

## 版本管理

项目遵循 [Semantic Versioning](https://semver.org/)。发布新版本时：

1. 更新 `VERSION`。
2. 更新 `CHANGELOG.md`。
3. 运行测试与构建脚本。
4. 创建 `vX.Y.Z` Git 标签和 GitHub Release。

## License

[MIT](LICENSE)
