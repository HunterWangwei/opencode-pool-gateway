# Changelog

本文档记录 OpenCode Pool Gateway 的重要变更，版本号遵循 Semantic Versioning。

## [Unreleased]

## [0.7.0] - 2026-08-14

### Added

- 新增独立代理池页面，支持以表格形式添加、编辑和删除多条代理。
- 代理表格显示每条代理绑定的凭证、请求成功次数和请求失败次数。
- 账号池卡片显示当前凭证实际绑定的代理。
- 代理协议新增 `socks5h://` 支持，域名解析通过 SOCKS5 代理完成。
- Zen Free 模型清单新增 `hy3-free` 和 `nemotron-3.5-lightning-free`。

### Changed

- 代理池按凭证 ID 进行稳定映射，同一凭证固定使用同一条代理，不随请求轮换。
- 代理优先级调整为：凭证独立代理、代理池固定代理、全局代理。
- 代理池从设置页移至独立导航页面，设置页仅保留全局代理和网关调度配置。
- Web 侧栏默认版本号同步为 `0.7.0`，加载后由 `/api/version` 返回的构建版本覆盖。

### Tests

- 新增 SOCKS5H 协议校验和稳定代理分配回归测试。

## [0.6.1] - 2026-08-11

### Changed

- 第三方客户端调用本站网关时，访问令牌同时支持 `Authorization: Bearer <token>` 和 `x-api-key: <token>`。
- 转发 `/zen/v1/messages` 与 `/zen/go/v1/messages` 时，自动使用账号池 OpenCode API Key 设置上游 `x-api-key`。
- 其他 Chat Completions、Responses、Models 和 Gemini 路由继续使用上游 Bearer 鉴权。

### Fixed

- 修复客户端使用 `x-api-key` 时被本站错误返回“访问令牌无效”的问题。
- 修复 Messages 路由仅设置 Bearer 导致 OpenCode 返回 `AuthError: Missing API key` 的问题。
- 转发前同时移除客户端的 `Authorization` 和 `x-api-key`，确保本站访问令牌不会发送给 OpenCode。

### Tests

- 新增 `x-api-key` 入站鉴权与 Messages 上游鉴权转换回归测试。

## [0.6.0] - 2026-08-11

### Added

- 添加账号新增“账号密码登录”方式，可通过 GitHub 账号、密码和邮箱密码在后台运行协议脚本。
- 协议脚本自动提取 Account、Workspace ID 和 auth Cookie，随后复用现有账号发现流程读取邮箱与本人 API Key。
- 内置用户提供的原始 `opencode_auth_extractor.py`，程序按其 `账号----密码----邮箱密码` 参数协议调用，不修改脚本行为。
- 自动检测 `python3`、`python` 或 Windows `py -3`，并确认 `requests` 依赖可用。

### Changed

- 协议登录任务固定在程序本体目录的 `temp/opencode-auth-*` 中运行。
- 每次运行保留脚本副本、结果 JSON、`stdout.log` 和 `stderr.log`，方便人工排查提取过程。
- 登录成功后若存在多个 API Key，可在保存前自行选择。

### Security

- `data/`、`temp/`、脚本测试结果和 Python 缓存目录均不会提交到 Git。
- GitHub 密码和邮箱密码只用于当前协议脚本进程，不会写入账号配置。

## [0.5.1] - 2026-08-10

### Fixed

- 移除启动时按 Workspace ID 和 Cookie 自动过滤“旧版演示账号”的迁移逻辑，任何用户账号都不会再被程序自动删除或合并。
- API Key 探测不再将所有 HTTP 403 响应误判为凭证失效；仅 HTTP 401 或明确的认证错误会判定 API Key 无效。
- `deepseek-v4-flash` 返回模型专属 `RegionError` 时，视为 API Key 和对应服务有效，不再错误阻止账号添加或标记 Go 未开通。
- Workspace ID 提取支持 `/workspace/<id>/go` 等非 billing 链接。

### Tests

- 新增账号配置完整保留、区域限制响应和明确认证错误的回归测试。

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
