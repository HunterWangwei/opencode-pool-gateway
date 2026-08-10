# GoQuota

GoQuota 是一个使用 Go 标准库构建的 OpenCode Go / Zen 账号额度监控工具。程序不依赖 OpenCode CLI，通过网页管理多个 Workspace，并直接查询 OpenCode 官方页面和模型接口。

> 当前版本：`0.1.0`

## 功能

- 批量管理多个 OpenCode Workspace。
- 分别识别 Go 订阅额度与 Zen 按量付费余额。
- 展示 Go 的 5 小时、每周、每月额度窗口和重置时间。
- 分别查询 Go 与 Zen 可用模型，不混用模型目录。
- 标记 Zen 已弃用模型并显示官方弃用日期。
- 账号卡片支持搜索、状态筛选、编辑、刷新和删除。
- Windows 控制台支持打开网页、刷新、帮助和安全退出。
- 数据与可执行文件放在同一目录，便于迁移和备份。

## 快速开始

### Windows

下载 `goquota-0.1.0-windows-amd64.exe`，双击运行，然后访问：

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

### Linux

```bash
chmod +x goquota-0.1.0-linux-amd64
./goquota-0.1.0-linux-amd64
```

ARM64 Ubuntu 使用 `goquota-0.1.0-linux-arm64`。服务器部署及 systemd 配置见 [docs/ubuntu.md](docs/ubuntu.md)。

## 添加账号

每个 Workspace 只添加一次，程序会同时检测 Go 与 Zen：

| 字段 | 用途 | 是否必填 |
| --- | --- | --- |
| 显示名称 | 本地识别账号 | 是 |
| Workspace ID | 形如 `wrk_...` | 是 |
| OpenCode API Key | 查询 Go/Zen 模型目录 | 否，但建议配置 |
| `auth` Cookie | 查询 Workspace Go 额度和 Zen 余额 | 是 |

API Key 与网站 Cookie 用途不同。仅有 API Key 不能查询 Workspace 页面中的 Go 额度和 Zen 余额。

## 数据目录

配置固定保存在可执行文件旁：

```text
data/accounts.json
```

该文件包含敏感凭证，目前为本地明文存储。请勿上传、分享或提交到 Git；仓库的 `.gitignore` 已默认排除整个 `data/` 目录。

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
./goquota-0.1.0-linux-amd64 --version
```

## 接口

| 接口 | 方法 | 说明 |
| --- | --- | --- |
| `/api/accounts` | GET/POST | 查询或添加账号 |
| `/api/accounts/{id}` | PUT/DELETE | 编辑或删除账号 |
| `/api/accounts/{id}/refresh` | POST | 刷新单个账号 |
| `/api/accounts/{id}/models` | GET | 分别查询 Go/Zen 模型 |
| `/api/refresh` | POST | 刷新全部账号 |
| `/api/version` | GET | 查询版本信息 |
| `/api/shutdown` | POST | 安全退出程序 |

## 安全说明

- 不要将服务端口直接暴露到公网。
- Ubuntu 部署建议通过 HTTPS 反向代理，并增加访问认证或 VPN。
- Cookie 和 API Key 均视为账号凭证；泄漏后应立即撤销或重新登录。
- GoQuota 是非官方项目，OpenCode 页面或接口结构变化可能导致采集暂时失效。

## 版本管理

项目遵循 [Semantic Versioning](https://semver.org/)。发布新版本时：

1. 更新 `VERSION`。
2. 更新 `CHANGELOG.md`。
3. 运行测试与构建脚本。
4. 创建 `vX.Y.Z` Git 标签和 GitHub Release。

## License

[MIT](LICENSE)
