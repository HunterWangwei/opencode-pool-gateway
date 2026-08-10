package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var gatewayRoutes = []string{"/zen/go/v1/chat/completions", "/zen/go/v1/models", "/zen/v1/responses", "/zen/v1/models"}
var gatewayUpstream = "https://opencode.ai"

type GatewayConfig struct {
	Mode        string `json:"mode"`
	RetryLimit  int    `json:"retryLimit"`
	GlobalProxy string `json:"globalProxy"`
}
type gatewayManager struct {
	sync.RWMutex
	Config     GatewayConfig
	path       string
	roundRobin uint64
}
type AccessToken struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	TokenHash string    `json:"tokenHash,omitempty"`
	Prefix    string    `json:"prefix"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	LastUsed  time.Time `json:"lastUsed,omitempty"`
}
type tokenManager struct {
	sync.RWMutex
	Tokens []AccessToken `json:"tokens"`
	path   string
}
type RequestLog struct {
	ID             string    `json:"id"`
	Time           time.Time `json:"time"`
	Route          string    `json:"route"`
	Method         string    `json:"method"`
	Model          string    `json:"model,omitempty"`
	CredentialID   string    `json:"credentialId,omitempty"`
	CredentialName string    `json:"credentialName,omitempty"`
	Attempt        int       `json:"attempt"`
	StatusCode     int       `json:"statusCode"`
	InputTokens    int64     `json:"inputTokens,omitempty"`
	OutputTokens   int64     `json:"outputTokens,omitempty"`
	CacheRead      int64     `json:"cacheRead,omitempty"`
	CacheWrite     int64     `json:"cacheWrite,omitempty"`
	DurationMS     int64     `json:"durationMs"`
	ErrorBody      string    `json:"errorBody,omitempty"`
}
type requestLogStore struct {
	sync.RWMutex
	Logs []RequestLog
	path string
}

func atomicJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
func (g *gatewayManager) load(path string) error {
	g.path = path
	g.Config = GatewayConfig{Mode: "round_robin"}
	b, e := os.ReadFile(path)
	if errors.Is(e, os.ErrNotExist) {
		return atomicJSON(path, g.Config)
	}
	if e != nil {
		return e
	}
	if e = json.Unmarshal(b, &g.Config); e != nil {
		return e
	}
	return validateGateway(&g.Config)
}
func validateGateway(c *GatewayConfig) error {
	if c.Mode == "" {
		c.Mode = "round_robin"
	}
	if c.Mode != "round_robin" && c.Mode != "priority" {
		return errors.New("调度模式无效")
	}
	if c.RetryLimit < 0 {
		return errors.New("重试次数不能小于 0")
	}
	if c.GlobalProxy != "" {
		return validateProxy(c.GlobalProxy)
	}
	return nil
}
func validateProxy(raw string) error {
	u, e := url.Parse(raw)
	if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("代理地址仅支持 http:// 或 https://")
	}
	return nil
}
func (g *gatewayManager) settingsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		g.RLock()
		c := g.Config
		g.RUnlock()
		writeJSON(w, 200, c)
	case http.MethodPut:
		var c GatewayConfig
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&c) != nil {
			writeError(w, 400, "配置格式无效")
			return
		}
		if e := validateGateway(&c); e != nil {
			writeError(w, 400, e.Error())
			return
		}
		g.Lock()
		e := atomicJSON(g.path, c)
		if e == nil {
			g.Config = c
		}
		g.Unlock()
		if e != nil {
			writeError(w, 500, e.Error())
			return
		}
		writeJSON(w, 200, c)
	default:
		writeError(w, 405, "method not allowed")
	}
}

func (t *tokenManager) load(path string) error {
	t.path = path
	b, e := os.ReadFile(path)
	if errors.Is(e, os.ErrNotExist) {
		return atomicJSON(path, t)
	}
	if e != nil {
		return e
	}
	return json.Unmarshal(b, t)
}
func (t *tokenManager) public() []AccessToken {
	t.RLock()
	defer t.RUnlock()
	out := append([]AccessToken(nil), t.Tokens...)
	for i := range out {
		out[i].TokenHash = ""
	}
	return out
}
func tokenHash(v string) string { s := sha256.Sum256([]byte(v)); return hex.EncodeToString(s[:]) }
func (t *tokenManager) tokensHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, 200, t.public())
	case http.MethodPost:
		var in struct {
			Name string `json:"name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		in.Name = strings.TrimSpace(in.Name)
		if in.Name == "" {
			writeError(w, 400, "请输入令牌名称")
			return
		}
		plain := "gq_" + randomToken(32)
		item := AccessToken{ID: randomToken(12), Name: in.Name, TokenHash: tokenHash(plain), Prefix: plain[:10], Enabled: true, CreatedAt: time.Now()}
		t.Lock()
		t.Tokens = append(t.Tokens, item)
		e := atomicJSON(t.path, t)
		t.Unlock()
		if e != nil {
			writeError(w, 500, e.Error())
			return
		}
		item.TokenHash = ""
		writeJSON(w, 201, map[string]any{"token": plain, "item": item})
	default:
		writeError(w, 405, "method not allowed")
	}
}
func (t *tokenManager) tokenHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/tokens/")
	t.Lock()
	defer t.Unlock()
	idx := -1
	for i := range t.Tokens {
		if t.Tokens[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		writeError(w, 404, "令牌不存在")
		return
	}
	switch r.Method {
	case http.MethodPut:
		var in struct {
			Name    string `json:"name"`
			Enabled *bool  `json:"enabled"`
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil {
			writeError(w, 400, "格式无效")
			return
		}
		if strings.TrimSpace(in.Name) != "" {
			t.Tokens[idx].Name = strings.TrimSpace(in.Name)
		}
		if in.Enabled != nil {
			t.Tokens[idx].Enabled = *in.Enabled
		}
		if e := atomicJSON(t.path, t); e != nil {
			writeError(w, 500, e.Error())
			return
		}
		item := t.Tokens[idx]
		item.TokenHash = ""
		writeJSON(w, 200, item)
	case http.MethodDelete:
		t.Tokens = append(t.Tokens[:idx], t.Tokens[idx+1:]...)
		if e := atomicJSON(t.path, t); e != nil {
			writeError(w, 500, e.Error())
			return
		}
		w.WriteHeader(204)
	default:
		writeError(w, 405, "method not allowed")
	}
}
func bearer(r *http.Request) string {
	v := r.Header.Get("Authorization")
	if len(v) > 7 && strings.EqualFold(v[:7], "Bearer ") {
		return strings.TrimSpace(v[7:])
	}
	return ""
}
func (t *tokenManager) requireToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := tokenHash(bearer(r))
		t.Lock()
		ok := false
		for i := range t.Tokens {
			if t.Tokens[i].Enabled && t.Tokens[i].TokenHash == h {
				t.Tokens[i].LastUsed = time.Now()
				ok = true
				break
			}
		}
		if ok {
			_ = atomicJSON(t.path, t)
		}
		t.Unlock()
		if !ok {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeError(w, 401, "访问令牌无效")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *requestLogStore) load(path string) error {
	l.path = path
	b, e := os.ReadFile(path)
	if errors.Is(e, os.ErrNotExist) {
		return nil
	}
	if e != nil {
		return e
	}
	for _, line := range bytes.Split(b, []byte("\n")) {
		var x RequestLog
		if json.Unmarshal(line, &x) == nil {
			l.Logs = append(l.Logs, x)
		}
	}
	if len(l.Logs) > 2000 {
		l.Logs = l.Logs[len(l.Logs)-2000:]
	}
	return nil
}
func (l *requestLogStore) add(x RequestLog) {
	x.ID = randomToken(9)
	l.Lock()
	l.Logs = append(l.Logs, x)
	if len(l.Logs) > 2000 {
		l.Logs = l.Logs[len(l.Logs)-2000:]
	}
	b, _ := json.Marshal(x)
	f, e := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e == nil {
		_, _ = f.Write(append(b, '\n'))
		_ = f.Close()
	}
	l.Unlock()
}
func (l *requestLogStore) logsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		l.Lock()
		l.Logs = nil
		e := os.WriteFile(l.path, nil, 0600)
		l.Unlock()
		if e != nil {
			writeError(w, 500, e.Error())
			return
		}
		w.WriteHeader(204)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, 405, "method not allowed")
		return
	}
	n, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if n <= 0 || n > 1000 {
		n = 200
	}
	l.RLock()
	start := len(l.Logs) - n
	if start < 0 {
		start = 0
	}
	out := append([]RequestLog(nil), l.Logs[start:]...)
	l.RUnlock()
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	writeJSON(w, 200, out)
}

func eligibleAccounts(path string) []Account {
	store.RLock()
	defer store.RUnlock()
	out := []Account{}
	isGo := strings.HasPrefix(path, "/zen/go/")
	for _, a := range store.Accounts {
		if a.APIKey == "" {
			continue
		}
		if isGo && a.GoError != "" && !a.GoAvailable {
			continue
		}
		if !isGo && a.ZenError != "" && !a.ZenAvailable {
			continue
		}
		out = append(out, a)
	}
	if len(out) == 0 {
		for _, a := range store.Accounts {
			if a.APIKey != "" {
				out = append(out, a)
			}
		}
	}
	return out
}
func (g *gatewayManager) candidates(path string) ([]Account, int) {
	a := eligibleAccounts(path)
	g.RLock()
	c := g.Config
	g.RUnlock()
	if c.Mode == "priority" {
		sort.SliceStable(a, func(i, j int) bool { return a[i].Priority < a[j].Priority })
	} else if len(a) > 1 {
		start := int(atomic.AddUint64(&g.roundRobin, 1)-1) % len(a)
		a = append(a[start:], a[:start]...)
	}
	limit := len(a)
	if c.RetryLimit > 0 && c.RetryLimit < limit {
		limit = c.RetryLimit
	}
	return a[:limit], limit
}

var hopHeaders = map[string]bool{"connection": true, "proxy-connection": true, "keep-alive": true, "proxy-authenticate": true, "proxy-authorization": true, "te": true, "trailer": true, "transfer-encoding": true, "upgrade": true}

func copyHeaders(dst, src http.Header) {
	for k, v := range src {
		if hopHeaders[strings.ToLower(k)] {
			continue
		}
		for _, x := range v {
			dst.Add(k, x)
		}
	}
}
func transportFor(raw string) (*http.Transport, error) {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if raw == "" {
		return tr, nil
	}
	u, e := url.Parse(raw)
	if e != nil {
		return nil, e
	}
	tr.Proxy = http.ProxyURL(u)
	return tr, nil
}
func shouldRetry(code int) bool {
	return code == 401 || code == 403 || code == 408 || code == 429 || code >= 500
}
func modelFromBody(b []byte) string {
	var x struct {
		Model string `json:"model"`
	}
	_ = json.Unmarshal(b, &x)
	return x.Model
}
func usageFromBody(b []byte) (in, out, read, write int64) {
	var x map[string]any
	if json.Unmarshal(b, &x) != nil {
		return
	}
	u, _ := x["usage"].(map[string]any)
	num := func(keys ...string) int64 {
		for _, k := range keys {
			if v, ok := u[k].(float64); ok {
				return int64(v)
			}
		}
		return 0
	}
	in = num("prompt_tokens", "input_tokens")
	out = num("completion_tokens", "output_tokens")
	read = num("cache_read_input_tokens", "cache_read_tokens")
	write = num("cache_creation_input_tokens", "cache_write_tokens")
	if d, ok := u["input_tokens_details"].(map[string]any); ok {
		if v, ok := d["cached_tokens"].(float64); ok {
			read = int64(v)
		}
	}
	return
}
func cleanError(b []byte) string {
	s := strings.TrimSpace(string(b))
	s = regexp.MustCompile(`(?i)(bearer\s+|sk-|gq_)[A-Za-z0-9._~+/-]{8,}`).ReplaceAllString(s, "$1[REDACTED]")
	s = regexp.MustCompile(`(?i)(authorization|api[_-]?key|cookie)(["'\s:=]+)[^"'\s,}]+`).ReplaceAllString(s, "$1$2[REDACTED]")
	if len(s) > 4096 {
		s = s[:4096] + "…"
	}
	return s
}
func (g *gatewayManager) proxyHandler(w http.ResponseWriter, r *http.Request) {
	body, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<20))
	if e != nil {
		writeError(w, 400, "读取请求体失败")
		return
	}
	accounts, _ := g.candidates(r.URL.Path)
	if len(accounts) == 0 {
		writeError(w, 503, "没有可用的 OpenCode API Key 凭证")
		return
	}
	model := modelFromBody(body)
	var lastStatus int
	var lastHeader http.Header
	var lastBody []byte
	for i, a := range accounts {
		started := time.Now()
		target := gatewayUpstream + r.URL.EscapedPath()
		if r.URL.RawQuery != "" {
			target += "?" + r.URL.RawQuery
		}
		req, _ := http.NewRequestWithContext(r.Context(), r.Method, target, bytes.NewReader(body))
		copyHeaders(req.Header, r.Header)
		req.Header.Del("Authorization")
		req.Header.Set("Authorization", "Bearer "+a.APIKey)
		g.RLock()
		proxy := g.Config.GlobalProxy
		g.RUnlock()
		if a.ProxyURL != "" {
			proxy = a.ProxyURL
		}
		tr, te := transportFor(proxy)
		var resp *http.Response
		if te == nil {
			resp, te = (&http.Client{Transport: tr, Timeout: 0}).Do(req)
		}
		entry := RequestLog{Time: time.Now(), Route: r.URL.Path, Method: r.Method, Model: model, CredentialID: a.ID, CredentialName: a.Name, Attempt: i + 1, DurationMS: time.Since(started).Milliseconds()}
		if te != nil {
			entry.ErrorBody = te.Error()
			requestLogs.add(entry)
			lastBody = []byte(te.Error())
			lastStatus = 502
			continue
		}
		rb, re := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		entry.StatusCode = resp.StatusCode
		entry.InputTokens, entry.OutputTokens, entry.CacheRead, entry.CacheWrite = usageFromBody(rb)
		if re != nil {
			entry.ErrorBody = re.Error()
		} else if resp.StatusCode >= 400 {
			entry.ErrorBody = cleanError(rb)
		}
		requestLogs.add(entry)
		lastStatus, lastHeader, lastBody = resp.StatusCode, resp.Header, rb
		if re == nil && !shouldRetry(resp.StatusCode) {
			copyHeaders(w.Header(), resp.Header)
			w.WriteHeader(resp.StatusCode)
			_, _ = w.Write(rb)
			return
		}
	}
	if lastHeader != nil {
		copyHeaders(w.Header(), lastHeader)
	}
	if lastStatus == 0 {
		lastStatus = 502
	}
	w.WriteHeader(lastStatus)
	_, _ = w.Write(lastBody)
}
