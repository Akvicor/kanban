package app

import (
	"io/fs"
	"kanban/cmd/app/server/web"
	"kanban/cmd/config"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/Akvicor/glog"
	"github.com/labstack/echo/v4"
)

const (
	webCacheRevalidate = "no-cache"
	webCacheImmutable  = "public, max-age=31536000, immutable"
)

// getFS 返回前端文件系统：调试模式从磁盘实时读取，否则使用嵌入二进制的构建产物。
func getFS() fs.FS {
	if config.Global.Debug {
		glog.Debug("using live mode")
		return os.DirFS(config.Global.Server.WebPath)
	}
	glog.Debug("using embed mode")
	fss, err := fs.Sub(web.Resource, config.Global.Server.WebPath)
	if err != nil {
		glog.Fatal("get embed web file failed: %v", err)
	}
	return fss
}

// wrapImmutable 为存在的带内容哈希静态资源设置长期不可变缓存。
// 缺失的资源直接返回 404，避免旧页面请求已被替换的 chunk 时拿到 index.html。
func wrapImmutable(h http.Handler, webFS fs.FS) echo.HandlerFunc {
	return func(c echo.Context) error {
		assetPath := strings.TrimPrefix(path.Clean(c.Request().URL.Path), "/")
		if info, err := fs.Stat(webFS, assetPath); err == nil && !info.IsDir() {
			c.Response().Header().Set("Cache-Control", webCacheImmutable)
		} else {
			c.Response().Header().Set("Cache-Control", webCacheRevalidate)
			return c.String(http.StatusNotFound, "404 page not found\n")
		}
		h.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

// serveWebFile 返回根路径下真实存在的静态文件（图标、manifest 等），
// 其余路径回退到 index.html，以支持前端路由深链接。
func serveWebFile(webFS fs.FS) echo.HandlerFunc {
	fileServer := http.FileServer(http.FS(webFS))
	return func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", webCacheRevalidate)
		name := strings.TrimPrefix(path.Clean(c.Request().URL.Path), "/")
		if name == "" || !fs.ValidPath(name) {
			return writeWebIndex(c, webFS)
		}
		info, err := fs.Stat(webFS, name)
		if err != nil || info.IsDir() || name == "index.html" {
			return writeWebIndex(c, webFS)
		}
		fileServer.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

// writeWebIndex 直接输出入口 HTML，避免文件服务器把 /index.html 重定向到 /。
func writeWebIndex(c echo.Context, webFS fs.FS) error {
	content, err := fs.ReadFile(webFS, "index.html")
	if err != nil {
		return echo.ErrInternalServerError
	}
	return c.HTMLBlob(http.StatusOK, content)
}

// setupWebRoutes 注册前端页面路由。Vite 构建产物的带哈希资源位于 /static。
func setupWebRoutes(group *echo.Group, webFS fs.FS) {
	group.GET("/static/*", wrapImmutable(http.FileServer(http.FS(webFS)), webFS))

	serve := serveWebFile(webFS)
	group.GET("/", serve)
	group.GET("/*", serve)
}
