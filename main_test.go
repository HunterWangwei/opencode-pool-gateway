package main

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

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
