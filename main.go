package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed web/* opencode/opencode_auth_extractor.py
var embeddedFiles embed.FS

const (
	listenAddr   = ":8787"
	goModelsURL  = "https://opencode.ai/zen/go/v1/models"
	zenModelsURL = "https://opencode.ai/zen/v1/models"
	refreshEvery = 5 * time.Minute
)

const (
	userEmailServerID = "44e81edfbd76665bfe0657aa7f751d7e73ab8d4a1b00f5b9909ba57ece0cf874"
	keyListServerID   = "c22cd964237ba79f2f9b95faa2a14b804f870d1bab49279463379cc6a0fd0c85"
)

type UsageWindow struct {
	UsagePercent  float64   `json:"usagePercent"`
	ResetInSecond int       `json:"resetInSec"`
	ResetAt       time.Time `json:"resetAt"`
	LimitUSD      float64   `json:"limitUsd"`
}

type ModelInfo struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Provider          string `json:"provider,omitempty"`
	Deprecated        bool   `json:"deprecated,omitempty"`
	DeprecationDate   string `json:"deprecationDate,omitempty"`
	Available         bool   `json:"available"`
	UnavailableReason string `json:"unavailableReason,omitempty"`
}

var zenFreeModels = map[string]bool{
	"big-pickle":             true,
	"deepseek-v4-flash-free": true,
	"mimo-v2.5-free":         true,
	"laguna-s-2.1-free":      true,
	"ling-3.0-tiny-free":     true,
	"longcat-2.0-free":       true,
	"north-mini-code-free":   true,
	"nemotron-3-ultra-free":  true,
}

func normalizeModelID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	if slash := strings.LastIndex(id, "/"); slash >= 0 {
		id = id[slash+1:]
	}
	return id
}

func isZenFreeModel(id string) bool { return zenFreeModels[normalizeModelID(id)] }

func annotateModelAvailability(models []ModelInfo, available func(string) (bool, string)) {
	for i := range models {
		models[i].Available, models[i].UnavailableReason = available(models[i].ID)
	}
}

var zenDeprecationDates = map[string]string{
	"gpt-5.2-codex":      "2026-07-23",
	"gpt-5.1-codex":      "2026-07-23",
	"gpt-5.1-codex-max":  "2026-07-23",
	"gpt-5.1-codex-mini": "2026-07-23",
	"gpt-5-codex":        "2026-07-23",
	"claude-opus-4.1":    "2026-08-05",
	"claude-opus-4-1":    "2026-08-05",
	"claude-sonnet-4":    "2026-06-15",
	"claude-haiku-3.5":   "2026-02-16",
	"claude-haiku-3-5":   "2026-02-16",
	"gemini-3-pro":       "2026-03-09",
	"minimax-m2.5":       "2026-08-05",
	"minimax-m2-5":       "2026-08-05",
	"minimax-m2.1":       "2026-03-15",
	"minimax-m2-1":       "2026-03-15",
	"glm-5":              "2026-05-14",
	"glm-4.7":            "2026-03-15",
	"glm-4-7":            "2026-03-15",
	"glm-4.6":            "2026-03-15",
	"glm-4-6":            "2026-03-15",
	"kimi-k2.5":          "2026-08-05",
	"kimi-k2-5":          "2026-08-05",
	"kimi-k2-thinking":   "2026-03-06",
	"kimi-k2":            "2026-03-06",
	"qwen3-coder-480b":   "2026-02-06",
}

type Account struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	Type              string       `json:"type"`
	WorkspaceID       string       `json:"workspaceId"`
	APIKey            string       `json:"apiKey"`
	AuthCookie        string       `json:"authCookie"`
	Status            string       `json:"status"`
	Error             string       `json:"error,omitempty"`
	ModelCount        int          `json:"modelCount"`
	GoModelCount      int          `json:"goModelCount"`
	ZenModelCount     int          `json:"zenModelCount"`
	Rolling           *UsageWindow `json:"rolling,omitempty"`
	Weekly            *UsageWindow `json:"weekly,omitempty"`
	Monthly           *UsageWindow `json:"monthly,omitempty"`
	ZenBalance        float64      `json:"zenBalance,omitempty"`
	GoAvailable       bool         `json:"goAvailable"`
	ZenAvailable      bool         `json:"zenAvailable"`
	ZenBillingEnabled bool         `json:"zenBillingEnabled"`
	GoError           string       `json:"goError,omitempty"`
	ZenError          string       `json:"zenError,omitempty"`
	LastChecked       time.Time    `json:"lastChecked,omitempty"`
	Priority          int          `json:"priority"`
	ProxyURL          string       `json:"proxyUrl,omitempty"`
	APIOnly           bool         `json:"apiOnly,omitempty"`
	ProbeService      string       `json:"probeService,omitempty"`
}

type PublicAccount struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	Type              string       `json:"type"`
	WorkspaceID       string       `json:"workspaceId"`
	Status            string       `json:"status"`
	Error             string       `json:"error,omitempty"`
	ModelCount        int          `json:"modelCount"`
	GoModelCount      int          `json:"goModelCount"`
	ZenModelCount     int          `json:"zenModelCount"`
	Rolling           *UsageWindow `json:"rolling,omitempty"`
	Weekly            *UsageWindow `json:"weekly,omitempty"`
	Monthly           *UsageWindow `json:"monthly,omitempty"`
	ZenBalance        float64      `json:"zenBalance,omitempty"`
	GoAvailable       bool         `json:"goAvailable"`
	ZenAvailable      bool         `json:"zenAvailable"`
	ZenBillingEnabled bool         `json:"zenBillingEnabled"`
	GoError           string       `json:"goError,omitempty"`
	ZenError          string       `json:"zenError,omitempty"`
	LastChecked       time.Time    `json:"lastChecked,omitempty"`
	HasAPIKey         bool         `json:"hasApiKey"`
	HasCookie         bool         `json:"hasCookie"`
	Priority          int          `json:"priority"`
	ProxyURL          string       `json:"proxyUrl,omitempty"`
	APIOnly           bool         `json:"apiOnly,omitempty"`
	ProbeService      string       `json:"probeService,omitempty"`
}

type DiscoveredAPIKey struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email,omitempty"`
	Key     string `json:"key"`
	Display string `json:"display"`
}

type AccountDiscovery struct {
	Email    string             `json:"email,omitempty"`
	APIKeys  []DiscoveredAPIKey `json:"apiKeys"`
	Warnings []string           `json:"warnings,omitempty"`
}

type LoginExtraction struct {
	Account     string `json:"account"`
	WorkspaceID string `json:"workspace_id"`
	AuthCookie  string `json:"auth_cookie"`
	TempDir     string `json:"-"`
}

type APIKeyProbeResult struct {
	WorkspaceID string `json:"workspaceId,omitempty"`
	Service     string `json:"service"`
	Enabled     bool   `json:"enabled"`
	Message     string `json:"message"`
}

type APIKeyEntitlements struct {
	WorkspaceID string
	GoEnabled   bool
	ZenEnabled  bool
}

type accountStore struct {
	sync.RWMutex
	Accounts []Account `json:"accounts"`
	path     string
}

type loginFailure struct {
	Count       int
	FirstFailed time.Time
}

type authConfig struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"passwordHash"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type authManager struct {
	sync.Mutex
	username     string
	passwordHash string
	path         string
	sessions     map[string]time.Time
	failures     map[string]loginFailure
}

var store accountStore
var gateway gatewayManager
var accessTokens tokenManager
var requestLogs requestLogStore
var client = &http.Client{Timeout: 18 * time.Second}
var discoveryServerURL = "https://opencode.ai/_server"
var apiKeyProbeBase = "https://opencode.ai"
var shutdownOnce sync.Once
var shutdownSignal = make(chan struct{})
var version = "dev"
var commit = "unknown"
var buildDate = "unknown"

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-version") {
		fmt.Printf("OpenCode Pool Gateway %s (%s, %s)\n", version, commit, buildDate)
		return
	}
	webRoot, err := fs.Sub(embeddedFiles, "web")
	if err != nil {
		log.Fatal(err)
	}
	store.path = dataPath()
	if err := store.load(); err != nil {
		log.Printf("读取账号配置失败: %v", err)
	}
	if err := gateway.load(gatewayDataPath()); err != nil {
		log.Fatal("读取网关配置失败: ", err)
	}
	if err := accessTokens.load(tokenDataPath()); err != nil {
		log.Fatal("读取访问令牌失败: ", err)
	}
	if err := requestLogs.load(requestLogPath()); err != nil {
		log.Fatal("读取请求日志失败: ", err)
	}
	auth, generatedPassword, err := loadAuthManager(authDataPath())
	if err != nil {
		log.Fatal("读取登录配置失败: ", err)
	}
	if generatedPassword != "" {
		log.Printf("data/auth.json 不存在，已生成并保存初始登录凭证")
		log.Printf("登录账号: %s", auth.username)
		log.Printf("初始密码: %s", generatedPassword)
		log.Printf("请登录后立即在设置页面修改账号和密码")
	}

	appMux := http.NewServeMux()
	appMux.HandleFunc("/api/accounts", accountsHandler)
	appMux.HandleFunc("/api/accounts/discover", accountDiscoverHandler)
	appMux.HandleFunc("/api/accounts/login-extract", accountLoginExtractHandler)
	appMux.HandleFunc("/api/accounts/", accountHandler)
	appMux.HandleFunc("/api/refresh", refreshHandler)
	appMux.HandleFunc("/api/auth", auth.credentialsHandler)
	appMux.HandleFunc("/api/version", versionHandler)
	appMux.HandleFunc("/api/settings", gateway.settingsHandler)
	appMux.HandleFunc("/api/tokens", accessTokens.tokensHandler)
	appMux.HandleFunc("/api/tokens/", accessTokens.tokenHandler)
	appMux.HandleFunc("/api/logs", requestLogs.logsHandler)
	appMux.HandleFunc("/api/shutdown", shutdownHandler)
	appMux.Handle("/", http.FileServer(http.FS(webRoot)))
	mux := http.NewServeMux()
	mux.HandleFunc("/login", auth.loginHandler)
	mux.HandleFunc("/logout", auth.logoutHandler)
	for _, route := range gatewayRoutes {
		mux.Handle(route, accessTokens.requireToken(http.HandlerFunc(gateway.proxyHandler)))
	}
	mux.Handle("/", auth.requireAuth(appMux))

	server := &http.Server{Addr: listenAddr, Handler: securityHeaders(mux), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()
	go refreshAll()
	go refreshScheduler()
	printConsoleHelp()
	go readConsoleCommands()
	<-shutdownSignal
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
	log.Println("OpenCode Pool Gateway 已安全退出")
}

func loadAuthManager(path string) (*authManager, string, error) {
	b, err := os.ReadFile(path)
	if err == nil {
		var config authConfig
		if json.Unmarshal(b, &config) != nil || strings.TrimSpace(config.Username) == "" || !validPasswordHash(config.PasswordHash) {
			return nil, "", errors.New("data/auth.json 格式无效")
		}
		return newAuthManagerFromHash(config.Username, config.PasswordHash, path), "", nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, "", err
	}
	username := "admin"
	password := randomToken(18)
	if err := validateCredentials(username, password); err != nil {
		return nil, "", err
	}
	hash := hashPassword(password)
	manager := newAuthManagerFromHash(username, hash, path)
	if err := saveAuthConfig(path, authConfig{Username: username, PasswordHash: hash, UpdatedAt: time.Now()}); err != nil {
		return nil, "", err
	}
	return manager, password, nil
}

func newAuthManager(username, password string) *authManager {
	return newAuthManagerFromHash(username, hashPassword(password), "")
}

func newAuthManagerFromHash(username, passwordHash, path string) *authManager {
	return &authManager{username: username, passwordHash: passwordHash, path: path, sessions: make(map[string]time.Time), failures: make(map[string]loginFailure)}
}

func (a *authManager) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.authenticated(r) {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, http.StatusUnauthorized, "请先登录")
			return
		}
		nextURL := r.URL.RequestURI()
		http.Redirect(w, r, "/login?next="+url.QueryEscape(nextURL), http.StatusSeeOther)
	})
}

func (a *authManager) authenticated(r *http.Request) bool {
	cookie, err := r.Cookie("opg_session")
	if err != nil || cookie.Value == "" {
		return false
	}
	a.Lock()
	defer a.Unlock()
	expires, ok := a.sessions[cookie.Value]
	if !ok || time.Now().After(expires) {
		delete(a.sessions, cookie.Value)
		return false
	}
	return true
}

func (a *authManager) loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		body, err := embeddedFiles.ReadFile("web/login.html")
		if err != nil {
			http.Error(w, "login page unavailable", http.StatusInternalServerError)
			return
		}
		errorMessage := ""
		switch r.URL.Query().Get("error") {
		case "invalid":
			errorMessage = "账号或密码不正确"
		case "locked":
			errorMessage = "失败次数过多，请 5 分钟后重试"
		}
		noticeMessage := ""
		if r.URL.Query().Get("changed") == "1" {
			noticeMessage = "登录账号和密码已更新，请重新登录"
		} else if r.URL.Query().Get("expired") == "1" {
			noticeMessage = "管理登录已失效，请重新登录后继续"
		}
		page := strings.ReplaceAll(string(body), "{{ERROR}}", html.EscapeString(errorMessage))
		page = strings.ReplaceAll(page, "{{NOTICE}}", html.EscapeString(noticeMessage))
		page = strings.ReplaceAll(page, "{{NEXT}}", html.EscapeString(safeNext(r.URL.Query().Get("next"))))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = io.WriteString(w, page)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	clientIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	if a.loginBlocked(clientIP) {
		http.Redirect(w, r, "/login?error=locked", http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/login?error=invalid", http.StatusSeeOther)
		return
	}
	a.Lock()
	username, passwordHash := a.username, a.passwordHash
	a.Unlock()
	if !secureEqual(r.FormValue("username"), username) || !verifyPassword(r.FormValue("password"), passwordHash) {
		a.recordFailure(clientIP)
		time.Sleep(250 * time.Millisecond)
		http.Redirect(w, r, "/login?error=invalid", http.StatusSeeOther)
		return
	}
	a.Lock()
	delete(a.failures, clientIP)
	token := randomToken(32)
	a.sessions[token] = time.Now().Add(24 * time.Hour)
	a.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "opg_session", Value: token, Path: "/", MaxAge: 86400, HttpOnly: true, Secure: requestIsHTTPS(r), SameSite: http.SameSiteStrictMode})
	nextURL := safeNext(r.FormValue("next"))
	http.Redirect(w, r, nextURL, http.StatusSeeOther)
}

func (a *authManager) credentialsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.Lock()
		username := a.username
		a.Unlock()
		writeJSON(w, http.StatusOK, map[string]string{"username": username})
	case http.MethodPut:
		var input struct {
			CurrentPassword string `json:"currentPassword"`
			Username        string `json:"username"`
			NewPassword     string `json:"newPassword"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 16<<10)).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "请求数据无效")
			return
		}
		input.Username = strings.TrimSpace(input.Username)
		if err := validateCredentials(input.Username, input.NewPassword); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		a.Lock()
		defer a.Unlock()
		if !verifyPassword(input.CurrentPassword, a.passwordHash) {
			writeError(w, http.StatusUnauthorized, "当前密码不正确")
			return
		}
		hash := hashPassword(input.NewPassword)
		config := authConfig{Username: input.Username, PasswordHash: hash, UpdatedAt: time.Now()}
		if err := saveAuthConfig(a.path, config); err != nil {
			writeError(w, http.StatusInternalServerError, "保存登录配置失败: "+err.Error())
			return
		}
		a.username, a.passwordHash = input.Username, hash
		a.sessions = make(map[string]time.Time)
		a.failures = make(map[string]loginFailure)
		writeJSON(w, http.StatusOK, map[string]bool{"reauthenticate": true})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *authManager) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if cookie, err := r.Cookie("opg_session"); err == nil {
		a.Lock()
		delete(a.sessions, cookie.Value)
		a.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "opg_session", Path: "/", MaxAge: -1, HttpOnly: true, Secure: requestIsHTTPS(r), SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *authManager) loginBlocked(ip string) bool {
	a.Lock()
	defer a.Unlock()
	failure, ok := a.failures[ip]
	if !ok {
		return false
	}
	if time.Since(failure.FirstFailed) > 5*time.Minute {
		delete(a.failures, ip)
		return false
	}
	return failure.Count >= 5
}

func (a *authManager) recordFailure(ip string) {
	a.Lock()
	defer a.Unlock()
	failure := a.failures[ip]
	if failure.Count == 0 || time.Since(failure.FirstFailed) > 5*time.Minute {
		failure = loginFailure{FirstFailed: time.Now()}
	}
	failure.Count++
	a.failures[ip] = failure
}

func secureEqual(left, right string) bool {
	leftHash, rightHash := sha256.Sum256([]byte(left)), sha256.Sum256([]byte(right))
	return subtle.ConstantTimeCompare(leftHash[:], rightHash[:]) == 1
}

func validateCredentials(username, password string) error {
	if username == "" || len(username) > 64 {
		return errors.New("用户名长度必须为 1 到 64 个字符")
	}
	if len(password) < 8 || len(password) > 256 {
		return errors.New("密码长度必须为 8 到 256 个字符")
	}
	return nil
}

func hashPassword(password string) string {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		log.Fatal(err)
	}
	const iterations = 600000
	derived := pbkdf2SHA256([]byte(password), salt, iterations, 32)
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", iterations, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(derived))
}

func verifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations < 100000 || iterations > 2000000 {
		return false
	}
	salt, saltErr := base64.RawStdEncoding.DecodeString(parts[2])
	want, hashErr := base64.RawStdEncoding.DecodeString(parts[3])
	if saltErr != nil || hashErr != nil || len(salt) < 16 || len(want) != 32 {
		return false
	}
	got := pbkdf2SHA256([]byte(password), salt, iterations, len(want))
	return subtle.ConstantTimeCompare(got, want) == 1
}

func validPasswordHash(encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	salt, saltErr := base64.RawStdEncoding.DecodeString(parts[2])
	hash, hashErr := base64.RawStdEncoding.DecodeString(parts[3])
	return err == nil && iterations >= 100000 && iterations <= 2000000 && saltErr == nil && len(salt) >= 16 && hashErr == nil && len(hash) == 32
}

func pbkdf2SHA256(password, salt []byte, iterations, keyLength int) []byte {
	hashLength := sha256.Size
	blocks := (keyLength + hashLength - 1) / hashLength
	derived := make([]byte, 0, blocks*hashLength)
	for block := 1; block <= blocks; block++ {
		mac := hmac.New(sha256.New, password)
		_, _ = mac.Write(salt)
		var counter [4]byte
		binary.BigEndian.PutUint32(counter[:], uint32(block))
		_, _ = mac.Write(counter[:])
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			_, _ = mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		derived = append(derived, t...)
	}
	return derived[:keyLength]
}

func saveAuthConfig(path string, config authConfig) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		return err
	}
	return os.Chmod(path, 0600)
}

func randomToken(size int) string {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		log.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer)
}

func safeNext(value string) string {
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return "/"
	}
	return value
}

func requestIsHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") || os.Getenv("OPG_COOKIE_SECURE") == "1"
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'; img-src 'self' data:; frame-ancestors 'none'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, 405, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"version": version, "commit": commit, "buildDate": buildDate})
}

func dataPath() string {
	executable, err := os.Executable()
	if err != nil {
		return filepath.Join("data", "accounts.json")
	}
	return filepath.Join(filepath.Dir(executable), "data", "accounts.json")
}

func authDataPath() string {
	return filepath.Join(filepath.Dir(dataPath()), "auth.json")
}

func gatewayDataPath() string { return filepath.Join(filepath.Dir(dataPath()), "gateway.json") }
func tokenDataPath() string   { return filepath.Join(filepath.Dir(dataPath()), "tokens.json") }
func requestLogPath() string  { return filepath.Join(filepath.Dir(dataPath()), "requests.jsonl") }

func legacyDataPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "OpenCode Pool Gateway", "accounts.json")
}

func (s *accountStore) load() error {
	s.Lock()
	defer s.Unlock()
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		legacy := legacyDataPath()
		if legacy == "" || legacy == s.path {
			s.Accounts = []Account{}
			return nil
		}
		b, err = os.ReadFile(legacy)
		if errors.Is(err, os.ErrNotExist) {
			s.Accounts = []Account{}
			return nil
		}
		if err != nil {
			return err
		}
		defer func() {
			if saveErr := s.saveLocked(); saveErr == nil {
				_ = os.Remove(legacy)
			}
		}()
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &s.Accounts); err != nil {
		return err
	}
	if s.Accounts == nil {
		s.Accounts = []Account{}
	}
	return nil
}

func (s *accountStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.Accounts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0600)
}

func accountDiscoverHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var input struct{ WorkspaceID, AuthCookie string }
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "请求数据无效")
		return
	}
	input.WorkspaceID = strings.TrimSpace(input.WorkspaceID)
	input.AuthCookie = strings.TrimSpace(input.AuthCookie)
	if input.WorkspaceID == "" || input.AuthCookie == "" {
		writeError(w, http.StatusBadRequest, "Workspace ID 和 auth Cookie 均为必填")
		return
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(input.WorkspaceID) {
		writeError(w, http.StatusBadRequest, "Workspace ID 格式无效")
		return
	}
	discovery, err := discoverAccount(input.WorkspaceID, input.AuthCookie)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, discovery)
}

func accountLoginExtractHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var input struct {
		Account       string `json:"account"`
		Password      string `json:"password"`
		EmailPassword string `json:"emailPassword"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "登录参数无效")
		return
	}
	input.Account = strings.TrimSpace(input.Account)
	if input.Account == "" || input.Password == "" {
		writeError(w, http.StatusBadRequest, "账号和密码均为必填")
		return
	}
	extracted, err := runAuthExtractor(input.Account, input.Password, input.EmailPassword)
	input.Password, input.EmailPassword = "", ""
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if extracted.WorkspaceID == "" || extracted.AuthCookie == "" {
		writeError(w, http.StatusBadGateway, "登录成功，但未提取到 Workspace ID 或 auth Cookie；脚本文件已保留在: "+extracted.TempDir)
		return
	}
	discovery, discoverErr := discoverAccount(extracted.WorkspaceID, extracted.AuthCookie)
	if discoverErr != nil {
		writeError(w, http.StatusBadGateway, "已取得登录会话，但自动读取账号信息失败: "+discoverErr.Error())
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"account":     extracted.Account,
		"workspaceId": extracted.WorkspaceID,
		"authCookie":  extracted.AuthCookie,
		"email":       discovery.Email,
		"apiKeys":     discovery.APIKeys,
		"warnings":    discovery.Warnings,
	})
}

func runAuthExtractor(account, password, emailPassword string) (LoginExtraction, error) {
	if strings.Contains(account, "----") || strings.Contains(password, "----") || strings.Contains(emailPassword, "----") {
		return LoginExtraction{}, errors.New("账号或密码不能包含协议分隔符 ----")
	}
	python, prefix, err := findPython()
	if err != nil {
		return LoginExtraction{}, err
	}
	script, err := embeddedFiles.ReadFile("opencode/opencode_auth_extractor.py")
	if err != nil {
		return LoginExtraction{}, errors.New("读取内置登录脚本失败")
	}
	programDir := filepath.Dir(filepath.Dir(dataPath()))
	tempRoot := filepath.Join(programDir, "temp")
	if err := os.MkdirAll(tempRoot, 0700); err != nil {
		return LoginExtraction{}, errors.New("创建程序临时目录失败")
	}
	tempDir, err := os.MkdirTemp(tempRoot, "opencode-auth-")
	if err != nil {
		return LoginExtraction{}, errors.New("创建登录脚本临时目录失败")
	}
	scriptPath := filepath.Join(tempDir, "opencode_auth_extractor.py")
	if err := os.WriteFile(scriptPath, script, 0600); err != nil {
		return LoginExtraction{}, errors.New("准备登录脚本失败")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	credential := account + "----" + password
	if emailPassword != "" {
		credential += "----" + emailPassword
	}
	args := append(append([]string{}, prefix...), scriptPath, credential)
	cmd := exec.CommandContext(ctx, python, args...)
	cmd.Dir = tempDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	_ = os.WriteFile(filepath.Join(tempDir, "stdout.log"), stdout.Bytes(), 0600)
	_ = os.WriteFile(filepath.Join(tempDir, "stderr.log"), stderr.Bytes(), 0600)
	if runErr != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return LoginExtraction{}, fmt.Errorf("协议登录超时，请检查网络或邮箱验证码；脚本文件已保留在: %s", tempDir)
		}
		message := "协议登录失败"
		if match := regexp.MustCompile(`(?m)^FATAL:\s*(.+)$`).FindStringSubmatch(stdout.String()); len(match) == 2 {
			message += ": " + strings.TrimSpace(match[1])
		}
		if strings.Contains(stderr.String(), "requests not installed") {
			message = "Python 缺少 requests 依赖，请执行 pip install requests"
		}
		return LoginExtraction{}, fmt.Errorf("%s；脚本文件已保留在: %s", message, tempDir)
	}
	files, err := filepath.Glob(filepath.Join(tempDir, "opencode_*.json"))
	if err != nil || len(files) != 1 {
		return LoginExtraction{}, fmt.Errorf("登录脚本未生成唯一的账号结果文件；脚本文件已保留在: %s", tempDir)
	}
	resultBody, err := os.ReadFile(files[0])
	if err != nil {
		return LoginExtraction{}, errors.New("读取登录脚本结果失败")
	}
	var result LoginExtraction
	if err := json.Unmarshal(resultBody, &result); err != nil {
		return LoginExtraction{}, fmt.Errorf("登录脚本结果格式无效；脚本文件已保留在: %s", tempDir)
	}
	result.TempDir = tempDir
	return result, nil
}

func findPython() (string, []string, error) {
	for _, candidate := range []struct {
		name   string
		prefix []string
	}{{"python3", nil}, {"python", nil}, {"py", []string{"-3"}}} {
		if path, err := exec.LookPath(candidate.name); err == nil {
			args := append(append([]string{}, candidate.prefix...), "-c", "import requests")
			if exec.Command(path, args...).Run() == nil {
				return path, candidate.prefix, nil
			}
		}
	}
	return "", nil, errors.New("未找到可用的 Python 3 + requests，请先安装 Python 并执行 pip install requests")
}

func discoverAccount(workspaceID, authCookie string) (AccountDiscovery, error) {
	result := AccountDiscovery{APIKeys: []DiscoveredAPIKey{}}
	emailBody, err := callOpenCodeServerFunction(userEmailServerID, workspaceID, authCookie)
	if err != nil {
		return result, fmt.Errorf("自动读取账号邮箱失败: %w", err)
	}
	result.Email = parseDiscoveredEmail(emailBody)
	if result.Email == "" {
		result.Warnings = append(result.Warnings, "未读取到账号邮箱，可手动填写显示名称")
	}
	keyBody, err := callOpenCodeServerFunction(keyListServerID, workspaceID, authCookie)
	if err != nil {
		result.Warnings = append(result.Warnings, "未读取到 API Key，可手动填写")
		return result, nil
	}
	result.APIKeys = parseDiscoveredAPIKeys(keyBody)
	if len(result.APIKeys) == 0 {
		result.Warnings = append(result.Warnings, "当前账号没有可自动读取的 API Key，可留空或手动填写")
	}
	return result, nil
}

func callOpenCodeServerFunction(serverID, workspaceID, authCookie string) ([]byte, error) {
	body, _ := json.Marshal(map[string]any{
		"t": map[string]any{
			"t": 9,
			"i": 0,
			"l": 1,
			"a": []any{map[string]any{"t": 1, "s": workspaceID}},
			"o": 0,
		},
		"f": 31,
		"m": []any{},
	})
	req, err := http.NewRequest(http.MethodPost, discoveryServerURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("X-Server-Id", serverID)
	req.Header.Set("X-Server-Instance", "server-fn:1")
	req.Header.Set("Cookie", normalizeAuthCookie(authCookie))
	resp, err := client.Do(req)
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "authorize") || strings.Contains(lower, "login") {
			return nil, errors.New("auth Cookie 已失效或无权访问该 Workspace")
		}
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	finalURL := ""
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.Path
	}
	text := strings.ToLower(strings.TrimSpace(string(data)))
	if strings.Contains(finalURL, "authorize") || strings.Contains(finalURL, "login") || strings.Contains(text, "sign in") || strings.Contains(text, "login required") {
		return nil, errors.New("auth Cookie 已失效或无权访问该 Workspace")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, errors.New("auth Cookie 已失效或 Workspace ID 不匹配")
		}
		if resp.StatusCode == http.StatusInternalServerError {
			return nil, errors.New("OpenCode 账号接口调用失败，请确认 Cookie 完整且 Workspace ID 属于当前账号")
		}
		return nil, fmt.Errorf("OpenCode 返回 HTTP %d", resp.StatusCode)
	}
	if len(data) == 0 {
		return nil, errors.New("OpenCode 返回空数据")
	}
	return data, nil
}

func normalizeAuthCookie(value string) string {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "auth=") {
		return value
	}
	return "auth=" + value
}

func parseDiscoveredEmail(data []byte) string {
	var value any
	if json.Unmarshal(data, &value) == nil {
		if email := findEmail(value); email != "" {
			return email
		}
	}
	match := regexp.MustCompile(`[A-Za-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?)+`).Find(data)
	return string(match)
}

func findEmail(value any) string {
	switch value := value.(type) {
	case string:
		if strings.Contains(value, "@") {
			return value
		}
	case []any:
		for _, item := range value {
			if email := findEmail(item); email != "" {
				return email
			}
		}
	case map[string]any:
		for _, key := range []string{"email", "userEmail", "data", "value"} {
			if item, ok := value[key]; ok {
				if email := findEmail(item); email != "" {
					return email
				}
			}
		}
	}
	return ""
}

func parseDiscoveredAPIKeys(data []byte) []DiscoveredAPIKey {
	var value any
	if json.Unmarshal(data, &value) == nil {
		return collectDiscoveredAPIKeys(value)
	}
	return parseStreamedAPIKeys(string(data))
}

func collectDiscoveredAPIKeys(value any) []DiscoveredAPIKey {
	var objects []map[string]any
	collectObjects(value, &objects)
	seen := map[string]bool{}
	keys := make([]DiscoveredAPIKey, 0)
	for _, object := range objects {
		key, _ := object["key"].(string)
		key = strings.TrimSpace(key)
		if key == "" || strings.Contains(key, "...") || seen[key] {
			continue
		}
		seen[key] = true
		item := DiscoveredAPIKey{Key: key, ID: stringField(object, "id"), Name: stringField(object, "name"), Email: stringField(object, "email")}
		if item.Email == "" {
			item.Email = stringField(object, "userEmail")
		}
		if item.Name == "" {
			item.Name = "API Key"
		}
		item.Display = maskAPIKey(key)
		keys = append(keys, item)
	}
	return keys
}

func parseStreamedAPIKeys(text string) []DiscoveredAPIKey {
	objectRE := regexp.MustCompile(`\{[^{}]*\bkey:"(sk-[A-Za-z0-9]+)"[^{}]*\}`)
	field := func(object, name string) string {
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `:"((?:\\.|[^"\\])*)"`)
		match := re.FindStringSubmatch(object)
		if len(match) < 2 {
			return ""
		}
		value, err := strconv.Unquote(`"` + match[1] + `"`)
		if err != nil {
			return match[1]
		}
		return value
	}
	seen := map[string]bool{}
	keys := make([]DiscoveredAPIKey, 0)
	for _, match := range objectRE.FindAllStringSubmatch(text, -1) {
		key := match[1]
		if seen[key] {
			continue
		}
		seen[key] = true
		object := match[0]
		item := DiscoveredAPIKey{
			ID:      field(object, "id"),
			Name:    field(object, "name"),
			Email:   field(object, "email"),
			Key:     key,
			Display: field(object, "keyDisplay"),
		}
		if item.Name == "" {
			item.Name = "API Key"
		}
		if item.Display == "" {
			item.Display = maskAPIKey(key)
		}
		keys = append(keys, item)
	}
	return keys
}

func collectObjects(value any, objects *[]map[string]any) {
	switch value := value.(type) {
	case []any:
		for _, item := range value {
			collectObjects(item, objects)
		}
	case map[string]any:
		*objects = append(*objects, value)
		for _, item := range value {
			collectObjects(item, objects)
		}
	}
}

func stringField(object map[string]any, name string) string {
	value, _ := object[name].(string)
	return strings.TrimSpace(value)
}

func maskAPIKey(key string) string {
	if len(key) <= 10 {
		return strings.Repeat("•", len(key))
	}
	return key[:6] + "••••••" + key[len(key)-4:]
}

func probeAPIKey(apiKey, service string) (APIKeyProbeResult, error) {
	service = strings.ToLower(strings.TrimSpace(service))
	var path, payload string
	switch service {
	case "go":
		path = "/zen/go/v1/chat/completions"
	case "", "zen":
		service = "zen"
		path = "/zen/v1/chat/completions"
	default:
		return APIKeyProbeResult{}, errors.New("探测服务必须是 Go 或 Zen")
	}
	payload = `{"model":"deepseek-v4-flash","messages":[{"role":"user","content":"."}],"max_tokens":1}`
	req, _ := http.NewRequest(http.MethodPost, apiKeyProbeBase+path, strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+strings.TrimPrefix(strings.TrimSpace(apiKey), "Bearer "))
	req.Header.Set("Content-Type", "application/json")
	probeClient := &http.Client{Timeout: 45 * time.Second}
	resp, err := probeClient.Do(req)
	if err != nil {
		return APIKeyProbeResult{}, fmt.Errorf("API Key 探测失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	workspace := ""
	if match := regexp.MustCompile(`/workspace/(wrk_[A-Za-z0-9_-]+)(?:/|["'\\s]|$)`).FindSubmatch(body); len(match) == 2 {
		workspace = string(match[1])
	}
	var envelope struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &envelope)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return APIKeyProbeResult{WorkspaceID: workspace, Service: service, Enabled: true, Message: "探测请求成功，已确认服务可用"}, nil
	}
	if strings.EqualFold(envelope.Error.Type, "RegionError") {
		return APIKeyProbeResult{WorkspaceID: workspace, Service: service, Enabled: true, Message: envelope.Error.Message}, nil
	}
	if strings.EqualFold(envelope.Error.Type, "CreditsError") || workspace != "" {
		return APIKeyProbeResult{WorkspaceID: workspace, Service: service, Enabled: false, Message: envelope.Error.Message}, nil
	}
	if resp.StatusCode == http.StatusUnauthorized || strings.EqualFold(envelope.Error.Type, "AuthenticationError") || strings.EqualFold(envelope.Error.Type, "AuthError") {
		return APIKeyProbeResult{}, errors.New("API Key 无效或已失效")
	}
	return APIKeyProbeResult{}, fmt.Errorf("API Key 探测返回 HTTP %d: %s", resp.StatusCode, shortBody(body))
}

func probeAPIKeyEntitlements(apiKey string) (APIKeyEntitlements, error) {
	zen, zenErr := probeAPIKey(apiKey, "zen")
	goResult, goErr := probeAPIKey(apiKey, "go")
	if isInvalidAPIKeyProbeError(zenErr) || isInvalidAPIKeyProbeError(goErr) {
		return APIKeyEntitlements{}, errors.New("API Key 无效或已失效")
	}
	workspace := zen.WorkspaceID
	if workspace == "" {
		workspace = goResult.WorkspaceID
	}
	return APIKeyEntitlements{WorkspaceID: workspace, GoEnabled: goErr == nil && goResult.Enabled, ZenEnabled: zenErr == nil && zen.Enabled}, nil
}

func isInvalidAPIKeyProbeError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "API Key 无效或已失效")
}

func accountsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, publicAccounts())
	case http.MethodPost:
		var input struct {
			Name, WorkspaceID, APIKey, AuthCookie, ProxyURL string
			Priority                                        int
			APIOnly                                         bool
			ProbeService                                    string
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&input); err != nil {
			writeError(w, 400, "请求数据无效")
			return
		}
		input.Name = strings.TrimSpace(input.Name)
		input.WorkspaceID, input.APIKey, input.AuthCookie = strings.TrimSpace(input.WorkspaceID), strings.TrimSpace(input.APIKey), strings.TrimSpace(input.AuthCookie)
		input.ProxyURL = strings.TrimSpace(input.ProxyURL)
		if input.ProxyURL != "" {
			if err := validateProxy(input.ProxyURL); err != nil {
				writeError(w, 400, err.Error())
				return
			}
		}
		if input.APIOnly && input.APIKey == "" {
			writeError(w, 400, "API Key 添加模式必须填写 API Key")
			return
		}
		if !input.APIOnly && (input.WorkspaceID == "" || input.AuthCookie == "") {
			writeError(w, 400, "Workspace ID 和 auth Cookie 均为必填")
			return
		}
		if input.WorkspaceID != "" && !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(input.WorkspaceID) {
			writeError(w, 400, "Workspace ID 格式无效")
			return
		}
		store.RLock()
		for _, existing := range store.Accounts {
			if input.WorkspaceID != "" && existing.WorkspaceID == input.WorkspaceID {
				store.RUnlock()
				writeError(w, 409, "该 Workspace 已在监控中，无需重复添加")
				return
			}
		}
		store.RUnlock()
		if !input.APIOnly && (input.Name == "" || input.APIKey == "") {
			if discovered, discoverErr := discoverAccount(input.WorkspaceID, input.AuthCookie); discoverErr == nil {
				if input.Name == "" {
					input.Name = discovered.Email
				}
				if input.APIKey == "" && len(discovered.APIKeys) == 1 {
					input.APIKey = discovered.APIKeys[0].Key
				}
			}
		}
		entitlements := APIKeyEntitlements{}
		if input.APIOnly {
			var err error
			entitlements, err = probeAPIKeyEntitlements(input.APIKey)
			if err != nil {
				writeError(w, http.StatusBadGateway, err.Error())
				return
			}
			if input.WorkspaceID == "" {
				input.WorkspaceID = entitlements.WorkspaceID
			}
			if input.WorkspaceID != "" {
				store.RLock()
				for _, existing := range store.Accounts {
					if existing.WorkspaceID == input.WorkspaceID {
						store.RUnlock()
						writeError(w, http.StatusConflict, "探测到的 Workspace 已在账号池中")
						return
					}
				}
				store.RUnlock()
			}
		}
		if input.Name == "" && input.APIOnly {
			if input.WorkspaceID != "" {
				input.Name = input.WorkspaceID
			} else {
				input.Name = "API Key " + maskAPIKey(input.APIKey)
			}
		}
		if input.Name == "" {
			input.Name = input.WorkspaceID
		}
		a := Account{ID: "acc-" + strconv.FormatInt(time.Now().UnixNano(), 36), Name: input.Name, Type: "workspace", WorkspaceID: input.WorkspaceID, APIKey: strings.TrimPrefix(input.APIKey, "Bearer "), AuthCookie: input.AuthCookie, Status: "checking", Priority: input.Priority, ProxyURL: input.ProxyURL, APIOnly: input.APIOnly, ProbeService: "all"}
		if input.APIOnly {
			a.ZenAvailable = true
			a.ZenBillingEnabled = entitlements.ZenEnabled
			a.GoAvailable = entitlements.GoEnabled
		}
		refreshAccount(&a)
		store.Lock()
		store.Accounts = append(store.Accounts, a)
		err := store.saveLocked()
		store.Unlock()
		if err != nil {
			writeError(w, 500, "保存账号失败: "+err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, toPublic(a))
	default:
		writeError(w, 405, "method not allowed")
	}
}

func accountHandler(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/accounts/")
	if strings.HasSuffix(rest, "/models") {
		id := strings.TrimSuffix(rest, "/models")
		if r.Method != http.MethodGet {
			writeError(w, 405, "method not allowed")
			return
		}
		store.RLock()
		var account *Account
		for i := range store.Accounts {
			if store.Accounts[i].ID == id {
				copy := store.Accounts[i]
				account = &copy
				break
			}
		}
		store.RUnlock()
		if account == nil {
			writeError(w, 404, "账号不存在")
			return
		}
		goModels, goErr := fetchModels(goModelsURL, account.APIKey)
		zenModels, zenErr := fetchModels(zenModelsURL, account.APIKey)
		annotateZenDeprecations(zenModels)
		annotateModelAvailability(goModels, func(string) (bool, string) {
			if account.GoAvailable {
				return true, ""
			}
			return false, "当前凭证未开通 Go"
		})
		annotateModelAvailability(zenModels, func(id string) (bool, string) {
			if account.ZenBillingEnabled || isZenFreeModel(id) {
				return true, ""
			}
			return false, "Free 账号仅可使用 Zen 免费模型"
		})
		if goErr != nil && zenErr != nil {
			writeError(w, 502, "Go 模型查询失败："+goErr.Error()+"；Zen 模型查询失败："+zenErr.Error())
			return
		}
		result := map[string]any{
			"authenticated": account.APIKey != "",
			"workspaceId":   account.WorkspaceID,
			"go":            map[string]any{"models": goModels, "error": errorText(goErr)},
			"zen":           map[string]any{"models": zenModels, "error": errorText(zenErr)},
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if strings.HasSuffix(rest, "/refresh") {
		id := strings.TrimSuffix(rest, "/refresh")
		if r.Method != http.MethodPost {
			writeError(w, 405, "method not allowed")
			return
		}
		store.Lock()
		for i := range store.Accounts {
			if store.Accounts[i].ID != id {
				continue
			}
			refreshAccount(&store.Accounts[i])
			_ = store.saveLocked()
			result := toPublic(store.Accounts[i])
			store.Unlock()
			writeJSON(w, http.StatusOK, result)
			return
		}
		store.Unlock()
		writeError(w, 404, "账号不存在")
		return
	}
	id := rest
	if id == "" {
		writeError(w, 404, "账号不存在")
		return
	}
	switch r.Method {
	case http.MethodDelete:
		store.Lock()
		defer store.Unlock()
		for i := range store.Accounts {
			if store.Accounts[i].ID != id {
				continue
			}
			store.Accounts = append(store.Accounts[:i], store.Accounts[i+1:]...)
			if err := store.saveLocked(); err != nil {
				writeError(w, 500, err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, 404, "账号不存在")
	case http.MethodPut:
		var input struct {
			Name, WorkspaceID, APIKey, AuthCookie, ProxyURL string
			Priority                                        int
			APIOnly                                         bool
			ProbeService                                    string
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&input); err != nil {
			writeError(w, 400, "请求数据无效")
			return
		}
		input.Name, input.WorkspaceID = strings.TrimSpace(input.Name), strings.TrimSpace(input.WorkspaceID)
		input.APIKey, input.AuthCookie = strings.TrimSpace(input.APIKey), strings.TrimSpace(input.AuthCookie)
		input.ProxyURL = strings.TrimSpace(input.ProxyURL)
		if input.ProxyURL != "" {
			if err := validateProxy(input.ProxyURL); err != nil {
				writeError(w, 400, err.Error())
				return
			}
		}
		if !input.APIOnly && input.WorkspaceID == "" {
			writeError(w, 400, "Workspace ID 为必填")
			return
		}
		if input.WorkspaceID != "" && !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(input.WorkspaceID) {
			writeError(w, 400, "Workspace ID 格式无效")
			return
		}
		store.Lock()
		index := -1
		for i := range store.Accounts {
			if input.WorkspaceID != "" && store.Accounts[i].ID != id && store.Accounts[i].WorkspaceID == input.WorkspaceID {
				store.Unlock()
				writeError(w, 409, "该 Workspace 已在监控中")
				return
			}
			if store.Accounts[i].ID == id {
				index = i
			}
		}
		if index < 0 {
			store.Unlock()
			writeError(w, 404, "账号不存在")
			return
		}
		a := &store.Accounts[index]
		if input.Name != "" {
			a.Name = input.Name
		}
		if input.WorkspaceID != "" {
			a.WorkspaceID = input.WorkspaceID
		}
		a.Priority, a.ProxyURL = input.Priority, input.ProxyURL
		a.APIOnly = input.APIOnly || a.APIOnly
		if input.ProbeService != "" {
			a.ProbeService = input.ProbeService
		}
		if input.APIKey != "" {
			a.APIKey = strings.TrimPrefix(input.APIKey, "Bearer ")
		}
		if input.AuthCookie != "" {
			a.AuthCookie = input.AuthCookie
		}
		refreshAccount(a)
		err := store.saveLocked()
		result := toPublic(*a)
		store.Unlock()
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, result)
	default:
		writeError(w, 405, "method not allowed")
	}
}

func refreshHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "method not allowed")
		return
	}
	refreshAll()
	writeJSON(w, http.StatusOK, publicAccounts())
}

func refreshAll() {
	store.Lock()
	var wg sync.WaitGroup
	for i := range store.Accounts {
		wg.Add(1)
		go func(a *Account) { defer wg.Done(); refreshAccount(a) }(&store.Accounts[i])
	}
	wg.Wait()
	_ = store.saveLocked()
	store.Unlock()
}

func refreshAccount(a *Account) {
	a.LastChecked = time.Now()
	a.Error, a.GoError, a.ZenError = "", "", ""
	a.Status = "checking"
	a.Type = "workspace"
	goModels, goModelErr := fetchModels(goModelsURL, a.APIKey)
	zenModels, zenModelErr := fetchModels(zenModelsURL, a.APIKey)
	if goModelErr == nil {
		a.GoModelCount = len(goModels)
	}
	if zenModelErr == nil {
		a.ZenModelCount = len(zenModels)
	}
	a.ModelCount = a.GoModelCount + a.ZenModelCount
	if a.APIOnly {
		a.Error, a.GoError, a.ZenError = "", "", ""
		classifyAccount(a, 0)
		return
	}
	rolling, weekly, monthly, goErr := fetchDashboardUsage(a.WorkspaceID, a.AuthCookie)
	a.GoAvailable = goErr == nil
	if goErr == nil {
		a.Rolling, a.Weekly, a.Monthly = rolling, weekly, monthly
	} else {
		a.Rolling, a.Weekly, a.Monthly = nil, nil, nil
		a.GoError = goErr.Error()
	}
	balance, zenBillingEnabled, zenErr := fetchZenBalance(a.WorkspaceID, a.AuthCookie)
	a.ZenAvailable = zenErr == nil
	a.ZenBillingEnabled = zenErr == nil && zenBillingEnabled
	if zenErr == nil {
		a.ZenBalance = balance
	} else {
		a.ZenError = zenErr.Error()
	}
	if !a.GoAvailable && !a.ZenAvailable {
		a.Status = "error"
		a.Error = "Go 与 Zen 均查询失败，请检查 Workspace ID 或 auth Cookie"
		return
	}
	maxUsage := 0.0
	for _, item := range []*UsageWindow{rolling, weekly, monthly} {
		if item != nil && item.UsagePercent > maxUsage {
			maxUsage = item.UsagePercent
		}
	}
	classifyAccount(a, maxUsage)
}

func classifyAccount(a *Account, maxUsage float64) {
	switch {
	case a.GoAvailable && a.ZenBillingEnabled:
		a.Type = "go_zen"
	case a.GoAvailable:
		a.Type = "go"
	case a.ZenBillingEnabled:
		a.Type = "zen"
	default:
		a.Type = "free"
	}
	if a.GoAvailable && maxUsage >= 90 {
		a.Status = "critical"
	} else if a.GoAvailable && maxUsage >= 80 {
		a.Status = "warning"
	} else if a.ZenBillingEnabled && a.ZenBalance <= 1 {
		a.Status = "critical"
	} else if a.ZenBillingEnabled && a.ZenBalance <= 5 {
		a.Status = "warning"
	} else {
		a.Status = "healthy"
	}
}

func fetchZenBalance(workspaceID, authCookie string) (float64, bool, error) {
	url := "https://opencode.ai/workspace/" + workspaceID
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	cookie := authCookie
	if !strings.Contains(cookie, "=") {
		cookie = "auth=" + cookie
	}
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return 0, false, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return 0, false, fmt.Errorf("HTTP %d: %s", resp.StatusCode, shortBody(body))
	}
	if balance, ok := parseZenBalance(normalizeDashboard(string(body))); ok {
		normalized := normalizeDashboard(string(body))
		enabled, found := parseZenBillingEnabled(normalized)
		if !found {
			enabled = balance > 0
		}
		return balance, enabled, nil
	}
	return 0, false, errors.New("页面中未找到 Current balance，Cookie 可能已过期或官方页面结构已变化")
}

func parseZenBillingEnabled(text string) (bool, bool) {
	m := regexp.MustCompile(`(?i)["']?useBalance["']?\s*:\s*(true|false|!0|!1)`).FindStringSubmatch(text)
	if len(m) < 2 {
		return false, false
	}
	return m[1] == "true" || m[1] == "!0", true
}

func parseZenBalance(text string) (float64, bool) {
	renderedPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?is)data-slot=["']balance["'].{0,500}?\$\s*([0-9][0-9,]*(?:\.[0-9]+)?)`),
		regexp.MustCompile(`(?is)(?:Current\s+balance|当前余额).{0,500}?\$\s*([0-9][0-9,]*(?:\.[0-9]+)?)`),
	}
	for _, re := range renderedPatterns {
		m := re.FindStringSubmatch(text)
		if len(m) < 2 {
			continue
		}
		v, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64)
		if err == nil {
			return v, true
		}
	}
	// billing.get serializes BillingTable.balance in microcents.
	raw := regexp.MustCompile(`(?is)["']?balance["']?\s*:\s*["']?([0-9]+)["']?`).FindStringSubmatch(text)
	if len(raw) >= 2 {
		v, err := strconv.ParseFloat(raw[1], 64)
		if err == nil {
			return v / 100000000, true
		}
	}
	return 0, false
}

func fetchModels(url, apiKey string) ([]ModelInfo, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, shortBody(body))
	}
	var wrapped struct {
		Data   []json.RawMessage `json:"data"`
		Models []json.RawMessage `json:"models"`
	}
	if json.Unmarshal(body, &wrapped) == nil {
		if wrapped.Data != nil {
			return parseModels(wrapped.Data), nil
		}
		if wrapped.Models != nil {
			return parseModels(wrapped.Models), nil
		}
	}
	var list []json.RawMessage
	if json.Unmarshal(body, &list) == nil {
		return parseModels(list), nil
	}
	return nil, errors.New("模型接口返回格式无法识别")
}

func parseModels(raw []json.RawMessage) []ModelInfo {
	models := make([]ModelInfo, 0, len(raw))
	for _, item := range raw {
		var value map[string]any
		if json.Unmarshal(item, &value) != nil {
			continue
		}
		id := firstString(value, "id", "model", "slug")
		name := firstString(value, "name", "display_name", "displayName")
		provider := firstString(value, "provider", "owned_by", "ownedBy")
		if id == "" {
			id = name
		}
		if name == "" {
			name = id
		}
		if id != "" {
			models = append(models, ModelInfo{ID: id, Name: name, Provider: provider, Available: true})
		}
	}
	return models
}

func firstString(value map[string]any, keys ...string) string {
	for _, key := range keys {
		if text, ok := value[key].(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func annotateZenDeprecations(models []ModelInfo) {
	for i := range models {
		keys := []string{strings.ToLower(models[i].ID), strings.ToLower(models[i].Name)}
		for _, key := range keys {
			if date, ok := zenDeprecationDates[key]; ok {
				models[i].Deprecated = true
				models[i].DeprecationDate = date
				break
			}
		}
	}
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func fetchDashboardUsage(workspaceID, authCookie string) (*UsageWindow, *UsageWindow, *UsageWindow, error) {
	url := "https://opencode.ai/workspace/" + workspaceID + "/go"
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	cookie := authCookie
	if !strings.Contains(cookie, "=") {
		cookie = "auth=" + cookie
	}
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, nil, nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, shortBody(body))
	}
	text := normalizeDashboard(string(body))
	now := time.Now()
	rolling := parseUsageWindow("rollingUsage", text, 12, now)
	weekly := parseUsageWindow("weeklyUsage", text, 30, now)
	monthly := parseUsageWindow("monthlyUsage", text, 60, now)
	if rolling == nil && weekly == nil && monthly == nil {
		return nil, nil, nil, errors.New("页面中未找到额度数据，Cookie 可能已过期或官方页面结构已变化")
	}
	return rolling, weekly, monthly, nil
}

func normalizeDashboard(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\u0022`, `"`)
	return s
}

func parseUsageWindow(name, text string, limit float64, now time.Time) *UsageWindow {
	objectRE := regexp.MustCompile(`(?s)["']?` + regexp.QuoteMeta(name) + `["']?\s*:\s*(?:\$R\[\d+\]\s*=\s*)?\{([^{}]*)\}`)
	m := objectRE.FindStringSubmatch(text)
	if len(m) < 2 {
		return nil
	}
	usage, ok1 := captureNumber("usagePercent", m[1])
	reset, ok2 := captureNumber("resetInSec", m[1])
	if !ok1 || !ok2 {
		return nil
	}
	seconds := int(reset + .5)
	if seconds < 0 {
		seconds = 0
	}
	return &UsageWindow{UsagePercent: usage, ResetInSecond: seconds, ResetAt: now.Add(time.Duration(seconds) * time.Second), LimitUSD: limit}
}

func captureNumber(name, text string) (float64, bool) {
	re := regexp.MustCompile(`["']?` + regexp.QuoteMeta(name) + `["']?\s*:\s*["']?(-?\d+(?:\.\d+)?)["']?`)
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return 0, false
	}
	v, err := strconv.ParseFloat(m[1], 64)
	return v, err == nil
}

func publicAccounts() []PublicAccount {
	store.RLock()
	defer store.RUnlock()
	out := make([]PublicAccount, len(store.Accounts))
	for i, a := range store.Accounts {
		out[i] = toPublic(a)
	}
	return out
}
func toPublic(a Account) PublicAccount {
	return PublicAccount{
		ID: a.ID, Name: a.Name, Type: a.Type, WorkspaceID: a.WorkspaceID,
		Status: a.Status, Error: a.Error, ModelCount: a.ModelCount,
		GoModelCount: a.GoModelCount, ZenModelCount: a.ZenModelCount,
		Rolling: a.Rolling, Weekly: a.Weekly, Monthly: a.Monthly,
		ZenBalance: a.ZenBalance, GoAvailable: a.GoAvailable, ZenAvailable: a.ZenAvailable, ZenBillingEnabled: a.ZenBillingEnabled,
		GoError: a.GoError, ZenError: a.ZenError,
		LastChecked: a.LastChecked, HasAPIKey: a.APIKey != "", HasCookie: a.AuthCookie != "",
		Priority: a.Priority, ProxyURL: a.ProxyURL, APIOnly: a.APIOnly, ProbeService: a.ProbeService,
	}
}
func shortBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 180 {
		s = s[:180]
	}
	return s
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func refreshScheduler() {
	ticker := time.NewTicker(refreshEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			refreshAll()
		case <-shutdownSignal:
			return
		}
	}
}
func printConsoleHelp() {
	line := strings.Repeat("=", 52)
	log.Println(line)
	log.Println("OpenCode Pool Gateway 真实额度监控已启动")
	log.Println("访问地址: http://localhost:8787")
	log.Println("命令: [O] 打开网页  [R] 刷新额度  [Q] 安全退出  [H] 帮助")
	log.Printf("账号配置: %s", store.path)
	log.Printf("登录配置: %s", authDataPath())
	log.Println(line)
}
func readConsoleCommands() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		switch strings.ToLower(strings.TrimSpace(scanner.Text())) {
		case "q", "quit", "exit":
			requestShutdown()
			return
		case "o", "open":
			if err := openBrowser("http://localhost:8787"); err != nil {
				log.Printf("无法自动打开浏览器: %v", err)
			}
		case "r", "refresh":
			log.Println("正在刷新真实额度...")
			refreshAll()
			log.Println("刷新完成")
		case "h", "help", "?":
			printConsoleHelp()
		default:
			log.Println("未知命令，请输入 O、R、Q 或 H")
		}
	}
}

func openBrowser(url string) error {
	var command string
	var args []string
	switch runtime.GOOS {
	case "windows":
		command, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		command, args = "open", []string{url}
	default:
		command, args = "xdg-open", []string{url}
	}
	return exec.Command(command, args...).Start()
}
func requestShutdown() { shutdownOnce.Do(func() { close(shutdownSignal) }) }
func shutdownHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, 405, "method not allowed")
		return
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || (host != "127.0.0.1" && host != "::1") {
		writeError(w, 403, "local requests only")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "shutting_down"})
	go func() { time.Sleep(150 * time.Millisecond); requestShutdown() }()
}
