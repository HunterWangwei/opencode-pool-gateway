package main

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
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

//go:embed web/*
var webFS embed.FS

const (
	listenAddr   = ":8787"
	goModelsURL  = "https://opencode.ai/zen/go/v1/models"
	zenModelsURL = "https://opencode.ai/zen/v1/models"
	refreshEvery = 5 * time.Minute
)

type UsageWindow struct {
	UsagePercent  float64   `json:"usagePercent"`
	ResetInSecond int       `json:"resetInSec"`
	ResetAt       time.Time `json:"resetAt"`
	LimitUSD      float64   `json:"limitUsd"`
}

type ModelInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Provider        string `json:"provider,omitempty"`
	Deprecated      bool   `json:"deprecated,omitempty"`
	DeprecationDate string `json:"deprecationDate,omitempty"`
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
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Type          string       `json:"type"`
	WorkspaceID   string       `json:"workspaceId"`
	APIKey        string       `json:"apiKey"`
	AuthCookie    string       `json:"authCookie"`
	Status        string       `json:"status"`
	Error         string       `json:"error,omitempty"`
	ModelCount    int          `json:"modelCount"`
	GoModelCount  int          `json:"goModelCount"`
	ZenModelCount int          `json:"zenModelCount"`
	Rolling       *UsageWindow `json:"rolling,omitempty"`
	Weekly        *UsageWindow `json:"weekly,omitempty"`
	Monthly       *UsageWindow `json:"monthly,omitempty"`
	ZenBalance    float64      `json:"zenBalance,omitempty"`
	GoAvailable   bool         `json:"goAvailable"`
	ZenAvailable  bool         `json:"zenAvailable"`
	GoError       string       `json:"goError,omitempty"`
	ZenError      string       `json:"zenError,omitempty"`
	LastChecked   time.Time    `json:"lastChecked,omitempty"`
}

type PublicAccount struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Type          string       `json:"type"`
	WorkspaceID   string       `json:"workspaceId"`
	Status        string       `json:"status"`
	Error         string       `json:"error,omitempty"`
	ModelCount    int          `json:"modelCount"`
	GoModelCount  int          `json:"goModelCount"`
	ZenModelCount int          `json:"zenModelCount"`
	Rolling       *UsageWindow `json:"rolling,omitempty"`
	Weekly        *UsageWindow `json:"weekly,omitempty"`
	Monthly       *UsageWindow `json:"monthly,omitempty"`
	ZenBalance    float64      `json:"zenBalance,omitempty"`
	GoAvailable   bool         `json:"goAvailable"`
	ZenAvailable  bool         `json:"zenAvailable"`
	GoError       string       `json:"goError,omitempty"`
	ZenError      string       `json:"zenError,omitempty"`
	LastChecked   time.Time    `json:"lastChecked,omitempty"`
	HasAPIKey     bool         `json:"hasApiKey"`
	HasCookie     bool         `json:"hasCookie"`
}

type accountStore struct {
	sync.RWMutex
	Accounts []Account `json:"accounts"`
	path     string
}

var store accountStore
var client = &http.Client{Timeout: 18 * time.Second}
var shutdownOnce sync.Once
var shutdownSignal = make(chan struct{})
var version = "dev"
var commit = "unknown"
var buildDate = "unknown"

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-version") {
		fmt.Printf("GoQuota %s (%s, %s)\n", version, commit, buildDate)
		return
	}
	webRoot, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}
	store.path = dataPath()
	if err := store.load(); err != nil {
		log.Printf("读取账号配置失败: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/accounts", accountsHandler)
	mux.HandleFunc("/api/accounts/", accountHandler)
	mux.HandleFunc("/api/refresh", refreshHandler)
	mux.HandleFunc("/api/version", versionHandler)
	mux.HandleFunc("/api/shutdown", shutdownHandler)
	mux.Handle("/", http.FileServer(http.FS(webRoot)))

	server := &http.Server{Addr: listenAddr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
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
	log.Println("GoQuota 已安全退出")
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

func legacyDataPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "GoQuota", "accounts.json")
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
	valid := make([]Account, 0, len(s.Accounts))
	seen := make(map[string]int)
	for _, account := range s.Accounts {
		if account.WorkspaceID == "" || account.AuthCookie == "" {
			continue
		}
		account.Type = "workspace"
		if index, exists := seen[account.WorkspaceID]; exists {
			if valid[index].APIKey == "" {
				valid[index].APIKey = account.APIKey
			}
			continue
		}
		seen[account.WorkspaceID] = len(valid)
		valid = append(valid, account)
	}
	if len(valid) != len(s.Accounts) {
		log.Printf("已移除 %d 个旧版演示账号", len(s.Accounts)-len(valid))
	}
	s.Accounts = valid
	return s.saveLocked()
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

func accountsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, publicAccounts())
	case http.MethodPost:
		var input struct{ Name, WorkspaceID, APIKey, AuthCookie string }
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&input); err != nil {
			writeError(w, 400, "请求数据无效")
			return
		}
		input.Name = strings.TrimSpace(input.Name)
		input.WorkspaceID, input.APIKey, input.AuthCookie = strings.TrimSpace(input.WorkspaceID), strings.TrimSpace(input.APIKey), strings.TrimSpace(input.AuthCookie)
		if input.Name == "" {
			writeError(w, 400, "名称不能为空")
			return
		}
		if input.WorkspaceID == "" || input.AuthCookie == "" {
			writeError(w, 400, "名称、Workspace ID 和 auth Cookie 均为必填")
			return
		}
		if !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(input.WorkspaceID) {
			writeError(w, 400, "Workspace ID 格式无效")
			return
		}
		store.RLock()
		for _, existing := range store.Accounts {
			if existing.WorkspaceID == input.WorkspaceID {
				store.RUnlock()
				writeError(w, 409, "该 Workspace 已在监控中，无需重复添加")
				return
			}
		}
		store.RUnlock()
		a := Account{ID: "acc-" + strconv.FormatInt(time.Now().UnixNano(), 36), Name: input.Name, Type: "workspace", WorkspaceID: input.WorkspaceID, APIKey: strings.TrimPrefix(input.APIKey, "Bearer "), AuthCookie: input.AuthCookie, Status: "checking"}
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
		var input struct{ Name, WorkspaceID, APIKey, AuthCookie string }
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&input); err != nil {
			writeError(w, 400, "请求数据无效")
			return
		}
		input.Name, input.WorkspaceID = strings.TrimSpace(input.Name), strings.TrimSpace(input.WorkspaceID)
		input.APIKey, input.AuthCookie = strings.TrimSpace(input.APIKey), strings.TrimSpace(input.AuthCookie)
		if input.Name == "" || input.WorkspaceID == "" {
			writeError(w, 400, "名称和 Workspace ID 均为必填")
			return
		}
		if !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(input.WorkspaceID) {
			writeError(w, 400, "Workspace ID 格式无效")
			return
		}
		store.Lock()
		index := -1
		for i := range store.Accounts {
			if store.Accounts[i].ID != id && store.Accounts[i].WorkspaceID == input.WorkspaceID {
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
		a.Name, a.WorkspaceID = input.Name, input.WorkspaceID
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
	rolling, weekly, monthly, goErr := fetchDashboardUsage(a.WorkspaceID, a.AuthCookie)
	a.GoAvailable = goErr == nil
	if goErr == nil {
		a.Rolling, a.Weekly, a.Monthly = rolling, weekly, monthly
	} else {
		a.Rolling, a.Weekly, a.Monthly = nil, nil, nil
		a.GoError = goErr.Error()
	}
	balance, zenErr := fetchZenBalance(a.WorkspaceID, a.AuthCookie)
	a.ZenAvailable = zenErr == nil
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
	if a.GoAvailable && maxUsage >= 90 {
		a.Status = "critical"
	} else if a.GoAvailable && maxUsage >= 80 {
		a.Status = "warning"
	} else if !a.GoAvailable && a.ZenAvailable && balance <= 1 {
		a.Status = "critical"
	} else if !a.GoAvailable && a.ZenAvailable && balance <= 5 {
		a.Status = "warning"
	} else {
		a.Status = "healthy"
	}
}

func fetchZenBalance(workspaceID, authCookie string) (float64, error) {
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
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, shortBody(body))
	}
	if balance, ok := parseZenBalance(normalizeDashboard(string(body))); ok {
		return balance, nil
	}
	return 0, errors.New("页面中未找到 Current balance，Cookie 可能已过期或官方页面结构已变化")
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
			models = append(models, ModelInfo{ID: id, Name: name, Provider: provider})
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
		ZenBalance: a.ZenBalance, GoAvailable: a.GoAvailable, ZenAvailable: a.ZenAvailable,
		GoError: a.GoError, ZenError: a.ZenError,
		LastChecked: a.LastChecked, HasAPIKey: a.APIKey != "", HasCookie: a.AuthCookie != "",
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
	log.Println("GoQuota 真实额度监控已启动")
	log.Println("访问地址: http://localhost:8787")
	log.Println("命令: [O] 打开网页  [R] 刷新额度  [Q] 安全退出  [H] 帮助")
	log.Printf("账号配置: %s", store.path)
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
