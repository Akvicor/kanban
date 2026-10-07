package app

import (
	"kanban/cmd/config"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// TestIPExtractor 验证客户端 IP 只从受信任代理转发的 X-Forwarded-For 中读取。
func TestIPExtractor(t *testing.T) {
	cases := []struct {
		name    string
		proxies []string
		remote  string
		want    string
	}{
		{"未配置代理时忽略转发头", nil, "172.17.0.1:5000", "172.17.0.1"},
		{"来自受信任代理时采信转发头", []string{"172.17.0.1"}, "172.17.0.1:5000", "203.0.113.9"},
		{"私有网段默认不受信任", []string{"10.0.0.0/8"}, "172.17.0.1:5000", "172.17.0.1"},
		{"环回地址默认不受信任", []string{"10.0.0.0/8"}, "127.0.0.1:5000", "127.0.0.1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			extractor, err := ipExtractor(config.ServerModel{TrustedProxies: c.proxies})
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest("POST", "/api/auth/login", nil)
			req.RemoteAddr = c.remote
			req.Header.Set(echo.HeaderXForwardedFor, "203.0.113.9")
			req.Header.Set(echo.HeaderXRealIP, "198.51.100.7")
			if got := extractor(req); got != c.want {
				t.Fatalf("客户端 IP = %s, want %s", got, c.want)
			}
		})
	}
	if _, err := ipExtractor(config.ServerModel{TrustedProxies: []string{"bad"}}); err == nil {
		t.Fatal("非法代理地址应返回错误")
	}
}
