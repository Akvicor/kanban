package mw

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// contentSecurityPolicy 限定页面可以加载的资源：
//   - 脚本只来自本站；wasm-unsafe-eval 供上传时计算 sha256 的 WebAssembly 使用。
//   - 样式允许内联，界面库和编辑器会写入行内样式。
//   - 图片允许任意 http、https 和 data:：Markdown 描述可以引用外部图片，链接附件的站点图标是 data URL。
//   - 媒体、接口、同步连接和 iframe（PDF 预览）只来自本站；页面只能被本站嵌入。
const contentSecurityPolicy = "default-src 'self'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob: http: https:; media-src 'self' blob:; connect-src 'self'; frame-src 'self'; " +
	"frame-ancestors 'self'; object-src 'none'; base-uri 'self'; form-action 'self'"

// hstsValue 是服务自身提供 HTTPS 时发送的 Strict-Transport-Security。
const hstsValue = "max-age=31536000"

// SecurityHeaders 给所有响应加上浏览器安全响应头。
// HSTS 只在服务自身提供 HTTPS 时发送；部署在反向代理后时由代理负责。
func SecurityHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		header := c.Response().Header()
		header.Set("Content-Security-Policy", contentSecurityPolicy)
		header.Set(echo.HeaderXFrameOptions, "SAMEORIGIN")
		header.Set(echo.HeaderXContentTypeOptions, "nosniff")
		header.Set(echo.HeaderReferrerPolicy, "same-origin")
		if c.IsTLS() {
			header.Set(echo.HeaderStrictTransportSecurity, hstsValue)
		}
		return next(c)
	}
}

// apiBodyLimit 是 /api 请求体的上限：卡片描述最多 1048576 个字符，UTF-8 下最多约 4MB，再留出 JSON 的开销。
const apiBodyLimit = "8M"

// uploadChunkPath 是上传分片的接口，分片大小由上传服务自行限制（16MB）。
const uploadChunkPath = "/api/upload/chunk"

// APIBodyLimit 限制 /api 的请求体大小，上传分片接口除外。
func APIBodyLimit() echo.MiddlewareFunc {
	return middleware.BodyLimitWithConfig(middleware.BodyLimitConfig{
		Limit: apiBodyLimit,
		Skipper: func(c echo.Context) bool {
			return c.Request().URL.Path == uploadChunkPath
		},
	})
}
