# Changelog

本文档记录 OpenCode Pool Gateway 的重要变更，版本号遵循 Semantic Versioning。

## [Unreleased]

## [0.5.0] - 2026-08-10

### Added

- 账号类型细分为 Free、Go、Zen 和 Go + Zen，并在账号卡片中显示。
- 凭证模型目录新增“可用/不可用”标记和不可用原因。
- 添加账号新增“仅 API Key”模式，只需填写 API Key、优先级和独立代理。
- 仅 API Key 模式使用 `deepseek-v4-flash` 分别请求 `/zen/v1/chat/completions` 与 `/zen/go/v1/chat/completions` 探测权益，并从 `CreditsError` 付款链接提取 Workspace ID。

### Changed

- 模型目录不再用于验证 API Key；模型目录仅用于展示模型清单。
- API Key-only 凭证明确区分“服务权益探测”与“Workspace 配额读取”，无 Cookie 时不再将无法读取 Go 配额显示为未开通。
- 账号卡片将类型徽标移至独立元信息区域，长名称和 Workspace ID 使用省略显示，避免与运行状态重叠。
- API Key-only 表单使用明确的必填提示，不再显示“自动获取”。
- 模型目录接口返回空集合时统一序列化为 `[]`。

### Fixed

- Free 账号不再因 Zen 余额为零进入告警状态。
- 网关按账号权益筛选凭证：Free 账号不能请求 Go 模型，Zen 付费模型只使用已开启 Zen 计费的凭证。
- 没有符合模型权益的凭证时返回结构化 `CreditsError` 和 HTTP 402。
- PowerShell 构建脚本在 `go build` 失败时会立即终止，不再错误报告构建成功。

## [0.4.2] - 2026-08-10

### Added

- 全局代理和单凭证代理新增带用户名密码认证的 SOCKS5 支持。

### Changed

- 设置页改为响应式双栏布局，充分利用桌面端可用宽度。
- 桌面端侧栏固定在视口左侧，服务状态、同步时间和版本号不再随内容滚动。
- 请求耗时改为毫秒、秒、分秒三级自适应显示。

### Fixed

- 修复空请求日志或空令牌列表返回 `null` 导致页面读取 `length` 失败的问题。
- 请求日志新增 gzip 与 SSE 响应解析，可从流式请求中提取 Token 用量。
- 同时识别 Chat Completions 的 `prompt_tokens_details.cached_tokens` 和 Responses 的 `input_tokens_details.cached_tokens`。

### Dependencies

- 新增 `golang.org/x/net/proxy`，用于标准 SOCKS5 与用户名密码认证。

## [0.4.1] - 2026-08-10

### Added

- Zen 网关新增 Anthropic Messages、OpenAI-compatible Chat Completions 和 Gemini 动态模型路由。
- Go 网关新增 OpenAI Responses 与 Anthropic Messages 路由。
- 请求日志新增 Anthropic 与 Gemini 用量字段解析，并可从 Gemini URL 记录模型 ID。

## [0.4.0] - 2026-08-10

### Added

- 新增 OpenCode API 透明转发，兼容 Go Chat Completions、Go Models、Zen Responses 与 Zen Models 路由。
- 新增负载均衡轮询和优先级故障转移模式，并支持最大尝试次数配置。
- 账号凭证新增优先级与独立 HTTP/HTTPS 代理，全局代理可在设置页热更新。
- 新增本站访问令牌管理，令牌只在创建时显示明文，磁盘仅保存 SHA-256 摘要。
- 新增请求日志页，记录凭证、模型、Token 用量、缓存读写、耗时和脱敏错误报文。

### Changed

- 项目由 GoQuota / `opencode-go-quota-monitor` 更名为 OpenCode Pool Gateway / `opencode-pool-gateway`，以准确反映账号池、API 网关和故障转移能力。
- 可执行文件、Go module、网页品牌、部署目录、systemd 服务和环境变量统一使用新名称。

### Security

- 第三方客户端令牌不会转发到 OpenCode，上游只接收所选账号的 API Key。
- 日志不记录 API Key、Cookie 或完整本站访问令牌。

## [0.3.0] - 2026-08-10

### Added

- 添加账号时显示名称和 API Key 改为可选，可通过 Workspace ID 与 auth Cookie 自动读取账号邮箱。
- 自动读取当前登录用户本人拥有的完整 API Key；存在多个 Key 时可在网页中选择。
- 支持解析 OpenCode `key.list` 返回的 Seroval 流式响应，正确提取当前用户拥有的完整 API Key。

### Changed

- 登录凭证改为优先读取程序目录下的 `data/auth.json`。
- 首次启动自动生成随机初始密码，并将密码哈希写入配置文件。
- 设置页面可修改登录用户名和密码，保存后立即生效并注销旧会话。
- 管理密码最小长度调整为 8 个字符。
- 账号卡片与保存提示明确显示 API Key 是否已配置。

### Fixed

- 自动识别失败时保留手动填写兜底，并明确提示 Cookie 失效或 Workspace 不匹配。
- 修正 SolidStart 服务函数所需的 Seroval JSON 请求格式，避免将接口错误误判为 Cookie 失效。
- 识别失败后不会再次点击就静默保存未识别账号。
- OpenCode Pool Gateway 管理会话失效时自动返回登录页，避免将本地 401 错误显示为 OpenCode 账号识别失败。

## [0.2.0] - 2026-08-10

### Added

- 服务端登录验证，未认证用户无法访问页面和 API。
- 24 小时 HttpOnly、SameSite 会话 Cookie。
- 单 IP 登录失败限流：5 次失败后锁定 5 分钟。
- `OPG_USERNAME`、`OPG_PASSWORD` 和 `OPG_COOKIE_SECURE` 部署配置。
- 网页退出登录入口和基础安全响应头。

## [0.1.0] - 2026-08-10

### Added

- OpenCode Workspace 批量监控和账号卡片视图。
- Go 订阅额度与 Zen 按量付费余额分别识别。
- Go、Zen 可用模型目录分别实时查询。
- Zen 已弃用模型标记及官方弃用日期。
- 账号添加、编辑、删除、单项刷新和全量刷新。
- Windows 可见控制台及网页内安全退出入口。
- 配置随程序存放在 `data/accounts.json`。
- Windows 与 Linux 跨平台构建脚本。
