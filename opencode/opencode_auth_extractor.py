#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""OpenCode Auth Extractor. Usage: python opencode_auth_extractor.py account----password----email_pass [-v]"""
import sys, os, traceback

def _print(msg):
    print(msg, flush=True)

_print("OpenCode Auth Extractor v3")
_print(f"  Python: {sys.version}")
try:
    import requests; _print(f"  requests: {requests.__version__}")
except ImportError:
    _print("  FATAL: requests not installed. Run: pip install requests")
    sys.exit(1)
_print("")

import re, json, time, urllib.parse

OPENCODE  = "https://opencode.ai"
AUTH      = "https://auth.opencode.ai"
GITHUB    = "https://github.com"
EMAIL_API = "https://www.ruoanzhu.com/tools/emailApi2/getContent"
UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
H = {"Content-Type": "application/x-www-form-urlencoded", "Origin": GITHUB,
     "Sec-Fetch-Site": "same-origin", "Sec-Fetch-Dest": "document", "Sec-Fetch-Mode": "navigate"}

def dec(u): return u.replace("&amp;", "&")

def fields(html):
    fs = {}
    for m in re.finditer(r'<input[^>]*type="hidden"[^>]*name="([^"]+)"[^>]*value="([^"]*)"', html):
        n = m.group(1)
        if not n.startswith(("required_field_", "turbo_")): fs[n] = m.group(2)
    m = re.search(r'name="authenticity_token"\s+value="([^"]+)"', html)
    if m: fs["authenticity_token"] = m.group(1)
    return fs

def get_code(email, pwd, verbose):
    url = f"{EMAIL_API}?email={urllib.parse.quote(email)}&pwd={urllib.parse.quote(pwd)}&htmlOut=0&islj=0&num=3&route=1&islj=2"
    for attempt in range(10):
        try:
            r = requests.get(url, timeout=30)
            if r.status_code == 200:
                t = r.text
                try:
                    d = r.json()
                    if d.get("status") and d.get("lastOne"):
                        m = re.search(r"code[:\s]*(\d{6})", d["lastOne"].get("body",""), re.IGNORECASE)
                        if m: return m.group(1)
                except:
                    m = re.search(r"code[:\s]*(\d{6})", t, re.IGNORECASE)
                    if m: return m.group(1)
        except: pass
        if verbose: _print(f"  polling email... ({attempt+1}/10)")
        time.sleep(8)

class Extractor:
    def __init__(self, account, password, email_pass=None, verbose=False):
        self.gh = account; self.gp = password
        self.em = account; self.ep = email_pass
        self.v = verbose
        self.s = requests.Session()
        self.s.headers.update({"User-Agent": UA, "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"})
        self._av = None

    def log(self, msg):
        if self.v: _print(f"  {msg}")

    def _save(self):
        for c in self.s.cookies:
            if c.name == "authorization": self._av = c.value
    def _restore(self):
        if self._av: self.s.cookies.set("authorization", self._av, domain="auth.opencode.ai", path="/")

    def run(self):
        self.log("=" * 50)
        self.log("OpenCode Auth Extractor")

        self.log("[1/5] Initiating OpenAuth...")
        r = self.s.get(f"{OPENCODE}/auth", allow_redirects=False); self.log(f"  GET /auth -> {r.status_code}")
        r = self.s.get(f"{OPENCODE}{r.headers['Location']}", allow_redirects=False); self.log(f"  GET /auth/authorize -> {r.status_code}")
        r = self.s.get(r.headers["Location"], allow_redirects=False); self._save(); self.log(f"  OpenAuth page -> {r.status_code}")

        self.log("[2/5] GitHub OAuth redirect...")
        r = self.s.get(f"{AUTH}/github/authorize", allow_redirects=False)
        gh = dec(r.headers["Location"])
        self.log(f"  -> {gh[:80]}...")

        self.log("[3/5] GitHub login...")
        r = self.s.get(gh, allow_redirects=True); self._restore()
        f = fields(r.text)
        d = {"login": self.gh, "password": self.gp, "authenticity_token": f.get("authenticity_token",""),
             "return_to": f.get("return_to",""), "client_id": f.get("client_id",""),
             "timestamp": f.get("timestamp",""), "timestamp_secret": f.get("timestamp_secret",""),
             "webauthn-conditional": "undefined", "javascript-support": "true",
             "webauthn-support": "supported", "webauthn-iuvpaa-support": "supported", "commit": "Sign in"}
        h = {**H, "Referer": str(r.url)}
        r = self.s.post(f"{GITHUB}/session", data=d, headers=h, allow_redirects=False); self._restore()
        self.log(f"  POST /session -> {r.status_code}")

        if r.status_code not in (301,302,303,307,308): raise RuntimeError(f"Login failed: {r.status_code}")
        loc = dec(r.headers.get("Location",""))

        if "/sessions/verified-device" in loc:
            self.log("  Device verification required")
            if not self.ep: raise RuntimeError("Need email password: account----pwd----email_pwd")
            r = self.s.get(loc, allow_redirects=False, headers=h); self._restore()
            self.log("  Fetching verification code...")
            code = get_code(self.em, self.ep, self.v)
            if not code: raise RuntimeError("No verification code received")
            self.log(f"  Verification code: {code}")
            vt = re.search(r'name="authenticity_token"\s+value="([^"]+)"', r.text)
            vd = {"authenticity_token": vt.group(1) if vt else f.get("authenticity_token",""), "otp": code}
            r = self.s.post(f"{GITHUB}/sessions/verified-device", data=vd, headers=h, allow_redirects=False); self._restore()
            self.log(f"  POST verify -> {r.status_code}")
            if r.status_code in (301,302,303,307,308): loc = dec(r.headers.get("Location",""))
        elif "/sessions/two-factor" in loc:
            raise RuntimeError("2FA required, not supported")
        else:
            self.log("  No verification needed")

        self.log("[4/5] OAuth consent...")
        r = self.s.get(loc, allow_redirects=False, headers=h); self._restore()
        if r.status_code in (301,302,303,307,308):
            loc = dec(r.headers.get("Location",""))
            self.log(f"  Auto-authorized -> {loc[:80]}...")
        else:
            m = re.search(r'<meta[^>]*http-equiv="refresh"[^>]*url=([^"\s]+)', r.text, re.IGNORECASE)
            if m:
                loc = m.group(1); self.log("  Already authorized (meta-refresh)")
            else:
                m = re.search(r'name="authenticity_token"\s+value="([^"]+)"', r.text)
                if m:
                    ad = {"authenticity_token": m.group(1), "authorize": "1"}
                    h2 = {**H, "Referer": f"{GITHUB}/login/oauth/authorize"}
                    r = self.s.post(f"{GITHUB}/login/oauth/authorize", data=ad, headers=h2, allow_redirects=False); self._restore()
                    self.log(f"  POST authorize -> {r.status_code}")
                    if r.status_code in (301,302,303,307,308): loc = dec(r.headers.get("Location",""))

        self.log("[5/5] Following callback chain...")
        for i in range(8):
            if not loc: break
            self._restore()
            full = loc
            if loc.startswith("/"):
                if "auth.opencode" in str(getattr(r,"url","")): full = f"{AUTH}{loc}"
                elif "github" in str(getattr(r,"url","")): full = f"{GITHUB}{loc}"
                else: full = f"{OPENCODE}{loc}"
            self.s.headers.update({"Sec-Fetch-Site": "cross-site"})
            r = self.s.get(full, allow_redirects=False, timeout=15)
            self.log(f"  [{i}] {r.status_code} {full[:100]}")
            if r.status_code in (301,302,303,307,308):
                loc = dec(r.headers.get("Location",""))
            else:
                m = re.search(r'<meta[^>]*http-equiv="refresh"[^>]*url=([^"\s]+)', r.text, re.IGNORECASE)
                if m: loc = m.group(1); continue
                break

        url = str(r.url)
        ws_id = url.split("/workspace/")[-1].split("/")[0].split("?")[0] if "/workspace/" in url else None
        ac = None
        for c in self.s.cookies:
            if c.name == "auth": ac = f"{c.name}={c.value}"
        return {"workspace_id": ws_id, "auth_cookie": ac, "account": self.gh}

def main():
    try:
        verbose = "-v" in sys.argv or "--verbose" in sys.argv
        args = [a for a in sys.argv[1:] if not a.startswith("-")]

        if not args:
            _print("Usage: python opencode_auth_extractor.py account----password [-v]")
            _print("       python opencode_auth_extractor.py account----password----email_pass [-v]")
            return

        p = args[0].split("----")
        account, password = p[0], p[1] if len(p)>1 else ""
        email_pass = p[2] if len(p)>2 else None

        ext = Extractor(account, password, email_pass, verbose=verbose)
        r = ext.run()

        ident = r["account"].replace("@", "_at_").replace(".", "_")
        fname = f"opencode_{ident}.json"
        with open(fname, "w", encoding="utf-8") as f:
            json.dump({"workspace_id": r["workspace_id"], "auth_cookie": r["auth_cookie"], "account": r["account"]}, f, indent=2)

        _print(f"\n{'='*50}")
        _print(f"  Account:      {r['account']}")
        _print(f"  Workspace ID: {r['workspace_id']}")
        _print(f"  Auth Cookie:  {r['auth_cookie'][:80] if r['auth_cookie'] else 'N/A'}...")
        _print(f"  Output:       {fname}")
        _print(f"{'='*50}")

    except Exception as e:
        _print(f"\nFATAL: {e}")
        traceback.print_exc()
        sys.exit(1)

if __name__ == "__main__":
    main()
