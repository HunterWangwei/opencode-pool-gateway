# OpenCode Auth Extractor — 使用文档

> 纯协议层 GitHub OAuth 自动登录 opencode.ai，提取 Workspace ID 和 Auth Cookie。

---

## 前置：确认 Python 环境

先运行 `check_env.bat` 确认 Python 可用。正常输出如下：

```
=== Python Detection ===
C:\Users\...\python.exe
Python 3.12.x
=== PIP Detection ===
C:\Users\...\pip.exe
pip 24.x
```

如果 `python` 找不到，用 `py` 代替：
```powershell
py opencode_auth_extractor.py "..." -v
py -m pip install requests
```

---

## 安装依赖

```powershell
pip install requests
# 或
py -m pip install requests
```

---

## 使用

```powershell
python opencode_auth_extractor.py "账号----密码"                # 无验证
python opencode_auth_extractor.py "账号----密码----邮箱密码"     # 带验证
python opencode_auth_extractor.py "..." -v                      # 显示过程
```

---

## 参数格式

| 参数 | 必填 | 说明 |
|---|---|---|
| `账号----密码` | ✅ | 以 `----` 分隔，账号即邮箱 |
| `----邮箱密码` | 条件 | 设备验证时必填 |
| `-v` | 否 | 打印每步过程 |

---

## 输出

```
==================================================
  Account:      alexandrah256@outlook.ph
  Workspace ID: wrk_xxxxxxxxxxxx
  Auth Cookie:  auth=Fe26.2**...
  Output:       opencode_alexandrah256_at_outlook_ph.json
==================================================
```



**常见问题**

| 现象 | 解决 |
|---|---|
| 无任何输出 | 运行 `check_env.bat`，安装 Python + `pip install requests` |
| `2FA required` | 换仅邮箱验证的账号 |
| `Verification needed` | 补上 `----邮箱密码` |
| `No verification code` | 检查邮箱密码，重试 |
