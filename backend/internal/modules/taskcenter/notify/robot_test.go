package notify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"
)

func TestSendFeishuSignedSuccess(t *testing.T) {
	t.Parallel()
	const secret = "e2e-feishu-secret"
	var received feishuRobotRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("content-type = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"msg":"success"}`))
	}))
	defer server.Close()

	res := SendFeishu(context.Background(), FeishuDeps{
		URL:       server.URL + "/open-apis/bot/v2/hook/e2e-token",
		Secret:    secret,
		Timeout:   time.Second,
		AllowHTTP: true,
	}, robotTestPayload())

	if res.Status != "success" {
		t.Fatalf("status = %q, error = %q", res.Status, res.ErrorMessage)
	}
	if received.MsgType != "text" || !strings.Contains(received.Content.Text, "TradeMind 告警") {
		t.Fatalf("unexpected request: %#v", received)
	}
	if received.Timestamp == "" || received.Sign == "" {
		t.Fatalf("missing timestamp/sign: %#v", received)
	}
	mac := hmac.New(sha256.New, []byte(received.Timestamp+"\n"+secret))
	_, _ = mac.Write(nil)
	wantSign := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if received.Sign != wantSign {
		t.Fatalf("sign = %q, want %q", received.Sign, wantSign)
	}
	if strings.Contains(res.Target, "e2e-token") || strings.Contains(res.Target, secret) {
		t.Fatalf("target leaked secret material: %q", res.Target)
	}
	if res.RawSummary["vendorCode"] != 0 {
		t.Fatalf("vendor summary = %#v", res.RawSummary)
	}
}

func TestSendFeishuVendorFailure(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":19024,"msg":"Key Words Not Found"}`))
	}))
	defer server.Close()

	res := SendFeishu(context.Background(), FeishuDeps{URL: server.URL, AllowHTTP: true}, robotTestPayload())
	if res.Status != "failed" || !strings.Contains(res.ErrorMessage, "19024") {
		t.Fatalf("unexpected result: %#v", res)
	}
}

func TestSendFeishuLegacySuccessResponse(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"StatusCode":0,"StatusMessage":"success"}`))
	}))
	defer server.Close()

	res := SendFeishu(context.Background(), FeishuDeps{URL: server.URL, AllowHTTP: true}, robotTestPayload())
	if res.Status != "success" {
		t.Fatalf("unexpected result: %#v", res)
	}
}

func TestSendWeComSuccess(t *testing.T) {
	t.Parallel()
	var received weComRobotRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	res := SendWeCom(context.Background(), WeComDeps{
		URL:       server.URL + "/cgi-bin/webhook/send?key=e2e-key",
		Timeout:   time.Second,
		AllowHTTP: true,
	}, robotTestPayload())

	if res.Status != "success" {
		t.Fatalf("status = %q, error = %q", res.Status, res.ErrorMessage)
	}
	if received.MsgType != "text" || !strings.Contains(received.Text.Content, "原生机器人适配") {
		t.Fatalf("unexpected request: %#v", received)
	}
	if len(received.Text.Content) > 1900 || !utf8.ValidString(received.Text.Content) {
		t.Fatalf("invalid content size/encoding: bytes=%d", len(received.Text.Content))
	}
	if strings.Contains(res.Target, "e2e-key") {
		t.Fatalf("target leaked webhook key: %q", res.Target)
	}
}

func TestSendWeComVendorFailure(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":93000,"errmsg":"invalid webhook key e2e-key"}`))
	}))
	defer server.Close()

	res := SendWeCom(context.Background(), WeComDeps{URL: server.URL + "?key=e2e-key", AllowHTTP: true}, robotTestPayload())
	if res.Status != "failed" || !strings.Contains(res.ErrorMessage, "93000") {
		t.Fatalf("unexpected result: %#v", res)
	}
	if strings.Contains(res.ErrorMessage, "e2e-key") || strings.Contains(res.RawSummary["vendorMessage"].(string), "e2e-key") {
		t.Fatalf("vendor response leaked webhook key: %#v", res)
	}
}

func TestRobotSenderRejectsPlainHTTPWhenDisabled(t *testing.T) {
	t.Parallel()
	var called atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called.Store(true)
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	res := SendWeCom(context.Background(), WeComDeps{URL: server.URL}, robotTestPayload())
	if res.Status != "skipped" || res.ErrorMessage != "http not allowed" {
		t.Fatalf("unexpected result: %#v", res)
	}
	if called.Load() {
		t.Fatal("plain HTTP endpoint was called")
	}
}

func TestRobotSenderRejectsInvalidVendorJSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer server.Close()

	res := SendFeishu(context.Background(), FeishuDeps{URL: server.URL, AllowHTTP: true}, robotTestPayload())
	if res.Status != "failed" || res.ErrorMessage != "invalid feishu response" {
		t.Fatalf("unexpected result: %#v", res)
	}
}

func robotTestPayload() AlertNotificationPayload {
	return AlertNotificationPayload{
		Severity:          "high",
		FailureCategory:   "provider",
		Title:             "原生机器人适配失败",
		Message:           strings.Repeat("告警摘要", 300),
		SuggestedAction:   "检查机器人配置",
		TaskType:          "inventory_sync",
		DetailURL:         "https://admin.example.test/ops/task-center/alerts",
		OccurredAtRFC3339: "2026-08-11T08:00:00Z",
	}
}
