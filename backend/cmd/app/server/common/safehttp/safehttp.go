// Package safehttp 提供访问外部网址的 HTTP 客户端，拒绝连接本机、内网和保留地址。
//
// 检查发生在建立 TCP 连接时，针对的是实际连接的 IP，因此重定向后的地址、页面中引用的其他地址，
// 以及 DNS 解析在检查和连接之间发生变化（DNS 重绑定）的情况都同样受到限制。
package safehttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

// MaxRedirects 是最多跟随的重定向次数。
const MaxRedirects = 5

// ErrForbiddenAddress 表示目标地址是本机、内网或保留地址。
var ErrForbiddenAddress = errors.New("不允许访问本机或内网地址")

// Policy 判断是否允许连接某个地址。
type Policy func(addr netip.AddrPort) bool

// reservedPrefixes 是 netip 自带判断之外，仍然不能访问的网段。
//
// 198.18.0.0/15（基准测试保留）不在其中：它在公网不可路由，实际只出现在透明代理的 fake-ip 解析结果里，
// 连接由代理转发到真实网站。参考项目同样允许它，拦截会使这类网络中抓不到站点图标。
var reservedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),      // 本网络
	netip.MustParsePrefix("100.64.0.0/10"),  // 运营商级 NAT
	netip.MustParsePrefix("192.0.0.0/24"),   // IETF 协议分配
	netip.MustParsePrefix("240.0.0.0/4"),    // 保留，含广播地址
	netip.MustParsePrefix("64:ff9b:1::/48"), // 本地 NAT64，内嵌地址的格式不固定
	netip.MustParsePrefix("2001:db8::/32"),  // 文档示例
}

// nat64Prefix 是众所周知的 NAT64 前缀，地址的最后 32 位是要访问的 IPv4 地址。
var nat64Prefix = netip.MustParsePrefix("64:ff9b::/96")

// PublicOnly 只允许公网单播地址。IPv4 映射的 IPv6 地址和 NAT64 地址按其中的 IPv4 判断：
// 纯 IPv6 网络通过 NAT64 访问 IPv4 网站，但内嵌的 IPv4 也可能指向内网。
func PublicOnly(addr netip.AddrPort) bool {
	ip := addr.Addr().Unmap()
	if nat64Prefix.Contains(ip) {
		bytes := ip.As16()
		ip = netip.AddrFrom4([4]byte(bytes[12:]))
	}
	if !ip.IsValid() || ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() {
		return false
	}
	for _, prefix := range reservedPrefixes {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

// New 返回按 policy 限制连接地址的客户端：不使用代理，整个请求（含重定向）最多 timeout，
// 最多跟随 MaxRedirects 次重定向，只允许 http 和 https。
func New(policy Policy, timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: timeout,
		Control: func(_, address string, _ syscall.RawConn) error {
			addr, err := netip.ParseAddrPort(address)
			if err != nil || !policy(addr) {
				return fmt.Errorf("%w: %s", ErrForbiddenAddress, address)
			}
			return nil
		},
	}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, address)
		},
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		MaxIdleConns:          4,
		IdleConnTimeout:       30 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > MaxRedirects {
				return fmt.Errorf("重定向超过 %d 次", MaxRedirects)
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("不支持的协议: %s", req.URL.Scheme)
			}
			return nil
		},
	}
}
