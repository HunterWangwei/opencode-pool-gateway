# Ubuntu 部署

以下示例将 OpenCode Pool Gateway 安装在 `/opt/opencode-pool-gateway`，并通过 systemd 运行。

## 安装

AMD64：

```bash
sudo useradd --system --home /opt/opencode-pool-gateway --shell /usr/sbin/nologin opencode-pool-gateway
sudo mkdir -p /opt/opencode-pool-gateway/data /opt/opencode-pool-gateway/temp
sudo cp opencode-pool-gateway-0.7.0-linux-amd64 /opt/opencode-pool-gateway/opencode-pool-gateway
sudo chown -R opencode-pool-gateway:opencode-pool-gateway /opt/opencode-pool-gateway
sudo chmod 750 /opt/opencode-pool-gateway/opencode-pool-gateway /opt/opencode-pool-gateway/data /opt/opencode-pool-gateway/temp
```

ARM64 服务器将文件名替换为 `opencode-pool-gateway-0.7.0-linux-arm64`。

账号密码登录需要 Python 3 和 `requests`：

```bash
sudo apt update
sudo apt install -y python3 python3-requests
```

首次启动会在 `/opt/opencode-pool-gateway/data/auth.json` 创建登录配置，并在 systemd 日志中显示一次随机初始密码。登录后请在网页“设置”中修改凭证。

HTTPS 部署可创建 `/etc/opencode-pool-gateway.env`，内容为 `OPG_COOKIE_SECURE=1`。该选项要求通过 HTTPS 访问。

## systemd

创建 `/etc/systemd/system/opencode-pool-gateway.service`：

```ini
[Unit]
Description=OpenCode account pool and API gateway
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=opencode-pool-gateway
Group=opencode-pool-gateway
WorkingDirectory=/opt/opencode-pool-gateway
EnvironmentFile=-/etc/opencode-pool-gateway.env
ExecStart=/opt/opencode-pool-gateway/opencode-pool-gateway
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/opt/opencode-pool-gateway/data /opt/opencode-pool-gateway/temp

[Install]
WantedBy=multi-user.target
```

启用服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now opencode-pool-gateway
sudo systemctl status opencode-pool-gateway
```

查看日志：

```bash
journalctl -u opencode-pool-gateway -f
```

## 网络安全

OpenCode Pool Gateway 当前监听 `8787` 端口，应用自身已提供登录验证。公网部署仍必须使用 HTTPS，优先使用以下方式之一：

- SSH 端口转发：`ssh -L 8787:127.0.0.1:8787 user@server`
- 仅允许可信内网或 VPN 访问。
- 使用 Caddy/Nginx 配置 HTTPS，并转发 `X-Forwarded-Proto`。

通过 SSH 转发后，在本机访问 `http://localhost:8787`。

## 网关接入

在网页“令牌管理”中创建本站访问令牌，再将第三方客户端的 API 地址指向服务器域名对应的 Go 或 Zen 路由。客户端发送的 Bearer Token 只用于 OpenCode Pool Gateway 鉴权，上游请求会自动换成账号池中选定凭证的 OpenCode API Key。

公网部署必须通过 HTTPS。反向代理需要保留请求方法、路径、查询参数、请求体和流式响应，并允许较长连接时间。

## 升级

```bash
sudo systemctl stop opencode-pool-gateway
sudo cp opencode-pool-gateway-NEW-linux-amd64 /opt/opencode-pool-gateway/opencode-pool-gateway
sudo chown opencode-pool-gateway:opencode-pool-gateway /opt/opencode-pool-gateway/opencode-pool-gateway
sudo chmod 750 /opt/opencode-pool-gateway/opencode-pool-gateway
sudo systemctl start opencode-pool-gateway
```

全部配置和日志位于 `/opt/opencode-pool-gateway/data/`，替换可执行文件不会删除配置。升级前仍建议备份该目录并妥善保护。

## 添加 OpenCode 账号

Ubuntu 服务器无法直接读取用户电脑浏览器中的 HttpOnly Cookie。请先在桌面浏览器通过 GitHub 或 Google 登录 OpenCode，然后手动复制 Workspace ID 和完整 `auth` Cookie 到 OpenCode Pool Gateway。

提交这两项后，OpenCode Pool Gateway 会自动读取邮箱和当前用户本人拥有的 API Key；如果有多个 Key，需在网页中选择一个。不要在服务器中保存 GitHub 或 Google 密码。
