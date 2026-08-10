# Ubuntu 部署

以下示例将 GoQuota 安装在 `/opt/goquota`，并通过 systemd 运行。

## 安装

AMD64：

```bash
sudo useradd --system --home /opt/goquota --shell /usr/sbin/nologin goquota
sudo mkdir -p /opt/goquota/data
sudo cp goquota-0.1.0-linux-amd64 /opt/goquota/goquota
sudo chown -R goquota:goquota /opt/goquota
sudo chmod 750 /opt/goquota/goquota /opt/goquota/data
```

ARM64 服务器将文件名替换为 `goquota-0.1.0-linux-arm64`。

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

GoQuota 当前监听 `8787` 端口。不要直接向公网开放，优先使用以下方式之一：

- SSH 端口转发：`ssh -L 8787:127.0.0.1:8787 user@server`
- 仅允许可信内网或 VPN 访问。
- 使用 Caddy/Nginx 配置 HTTPS 和额外身份认证。

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
