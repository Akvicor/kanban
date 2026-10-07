package app

import (
	"fmt"
	"kanban/cmd/config"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// HTTP 服务的连接超时。只限制读取请求头和空闲连接：分片上传、文件的 Range 下载和同步连接
// 都可能持续很久，因此不设置整个请求的读写时限。
const (
	readHeaderTimeout = 10 * time.Second
	idleTimeout       = 120 * time.Second
)

// Run 注册路由并启动 HTTP 服务，阻塞直到服务退出。
func Run() error {
	server := echo.New()
	server.HideBanner = true
	extractor, err := ipExtractor(config.Global.Server)
	if err != nil {
		return err
	}
	server.IPExtractor = extractor
	setupRoutes(server)

	for _, s := range []*http.Server{server.Server, server.TLSServer} {
		s.ReadHeaderTimeout = readHeaderTimeout
		s.IdleTimeout = idleTimeout
	}

	address := fmt.Sprintf("%s:%d", config.Global.Server.HttpIp, config.Global.Server.HttpPort)
	if config.Global.Server.EnableHttps && config.Global.Server.CrtFile != "" && config.Global.Server.KeyFile != "" {
		return server.StartTLS(address, config.Global.Server.CrtFile, config.Global.Server.KeyFile)
	}
	return server.Start(address)
}

// ipExtractor 返回获取客户端 IP 的方式。没有配置受信任代理时只用连接的来源地址，客户端发来的转发头一律不采信；
// 配置后只采信来自这些代理的 X-Forwarded-For。echo 默认还信任环回、链路本地和私有网段，
// 经 Docker 端口映射进来的请求来源正是私有网段，因此这里显式关闭这三项默认信任。
func ipExtractor(server config.ServerModel) (echo.IPExtractor, error) {
	nets, err := server.TrustedProxyNets()
	if err != nil {
		return nil, err
	}
	if len(nets) == 0 {
		return echo.ExtractIPDirect(), nil
	}
	options := []echo.TrustOption{echo.TrustLoopback(false), echo.TrustLinkLocal(false), echo.TrustPrivateNet(false)}
	for _, network := range nets {
		options = append(options, echo.TrustIPRange(network))
	}
	return echo.ExtractIPFromXFFHeader(options...), nil
}
