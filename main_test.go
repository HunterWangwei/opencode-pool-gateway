package main

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuthenticationFlow(t *testing.T) {
	auth := newAuthManager("admin", "correct-horse-battery")
	protected := auth.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))

	unauthorized := httptest.NewRecorder()
	protected.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/accounts", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauthorized.Code)
	}

	form := url.Values{"username": {"admin"}, "password": {"correct-horse-battery"}, "next": {"/#accounts"}}
	loginRequest := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	loginRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRequest.RemoteAddr = "127.0.0.1:50000"
	loginResponse := httptest.NewRecorder()
	auth.loginHandler(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusSeeOther {
		t.Fatalf("expected login redirect, got %d", loginResponse.Code)
	}
	cookies := loginResponse.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("unexpected session cookie: %+v", cookies)
	}

	authorizedRequest := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	authorizedRequest.AddCookie(cookies[0])
	authorized := httptest.NewRecorder()
	protected.ServeHTTP(authorized, authorizedRequest)
	if authorized.Code != http.StatusNoContent {
		t.Fatalf("expected authenticated request, got %d", authorized.Code)
	}
}

func TestSafeNext(t *testing.T) {
	for _, unsafe := range []string{"", "https://example.com", "//example.com"} {
		if got := safeNext(unsafe); got != "/" {
			t.Fatalf("unsafe redirect %q became %q", unsafe, got)
		}
	}
	if got := safeNext("/#accounts"); got != "/#accounts" {
		t.Fatalf("valid local redirect changed to %q", got)
	}
}

func TestCredentialMinimumLength(t *testing.T) {
	if err := validateCredentials("admin", "12345678"); err != nil {
		t.Fatalf("8-character password should be accepted: %v", err)
	}
	if err := validateCredentials("admin", "1234567"); err == nil {
		t.Fatal("7-character password should be rejected")
	}
}

func TestAuthConfigLoadAndHotUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "auth.json")
	initialHash := hashPassword("initial-password-123")
	if err := saveAuthConfig(path, authConfig{Username: "configured-admin", PasswordHash: initialHash, UpdatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	auth, generated, err := loadAuthManager(path)
	if err != nil || generated != "" || auth.username != "configured-admin" || !verifyPassword("initial-password-123", auth.passwordHash) {
		t.Fatalf("config was not loaded: generated=%q err=%v", generated, err)
	}
	auth.sessions["existing-session"] = time.Now().Add(time.Hour)

	payload := `{"currentPassword":"initial-password-123","username":"new-admin","newPassword":"updated-password-456"}`
	request := httptest.NewRequest(http.MethodPut, "/api/auth", strings.NewReader(payload))
	response := httptest.NewRecorder()
	auth.credentialsHandler(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("credential update failed: %d %s", response.Code, response.Body.String())
	}
	if auth.username != "new-admin" || !verifyPassword("updated-password-456", auth.passwordHash) || verifyPassword("initial-password-123", auth.passwordHash) {
		t.Fatal("credentials were not hot-updated")
	}
	if len(auth.sessions) != 0 {
		t.Fatal("existing sessions were not invalidated")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if strings.Contains(text, "initial-password-123") || strings.Contains(text, "updated-password-456") || !strings.Contains(text, "pbkdf2-sha256$") {
		t.Fatalf("auth config did not store a safe hash: %s", text)
	}
}

func TestParseModels(t *testing.T) {
	raw := []json.RawMessage{json.RawMessage(`{"id":"model-a","name":"Model A","provider":"OpenCode"}`), json.RawMessage(`{"model":"model-b","owned_by":"Vendor B"}`)}
	models := parseModels(raw)
	if len(models) != 2 || models[0].Name != "Model A" || models[1].ID != "model-b" || models[1].Provider != "Vendor B" {
		t.Fatalf("unexpected models: %+v", models)
	}
}

func TestAnnotateZenDeprecations(t *testing.T) {
	models := []ModelInfo{{ID: "gpt-5.2-codex", Name: "GPT 5.2 Codex"}, {ID: "claude-opus-5", Name: "Claude Opus 5"}}
	annotateZenDeprecations(models)
	if !models[0].Deprecated || models[0].DeprecationDate != "2026-07-23" {
		t.Fatalf("deprecated model was not annotated: %+v", models[0])
	}
	if models[1].Deprecated {
		t.Fatalf("active model was marked deprecated: %+v", models[1])
	}
}

func TestParseDashboardUsageJSON(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	page := `<script>self.__next_f.push([1,"{\"rollingUsage\":{\"usagePercent\":12.5,\"resetInSec\":3600},\"weeklyUsage\":{\"usagePercent\":\"25\",\"resetInSec\":\"7200\"},\"monthlyUsage\":{\"usagePercent\":50,\"resetInSec\":10800}}"])</script>`
	text := normalizeDashboard(page)
	tests := []struct {
		name  string
		want  float64
		reset int
		limit float64
	}{{"rollingUsage", 12.5, 3600, 12}, {"weeklyUsage", 25, 7200, 30}, {"monthlyUsage", 50, 10800, 60}}
	for _, tt := range tests {
		got := parseUsageWindow(tt.name, text, tt.limit, now)
		if got == nil {
			t.Fatalf("%s was not parsed", tt.name)
		}
		if math.Abs(got.UsagePercent-tt.want) > 0.001 || got.ResetInSecond != tt.reset || got.LimitUSD != tt.limit {
			t.Fatalf("%s unexpected value: %+v", tt.name, got)
		}
		if !got.ResetAt.Equal(now.Add(time.Duration(tt.reset) * time.Second)) {
			t.Fatalf("%s reset time incorrect", tt.name)
		}
	}
}

func TestParseDashboardUsageReactAssignment(t *testing.T) {
	page := `$R[24]($R[18],$R[30]={mine:!0,useBalance:!0,rollingUsage:$R[31]={status:"ok",resetInSec:18000,usagePercent:0},weeklyUsage:$R[32]={status:"ok",resetInSec:162822,usagePercent:31},monthlyUsage:$R[33]={status:"ok",resetInSec:1404782,usagePercent:21}});`
	weekly := parseUsageWindow("weeklyUsage", normalizeDashboard(page), 30, time.Now())
	if weekly == nil || weekly.UsagePercent != 31 || weekly.ResetInSecond != 162822 {
		t.Fatalf("unexpected weekly usage: %+v", weekly)
	}
}

func TestParseDashboardUsageMissing(t *testing.T) {
	if got := parseUsageWindow("rollingUsage", `<html>login required</html>`, 12, time.Now()); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestParseZenBalanceHTML(t *testing.T) {
	page := `<section><h2>Current balance</h2><div class="amount">$18.42</div></section>`
	got, ok := parseZenBalance(page)
	if !ok || math.Abs(got-18.42) > 0.001 {
		t.Fatalf("unexpected balance %.2f, ok=%v", got, ok)
	}
}

func TestParseZenBalanceSerializedData(t *testing.T) {
	page := `self.__next_f.push([1,"{\"workspace\":{\"balance\":725000000}}"]);`
	got, ok := parseZenBalance(normalizeDashboard(page))
	if !ok || math.Abs(got-7.25) > 0.001 {
		t.Fatalf("unexpected balance %.2f, ok=%v", got, ok)
	}
}

func TestParseZenBalanceChineseHTML(t *testing.T) {
	page := `<span data-slot="balance">当前余额 <b>$1,234.56</b></span>`
	got, ok := parseZenBalance(page)
	if !ok || math.Abs(got-1234.56) > 0.001 {
		t.Fatalf("unexpected balance %.2f, ok=%v", got, ok)
	}
}

func TestParseZenBalanceMissing(t *testing.T) {
	if _, ok := parseZenBalance("login required"); ok {
		t.Fatal("expected missing balance")
	}
}

func TestParseZenBillingEnabled(t *testing.T) {
	for _, tc := range []struct {
		text    string
		enabled bool
	}{
		{`{"useBalance":true}`, true},
		{`useBalance:!0`, true},
		{`{"useBalance":false}`, false},
		{`useBalance:!1`, false},
	} {
		got, found := parseZenBillingEnabled(tc.text)
		if !found || got != tc.enabled {
			t.Fatalf("unexpected billing state for %q: %v %v", tc.text, got, found)
		}
	}
	if _, found := parseZenBillingEnabled(`{"balance":0}`); found {
		t.Fatal("missing useBalance should remain unknown")
	}
}

func TestFreeAccountIsHealthy(t *testing.T) {
	a := Account{ZenAvailable: true, ZenBillingEnabled: false, ZenBalance: 0}
	classifyAccount(&a, 0)
	if a.Type != "free" || a.Status != "healthy" {
		t.Fatalf("free account was misclassified: %+v", a)
	}
	a = Account{ZenAvailable: true, ZenBillingEnabled: true, ZenBalance: 0}
	classifyAccount(&a, 0)
	if a.Type != "zen" || a.Status != "critical" {
		t.Fatalf("enabled zero-balance Zen account was misclassified: %+v", a)
	}
}

func TestAccountStoreLoadPreservesAPIOnlyAndIncompleteAccounts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accounts.json")
	want := []Account{
		{ID: "api-only", Name: "API Key account", APIKey: "sk-test", APIOnly: true},
		{ID: "workspace", Name: "Workspace account", WorkspaceID: "wrk_test", AuthCookie: "cookie"},
		{ID: "incomplete", Name: "Keep my account"},
	}
	body, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	s := accountStore{path: path}
	if err := s.load(); err != nil {
		t.Fatal(err)
	}
	if len(s.Accounts) != len(want) {
		t.Fatalf("load removed user accounts: got %d, want %d", len(s.Accounts), len(want))
	}
	for i := range want {
		if s.Accounts[i].ID != want[i].ID {
			t.Fatalf("account order/content changed at %d: got %q, want %q", i, s.Accounts[i].ID, want[i].ID)
		}
	}
}

func TestAPIKeyProbeExtractsWorkspaceFromCreditsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zen/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Fatalf("unexpected probe request: %s %q", r.URL.Path, r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"CreditsError","message":"没有付款方式。请在此处添加付款方式：https://opencode.ai/workspace/wrk_probe123/billing"}}`))
	}))
	defer server.Close()
	oldBase := apiKeyProbeBase
	apiKeyProbeBase = server.URL
	defer func() { apiKeyProbeBase = oldBase }()
	result, err := probeAPIKey("sk-test", "zen")
	if err != nil || result.Enabled || result.WorkspaceID != "wrk_probe123" || result.Service != "zen" {
		t.Fatalf("unexpected probe result: %+v, %v", result, err)
	}
}

func TestAPIKeyProbeDetectsEnabledService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"id":"response-test"}`)) }))
	defer server.Close()
	oldBase := apiKeyProbeBase
	apiKeyProbeBase = server.URL
	defer func() { apiKeyProbeBase = oldBase }()
	result, err := probeAPIKey("sk-test", "go")
	if err != nil || !result.Enabled || result.Service != "go" {
		t.Fatalf("unexpected probe result: %+v, %v", result, err)
	}
}

func TestAPIKeyProbeAcceptsRegionErrorAndExtractsWorkspace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"RegionError","message":"The latest version of this model is only available hosted in China and requires explicit opt in: https://opencode.ai/workspace/wrk_region123/go"}}`))
	}))
	defer server.Close()
	oldBase := apiKeyProbeBase
	apiKeyProbeBase = server.URL
	defer func() { apiKeyProbeBase = oldBase }()
	result, err := probeAPIKey("sk-test", "go")
	if err != nil || !result.Enabled || result.WorkspaceID != "wrk_region123" || result.Service != "go" {
		t.Fatalf("model-specific region restriction must confirm service entitlement: %+v, %v", result, err)
	}
}

func TestAPIKeyProbeRejectsExplicitAuthenticationError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"AuthenticationError","message":"invalid token"}}`))
	}))
	defer server.Close()
	oldBase := apiKeyProbeBase
	apiKeyProbeBase = server.URL
	defer func() { apiKeyProbeBase = oldBase }()
	if _, err := probeAPIKey("sk-test", "go"); !isInvalidAPIKeyProbeError(err) {
		t.Fatalf("explicit authentication error should reject API key: %v", err)
	}
}

func TestAPIKeyEntitlementsProbeBothServices(t *testing.T) {
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		if r.URL.Path == "/zen/v1/chat/completions" {
			w.WriteHeader(http.StatusPaymentRequired)
			_, _ = w.Write([]byte(`{"error":{"type":"CreditsError","message":"https://opencode.ai/workspace/wrk_both123/billing"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"ok"}`))
	}))
	defer server.Close()
	oldBase := apiKeyProbeBase
	apiKeyProbeBase = server.URL
	defer func() { apiKeyProbeBase = oldBase }()
	result, err := probeAPIKeyEntitlements("sk-test")
	if err != nil || result.WorkspaceID != "wrk_both123" || !result.GoEnabled || result.ZenEnabled {
		t.Fatalf("unexpected entitlement result: %+v, %v", result, err)
	}
	if requests["/zen/v1/chat/completions"] != 1 || requests["/zen/go/v1/chat/completions"] != 1 {
		t.Fatalf("both services were not probed: %+v", requests)
	}
}

func TestAPIKeyEntitlementsAllowsInconclusivePaidProbes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"error","message":"Internal server error"}}`))
	}))
	defer server.Close()
	oldBase := apiKeyProbeBase
	apiKeyProbeBase = server.URL
	defer func() { apiKeyProbeBase = oldBase }()
	result, err := probeAPIKeyEntitlements("sk-valid")
	if err != nil || result.GoEnabled || result.ZenEnabled || result.WorkspaceID != "" {
		t.Fatalf("valid key with inconclusive probes should be saved: %+v, %v", result, err)
	}
}

func TestAPIKeyEntitlementsDoesNotRequestModelCatalog(t *testing.T) {
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	oldBase := apiKeyProbeBase
	apiKeyProbeBase = server.URL
	defer func() { apiKeyProbeBase = oldBase }()
	_, _ = probeAPIKeyEntitlements("sk-test")
	for _, path := range paths {
		if strings.HasSuffix(path, "/models") {
			t.Fatalf("model catalog must not be probed: %s", path)
		}
	}
}

func TestParseDiscoveredEmail(t *testing.T) {
	for _, body := range [][]byte{
		[]byte(`"owner@example.com"`),
		[]byte(`{"data":{"userEmail":"owner@example.com"}}`),
	} {
		if got := parseDiscoveredEmail(body); got != "owner@example.com" {
			t.Fatalf("unexpected email %q", got)
		}
	}
}

func TestParseDiscoveredAPIKeysKeepsOnlyFullKeys(t *testing.T) {
	body := []byte(`{"data":[{"id":"key-1","name":"Default","email":"owner@example.com","key":"sk-full-secret-value"},{"id":"key-2","name":"Other member","key":"sk-abc...xyz"}]}`)
	keys := parseDiscoveredAPIKeys(body)
	if len(keys) != 1 || keys[0].ID != "key-1" || keys[0].Key != "sk-full-secret-value" {
		t.Fatalf("unexpected keys: %+v", keys)
	}
	if strings.Contains(keys[0].Display, "full-secret") || !strings.HasPrefix(keys[0].Display, "sk-ful") {
		t.Fatalf("key was not safely masked: %q", keys[0].Display)
	}
}

func TestParseDiscoveredAPIKeysFromSerovalStream(t *testing.T) {
	body := []byte(`;0x00000123;((self.$R=self.$R||{})["server-fn:1"]=[],($R=>$R[0]=[$R[1]={id:"key_123",name:"Default API Key",key:"sk-AbCdEf0123456789",timeUsed:null,userID:"usr_123",email:"owner@example.com",keyDisplay:"sk-AbCd...6789"}])($R["server-fn:1"]))`)
	keys := parseDiscoveredAPIKeys(body)
	if len(keys) != 1 {
		t.Fatalf("expected one streamed key, got %+v", keys)
	}
	if keys[0].ID != "key_123" || keys[0].Name != "Default API Key" || keys[0].Email != "owner@example.com" || keys[0].Key != "sk-AbCdEf0123456789" || keys[0].Display != "sk-AbCd...6789" {
		t.Fatalf("unexpected streamed key: %+v", keys[0])
	}
}

func TestAccountDiscoveryMultipleKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "auth=test-cookie" {
			t.Fatalf("unexpected cookie header: %q", r.Header.Get("Cookie"))
		}
		var payload struct {
			Tree struct {
				Type  int `json:"t"`
				Items []struct {
					Type  int    `json:"t"`
					Value string `json:"s"`
				} `json:"a"`
			} `json:"t"`
			Features int `json:"f"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Tree.Type != 9 || len(payload.Tree.Items) != 1 || payload.Tree.Items[0].Type != 1 || payload.Tree.Items[0].Value != "wrk_test" || payload.Features != 31 {
			t.Fatalf("request was not Seroval JSON: %+v", payload)
		}
		switch r.Header.Get("X-Server-Id") {
		case userEmailServerID:
			_, _ = w.Write([]byte(`"owner@example.com"`))
		case keyListServerID:
			_, _ = w.Write([]byte(`[{"id":"one","name":"First","key":"sk-first-secret"},{"id":"two","name":"Second","key":"sk-second-secret"}]`))
		default:
			t.Fatalf("unexpected server id")
		}
	}))
	defer server.Close()
	oldURL, oldClient := discoveryServerURL, client
	discoveryServerURL, client = server.URL, server.Client()
	defer func() { discoveryServerURL, client = oldURL, oldClient }()

	result, err := discoverAccount("wrk_test", "test-cookie")
	if err != nil {
		t.Fatal(err)
	}
	if result.Email != "owner@example.com" || len(result.APIKeys) != 2 {
		t.Fatalf("unexpected discovery result: %+v", result)
	}
}

func TestAccountDiscoveryRejectsExpiredCookie(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/authorize", http.StatusFound)
	}))
	defer server.Close()
	oldURL, oldClient := discoveryServerURL, client
	discoveryServerURL, client = server.URL, server.Client()
	defer func() { discoveryServerURL, client = oldURL, oldClient }()

	if _, err := discoverAccount("wrk_test", "expired"); err == nil || !strings.Contains(err.Error(), "Cookie") {
		t.Fatalf("expected explicit expired-cookie error, got %v", err)
	}
}
