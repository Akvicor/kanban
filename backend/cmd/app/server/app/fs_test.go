package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v4"
)

// TestWebCacheSemantics 验证静态资源缓存头、深链接回退和缺失资源的 404。
func TestWebCacheSemantics(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "static"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"index.html":        "index",
		"manifest.json":     "manifest",
		"static/app.abc.js": "static",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	e := echo.New()
	setupWebRoutes(e.Group(""), os.DirFS(root))
	tests := []struct {
		path         string
		cacheControl string
		body         string
	}{
		{path: "/static/app.abc.js", cacheControl: webCacheImmutable, body: "static"},
		{path: "/", cacheControl: webCacheRevalidate, body: "index"},
		{path: "/index.html", cacheControl: webCacheRevalidate, body: "index"},
		{path: "/board/1/panel/2", cacheControl: webCacheRevalidate, body: "index"},
		{path: "/manifest.json", cacheControl: webCacheRevalidate, body: "manifest"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("状态码=%d, body=%s", recorder.Code, recorder.Body.String())
			}
			if got := recorder.Header().Get("Cache-Control"); got != test.cacheControl {
				t.Fatalf("Cache-Control=%q, 期望=%q", got, test.cacheControl)
			}
			if recorder.Body.String() != test.body {
				t.Fatalf("响应=%q, 期望=%q", recorder.Body.String(), test.body)
			}
		})
	}

	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/static/missing.abc.js", nil))
	if recorder.Code != http.StatusNotFound || recorder.Header().Get("Cache-Control") != webCacheRevalidate {
		t.Fatalf("缺失 chunk 被长期缓存: status=%d cache=%q", recorder.Code, recorder.Header().Get("Cache-Control"))
	}
}
