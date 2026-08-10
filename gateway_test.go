package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEmptyManagementListsAreArrays(t *testing.T) {
	logs := requestLogStore{}
	logResponse := httptest.NewRecorder()
	logs.logsHandler(logResponse, httptest.NewRequest(http.MethodGet, "/api/logs", nil))
	var logItems []RequestLog
	if err := json.Unmarshal(logResponse.Body.Bytes(), &logItems); err != nil || string(logResponse.Body.Bytes()) == "null\n" {
		t.Fatalf("empty logs must be a JSON array: %q, %v", logResponse.Body.String(), err)
	}
	tokens := tokenManager{}
	tokenResponse := httptest.NewRecorder()
	tokens.tokensHandler(tokenResponse, httptest.NewRequest(http.MethodGet, "/api/tokens", nil))
	var tokenItems []AccessToken
	if err := json.Unmarshal(tokenResponse.Body.Bytes(), &tokenItems); err != nil || string(tokenResponse.Body.Bytes()) == "null\n" {
		t.Fatalf("empty tokens must be a JSON array: %q, %v", tokenResponse.Body.String(), err)
	}
}

func TestGatewayTokenAndTransparentForwarding(t *testing.T) {
	var gotPath, gotQuery, gotBody, gotAuth, gotCustom string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery, gotAuth, gotCustom = r.URL.Path, r.URL.RawQuery, r.Header.Get("Authorization"), r.Header.Get("X-Client-Test")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("X-Upstream-Test", "yes")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"usage":{"input_tokens":3,"output_tokens":4}}`))
	}))
	defer upstream.Close()
	oldUpstream := gatewayUpstream
	oldAccounts, oldConfig := store.Accounts, gateway.Config
	oldTokenItems, oldTokenPath := accessTokens.Tokens, accessTokens.path
	oldLogItems, oldLogPath := requestLogs.Logs, requestLogs.path
	defer func() {
		gatewayUpstream, store.Accounts, gateway.Config = oldUpstream, oldAccounts, oldConfig
		accessTokens.Tokens, accessTokens.path = oldTokenItems, oldTokenPath
		requestLogs.Logs, requestLogs.path = oldLogItems, oldLogPath
	}()
	gatewayUpstream = upstream.URL
	store.Accounts = []Account{{ID: "a1", Name: "first", APIKey: "sk-upstream", GoAvailable: true}}
	gateway.Config = GatewayConfig{Mode: "round_robin"}
	plain := "gq_client_secret"
	accessTokens.Tokens = []AccessToken{{ID: "t1", TokenHash: tokenHash(plain), Enabled: true}}
	accessTokens.path = t.TempDir() + "/tokens.json"
	requestLogs.path = t.TempDir() + "/requests.jsonl"

	req := httptest.NewRequest(http.MethodPost, "/zen/go/v1/chat/completions?beta=1&x=a%2Fb", strings.NewReader(`{"model":"m1","messages":[]}`))
	req.Header.Set("Authorization", "Bearer "+plain)
	req.Header.Set("X-Client-Test", "kept")
	rec := httptest.NewRecorder()
	accessTokens.requireToken(http.HandlerFunc(gateway.proxyHandler)).ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || rec.Header().Get("X-Upstream-Test") != "yes" {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
	if gotPath != "/zen/go/v1/chat/completions" || gotQuery != "beta=1&x=a%2Fb" || gotBody != `{"model":"m1","messages":[]}` || gotAuth != "Bearer sk-upstream" || gotCustom != "kept" {
		t.Fatalf("request was changed: path=%q query=%q body=%q auth=%q custom=%q", gotPath, gotQuery, gotBody, gotAuth, gotCustom)
	}
	if len(requestLogs.Logs) != 1 || requestLogs.Logs[0].InputTokens != 3 || requestLogs.Logs[0].OutputTokens != 4 {
		t.Fatalf("usage log missing: %+v", requestLogs.Logs)
	}
}

func TestPriorityCandidatesAndRetryLimit(t *testing.T) {
	oldAccounts := store.Accounts
	defer func() { store.Accounts = oldAccounts }()
	store.Accounts = []Account{{ID: "high", APIKey: "1", Priority: 5}, {ID: "zero", APIKey: "2", Priority: 0}, {ID: "mid", APIKey: "3", Priority: 2}}
	g := gatewayManager{Config: GatewayConfig{Mode: "priority", RetryLimit: 2}}
	got, _ := g.candidates("/zen/v1/models", "")
	if len(got) != 2 || got[0].ID != "zero" || got[1].ID != "mid" {
		t.Fatalf("unexpected candidates: %+v", got)
	}
}

func TestCredentialModelEntitlements(t *testing.T) {
	oldAccounts := store.Accounts
	defer func() { store.Accounts = oldAccounts }()
	store.Accounts = []Account{
		{ID: "free", APIKey: "1", Type: "free"},
		{ID: "go", APIKey: "2", Type: "go", GoAvailable: true},
		{ID: "zen", APIKey: "3", Type: "zen", ZenAvailable: true, ZenBillingEnabled: true},
	}
	goAccounts := eligibleAccounts("/zen/go/v1/chat/completions", "glm-5")
	if len(goAccounts) != 1 || goAccounts[0].ID != "go" {
		t.Fatalf("unexpected Go candidates: %+v", goAccounts)
	}
	freeAccounts := eligibleAccounts("/zen/v1/chat/completions", "deepseek-v4-flash-free")
	if len(freeAccounts) != 3 {
		t.Fatalf("Zen free model should accept all credentials: %+v", freeAccounts)
	}
	paidAccounts := eligibleAccounts("/zen/v1/messages", "claude-sonnet-4")
	if len(paidAccounts) != 1 || paidAccounts[0].ID != "zen" {
		t.Fatalf("unexpected paid Zen candidates: %+v", paidAccounts)
	}
}

func TestModelAvailabilityAnnotation(t *testing.T) {
	models := []ModelInfo{{ID: "deepseek-v4-flash-free"}, {ID: "claude-sonnet-4"}}
	annotateModelAvailability(models, func(id string) (bool, string) {
		if isZenFreeModel(id) {
			return true, ""
		}
		return false, "需要 Zen 计费"
	})
	if !models[0].Available || models[1].Available || models[1].UnavailableReason == "" {
		t.Fatalf("unexpected availability: %+v", models)
	}
}

func TestCleanErrorRedactsSecrets(t *testing.T) {
	got := cleanError([]byte(`{"authorization":"Bearer secret-value-123","api_key":"sk-secretvalue123"}`))
	if strings.Contains(got, "secret-value") || strings.Contains(got, "secretvalue") {
		t.Fatalf("secret leaked: %s", got)
	}
}

func TestZenProtocolRoutes(t *testing.T) {
	want := map[string]bool{
		"/zen/v1/responses":        true,
		"/zen/v1/messages":         true,
		"/zen/v1/chat/completions": true,
		"/zen/v1/models":           true,
		"/zen/v1/models/":          true,
	}
	for _, route := range gatewayRoutes {
		delete(want, route)
	}
	if len(want) != 0 {
		t.Fatalf("missing Zen routes: %+v", want)
	}
	if got := requestModel("/zen/v1/models/gemini-3.6-flash", nil); got != "gemini-3.6-flash" {
		t.Fatalf("unexpected Gemini model: %q", got)
	}
	if got := requestModel("/zen/v1/models/gemini-3.6-flash:generateContent", nil); got != "gemini-3.6-flash" {
		t.Fatalf("unexpected Gemini action model: %q", got)
	}
}

func TestGoProtocolRoutes(t *testing.T) {
	want := map[string]bool{
		"/zen/go/v1/chat/completions": true,
		"/zen/go/v1/responses":        true,
		"/zen/go/v1/messages":         true,
		"/zen/go/v1/models":           true,
	}
	for _, route := range gatewayRoutes {
		delete(want, route)
	}
	if len(want) != 0 {
		t.Fatalf("missing Go routes: %+v", want)
	}
}

func TestUsageFromAnthropicAndGemini(t *testing.T) {
	in, out, read, write := usageFromBody([]byte(`{"usage":{"input_tokens":10,"output_tokens":20,"cache_read_input_tokens":3,"cache_creation_input_tokens":4}}`))
	if in != 10 || out != 20 || read != 3 || write != 4 {
		t.Fatalf("unexpected Anthropic usage: %d %d %d %d", in, out, read, write)
	}
	in, out, read, write = usageFromBody([]byte(`{"usageMetadata":{"promptTokenCount":11,"candidatesTokenCount":22,"cachedContentTokenCount":5}}`))
	if in != 11 || out != 22 || read != 5 || write != 0 {
		t.Fatalf("unexpected Gemini usage: %d %d %d %d", in, out, read, write)
	}
}

func TestSOCKS5ProxyConfiguration(t *testing.T) {
	raw := "socks5://user:password@127.0.0.1:1080"
	if err := validateProxy(raw); err != nil {
		t.Fatalf("SOCKS5 proxy rejected: %v", err)
	}
	transport, err := transportFor(raw)
	if err != nil {
		t.Fatalf("SOCKS5 transport failed: %v", err)
	}
	if transport.Proxy != nil || transport.DialContext == nil {
		t.Fatal("SOCKS5 dialer was not installed")
	}
	if err := validateProxy("socks4://127.0.0.1:1080"); err == nil {
		t.Fatal("unsupported proxy scheme accepted")
	}
}

func TestUsageFromSSEAndGzip(t *testing.T) {
	sse := []byte("data: {\"choices\":[]}\n\ndata: {\"usage\":{\"prompt_tokens\":99,\"completion_tokens\":123,\"prompt_tokens_details\":{\"cached_tokens\":8}}}\n\ndata: [DONE]\n")
	in, out, read, write := usageFromBody(sse)
	if in != 99 || out != 123 || read != 8 || write != 0 {
		t.Fatalf("unexpected SSE usage: %d %d %d %d", in, out, read, write)
	}
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	_, _ = zw.Write([]byte(`{"usage":{"prompt_tokens":11,"completion_tokens":22}}`))
	_ = zw.Close()
	header := http.Header{"Content-Encoding": {"gzip"}}
	in, out, _, _ = usageFromBody(responseBodyForInspection(header, compressed.Bytes()))
	if in != 11 || out != 22 {
		t.Fatalf("unexpected gzip usage: %d %d", in, out)
	}
}
