# Ubuntu 部署

以下示例将 GoQuota 安装在 `/opt/goquota`，并通过 systemd 运行。

## 安装

AMD64：

```bash
sudo useradd --system --home /opt/goquota --shell /usr/sbin/nologin goquota
sudo mkdir -p /opt/goquota/data
sudo cp goquota-0.2.0-linux-amd64 /opt/goquota/goquota
sudo chown -R goquota:goquota /opt/goquota
sudo chmod 750 /opt/goquota/goquota /opt/goquota/data
```

ARM64 服务器将文件名替换为 `goquota-0.2.0-linux-arm64`。

创建仅 root 可读的登录环境文件：

```bash
sudo install -m 600 /dev/null /etc/goquota.env
sudo sh -c 'cat > /etc/goquota.env' <<'EOF'
GOQUOTA_USERNAME=admin
GOQUOTA_PASSWORD=请替换为至少12位的随机强密码
GOQUOTA_COOKIE_SECURE=1
EOF
```

`GOQUOTA_COOKIE_SECURE=1` 要求通过 HTTPS 访问。如果还没有配置 HTTPS，首次调试时可暂时设为 `0`。

## systemd

创建 `/etc/systemd/system/goquota.service`：

```ini
[Unit]
Description=GoQuota OpenCode quota monitor
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=goquota
Group=goquota
WorkingDirectory=/opt/goquota
EnvironmentFile=/etc/goquota.env
ExecStart=/opt/goquota/goquota
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=/opt/goquota/data

[Install]
WantedBy=multi-user.target
```

启用服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now goquota
sudo systemctl status goquota
```

查看日志：

```bash
journalctl -u goquota -f
```

## 网络安全

GoQuota 当前监听 `8787` 端口，应用自身已提供登录验证。公网部署仍必须使用 HTTPS，优先使用以下方式之一：

- SSH 端口转发：`ssh -L 8787:127.0.0.1:8787 user@server`
- 仅允许可信内网或 VPN 访问。
- 使用 Caddy/Nginx 配置 HTTPS，并转发 `X-Forwarded-Proto`。

通过 SSH 转发后，在本机访问 `http://localhost:8787`。

## 升级

```bash
sudo systemctl stop goquota
sudo cp goquota-NEW-linux-amd64 /opt/goquota/goquota
sudo chown goquota:goquota /opt/goquota/goquota
sudo chmod 750 /opt/goquota/goquota
sudo systemctl start goquota
```

账号数据位于 `/opt/goquota/data/accounts.json`，替换可执行文件不会删除配置。升级前仍建议备份该文件并妥善保护。
