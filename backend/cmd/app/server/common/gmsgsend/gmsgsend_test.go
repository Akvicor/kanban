package gmsgsend

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Akvicor/gmsg/gmodel"
)

func TestSendUsesChannelCredentials(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/send" || r.Header.Get("X-Access-Token") != "token-a" {
			t.Errorf("请求 = %s %s %v", r.Method, r.URL.Path, r.Header)
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = io.WriteString(w, `{"code":0,"data":42}`)
	}))
	defer server.Close()

	channel := Channel{API: server.URL + "/", Token: "token-a", Sign: "sign-a", Format: gmodel.TypeMarkdown}
	if err := Send(context.Background(), server.Client(), channel, Message{Title: "提醒", Body: "正文", At: time.Unix(100, 0)}); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"id": float64(0), "sign": "sign-a", "type": "markdown", "at": float64(100), "title": "提醒", "msg": "正文"}
	for key, value := range want {
		if body[key] != value {
			t.Fatalf("请求体 %s = %v, want %v", key, body[key], value)
		}
	}
}

func TestSendFailuresHideSecrets(t *testing.T) {
	responses := []string{`{"code":4,"msg":"token token-a 无效"}`, `{"code":0,"data":0}`, `not json`}
	for _, response := range responses {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, response)
		}))
		err := Send(context.Background(), server.Client(), Channel{API: server.URL, Token: "token-a", Sign: "sign-a", Format: gmodel.TypeText}, Message{})
		server.Close()
		if err == nil || strings.Contains(err.Error(), "token-a") {
			t.Fatalf("响应 %s 的错误 = %v", response, err)
		}
	}
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, strings.Repeat("x", 2000)+"sign-a", http.StatusBadGateway)
	}))
	defer failing.Close()
	err := Send(context.Background(), failing.Client(), Channel{API: failing.URL, Token: "t", Sign: "sign-a", Format: gmodel.TypeText}, Message{})
	if err == nil || len([]rune(err.Error())) > MaxErrorLength || strings.Contains(err.Error(), "sign-a") {
		t.Fatalf("HTTP 错误 = %v", err)
	}
}

func TestSendFailuresOmitResponseBody(t *testing.T) {
	cases := []struct {
		status int
		body   string
	}{
		{http.StatusInternalServerError, "internal-secret-body"},
		{http.StatusOK, `{"AccessKeyId":"internal-secret-body"}x`},
		{http.StatusOK, "internal-secret-body"},
	}
	for _, c := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(c.status)
			_, _ = io.WriteString(w, c.body)
		}))
		err := Send(context.Background(), NewClient(time.Second), Channel{API: server.URL, Token: "t", Sign: "s", Format: gmodel.TypeText}, Message{})
		server.Close()
		if err == nil || strings.Contains(err.Error(), "internal-secret-body") {
			t.Fatalf("状态 %d 响应 %q 的错误 = %v", c.status, c.body, err)
		}
	}
}

func TestSendDoesNotFollowRedirects(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("重定向目标不应收到请求")
		_, _ = io.WriteString(w, `{"code":0,"data":42}`)
	}))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/api/send", http.StatusFound)
	}))
	defer redirect.Close()

	err := Send(context.Background(), NewClient(time.Second), Channel{API: redirect.URL, Token: "t", Sign: "s", Format: gmodel.TypeText}, Message{})
	if err == nil || !strings.Contains(err.Error(), "302") {
		t.Fatalf("重定向的错误 = %v", err)
	}
}

func TestCheckAPI(t *testing.T) {
	for _, api := range []string{"http://10.0.0.2:8080", "https://msg.example.com/gmsg/"} {
		if err := CheckAPI(api); err != nil {
			t.Fatalf("CheckAPI(%q) = %v", api, err)
		}
	}
	for _, api := range []string{"", "ftp://msg.example.com", "https://", "https://msg.example.com/x?", "https://msg.example.com/x?a=1", "https://msg.example.com/#a"} {
		if err := CheckAPI(api); err == nil {
			t.Fatalf("CheckAPI(%q) 应返回错误", api)
		}
	}
	// 库中已有的不合规渠道在发送时同样被拒绝，不发出请求。
	err := Send(context.Background(), NewClient(time.Second), Channel{API: "http://127.0.0.1:1/x?", Token: "token-a", Sign: "sign-a"}, Message{})
	if err == nil || !strings.Contains(err.Error(), ErrInvalidAPI.Error()) {
		t.Fatalf("不合规 API 发送的错误 = %v", err)
	}
}
