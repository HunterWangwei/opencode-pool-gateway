# Ubuntu 閮ㄧ讲

浠ヤ笅绀轰緥灏?OpenCode Pool Gateway 瀹夎鍦?`/opt/opencode-pool-gateway`锛屽苟閫氳繃 systemd 杩愯銆?
## 瀹夎

AMD64锛?
```bash
sudo useradd --system --home /opt/opencode-pool-gateway --shell /usr/sbin/nologin opencode-pool-gateway
sudo mkdir -p /opt/opencode-pool-gateway/data /opt/opencode-pool-gateway/temp
sudo cp opencode-pool-gateway-0.7.3-linux-amd64 /opt/opencode-pool-gateway/opencode-pool-gateway
sudo chown -R opencode-pool-gateway:opencode-pool-gateway /opt/opencode-pool-gateway
sudo chmod 750 /opt/opencode-pool-gateway/opencode-pool-gateway /opt/opencode-pool-gateway/data /opt/opencode-pool-gateway/temp
```

ARM64 鏈嶅姟鍣ㄥ皢鏂囦欢鍚嶆浛鎹负 `opencode-pool-gateway-0.7.3-linux-arm64`銆?
璐﹀彿瀵嗙爜鐧诲綍闇€瑕?Python 3 鍜?`requests`锛?
```bash
sudo apt update
sudo apt install -y python3 python3-requests
```

棣栨鍚姩浼氬湪 `/opt/opencode-pool-gateway/data/auth.json` 鍒涘缓鐧诲綍閰嶇疆锛屽苟鍦?systemd 鏃ュ織涓樉绀轰竴娆￠殢鏈哄垵濮嬪瘑鐮併€傜櫥褰曞悗璇峰湪缃戦〉鈥滆缃€濅腑淇敼鍑瘉銆?
HTTPS 閮ㄧ讲鍙垱寤?`/etc/opencode-pool-gateway.env`锛屽唴瀹逛负 `OPG_COOKIE_SECURE=1`銆傝閫夐」瑕佹眰閫氳繃 HTTPS 璁块棶銆?
## systemd

鍒涘缓 `/etc/systemd/system/opencode-pool-gateway.service`锛?
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

鍚敤鏈嶅姟锛?
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now opencode-pool-gateway
sudo systemctl status opencode-pool-gateway
```

鏌ョ湅鏃ュ織锛?
```bash
journalctl -u opencode-pool-gateway -f
```

## 缃戠粶瀹夊叏

OpenCode Pool Gateway 褰撳墠鐩戝惉 `8787` 绔彛锛屽簲鐢ㄨ嚜韬凡鎻愪緵鐧诲綍楠岃瘉銆傚叕缃戦儴缃蹭粛蹇呴』浣跨敤 HTTPS锛屼紭鍏堜娇鐢ㄤ互涓嬫柟寮忎箣涓€锛?
- SSH 绔彛杞彂锛歚ssh -L 8787:127.0.0.1:8787 user@server`
- 浠呭厑璁稿彲淇″唴缃戞垨 VPN 璁块棶銆?- 浣跨敤 Caddy/Nginx 閰嶇疆 HTTPS锛屽苟杞彂 `X-Forwarded-Proto`銆?
閫氳繃 SSH 杞彂鍚庯紝鍦ㄦ湰鏈鸿闂?`http://localhost:8787`銆?
## 缃戝叧鎺ュ叆

鍦ㄧ綉椤碘€滀护鐗岀鐞嗏€濅腑鍒涘缓鏈珯璁块棶浠ょ墝锛屽啀灏嗙涓夋柟瀹㈡埛绔殑 API 鍦板潃鎸囧悜鏈嶅姟鍣ㄥ煙鍚嶅搴旂殑 Go 鎴?Zen 璺敱銆傚鎴风鍙戦€佺殑 Bearer Token 鍙敤浜?OpenCode Pool Gateway 閴存潈锛屼笂娓歌姹備細鑷姩鎹㈡垚璐﹀彿姹犱腑閫夊畾鍑瘉鐨?OpenCode API Key銆?
鍏綉閮ㄧ讲蹇呴』閫氳繃 HTTPS銆傚弽鍚戜唬鐞嗛渶瑕佷繚鐣欒姹傛柟娉曘€佽矾寰勩€佹煡璇㈠弬鏁般€佽姹備綋鍜屾祦寮忓搷搴旓紝骞跺厑璁歌緝闀胯繛鎺ユ椂闂淬€?
## 鍗囩骇

```bash
sudo systemctl stop opencode-pool-gateway
sudo cp opencode-pool-gateway-NEW-linux-amd64 /opt/opencode-pool-gateway/opencode-pool-gateway
sudo chown opencode-pool-gateway:opencode-pool-gateway /opt/opencode-pool-gateway/opencode-pool-gateway
sudo chmod 750 /opt/opencode-pool-gateway/opencode-pool-gateway
sudo systemctl start opencode-pool-gateway
```

鍏ㄩ儴閰嶇疆鍜屾棩蹇椾綅浜?`/opt/opencode-pool-gateway/data/`锛屾浛鎹㈠彲鎵ц鏂囦欢涓嶄細鍒犻櫎閰嶇疆銆傚崌绾у墠浠嶅缓璁浠借鐩綍骞跺Ε鍠勪繚鎶ゃ€?
## 娣诲姞 OpenCode 璐﹀彿

Ubuntu 鏈嶅姟鍣ㄦ棤娉曠洿鎺ヨ鍙栫敤鎴风數鑴戞祻瑙堝櫒涓殑 HttpOnly Cookie銆傝鍏堝湪妗岄潰娴忚鍣ㄩ€氳繃 GitHub 鎴?Google 鐧诲綍 OpenCode锛岀劧鍚庢墜鍔ㄥ鍒?Workspace ID 鍜屽畬鏁?`auth` Cookie 鍒?OpenCode Pool Gateway銆?
鎻愪氦杩欎袱椤瑰悗锛孫penCode Pool Gateway 浼氳嚜鍔ㄨ鍙栭偖绠卞拰褰撳墠鐢ㄦ埛鏈汉鎷ユ湁鐨?API Key锛涘鏋滄湁澶氫釜 Key锛岄渶鍦ㄧ綉椤典腑閫夋嫨涓€涓€備笉瑕佸湪鏈嶅姟鍣ㄤ腑淇濆瓨 GitHub 鎴?Google 瀵嗙爜銆?

