package safehttp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"testing"
	"time"
)

func TestPublicOnly(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1:80":             false,
		"10.1.2.3:80":              false,
		"172.16.0.1:80":            false,
		"192.168.1.1:80":           false,
		"169.254.169.254:80":       false,
		"100.64.0.1:80":            false,
		"0.0.0.0:80":               false,
		"[::1]:80":                 false,
		"[fd00::1]:80":             false,
		"[fe80::1]:80":             false,
		"[::ffff:10.0.0.1]:80":     false,
		"[64:ff9b::a00:1]:80":      false, // NAT64 内嵌 10.0.0.1
		"[64:ff9b::5db8:d822]:443": true,  // NAT64 内嵌 93.184.216.34
		"198.18.1.55:443":          true,  // 透明代理的 fake-ip
		"240.0.0.1:80":             false,
		"93.184.216.34:443":        true,
		"[2606:4700::1]:443":       true,
	}
	for address, want := range cases {
		if got := PublicOnly(netip.MustParseAddrPort(address)); got != want {
			t.Errorf("PublicOnly(%s) = %v, want %v", address, got, want)
		}
	}
}

func TestBlocksLoopbackAndRedirects(t *testing.T) {
	inner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("secret"))
	}))
	defer inner.Close()
	outer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, inner.URL, http.StatusFound)
	}))
	defer outer.Close()

	// 默认策略下直接访问本机地址被拒绝。
	if _, err := New(PublicOnly, 2*time.Second).Get(inner.URL); !errors.Is(err, ErrForbiddenAddress) {
		t.Fatalf("访问本机 err = %v", err)
	}
	// 只放行外层服务的端口：外层可以访问，但它重定向到的内层地址在连接时被拒绝。
	outerURL, _ := url.Parse(outer.URL)
	allowOuter := func(addr netip.AddrPort) bool { return addr.String() == outerURL.Host }
	if _, err := New(allowOuter, 2*time.Second).Get(outer.URL); !errors.Is(err, ErrForbiddenAddress) {
		t.Fatalf("重定向到内网 err = %v", err)
	}
}
