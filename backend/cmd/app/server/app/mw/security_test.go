package mw

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// TestSecurityHeaders 验证响应带有安全头，HSTS 只在服务自身提供 HTTPS 时发送。
func TestSecurityHeaders(t *testing.T) {
	e := echo.New()
	e.Use(SecurityHeaders)
	e.GET("/", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })

	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "http://kanban.test/", nil))
	header := recorder.Header()
	for _, name := range []string{"Content-Security-Policy", echo.HeaderXFrameOptions, echo.HeaderXContentTypeOptions, echo.HeaderReferrerPolicy} {
		if header.Get(name) == "" {
			t.Fatalf("缺少响应头 %s", name)
		}
	}
	if header.Get(echo.HeaderStrictTransportSecurity) != "" {
		t.Fatal("HTTP 下不应发送 HSTS")
	}

	// httptest.NewRequest 对 https 地址会设置 TLS 连接状态，相当于服务自身提供 HTTPS。
	recorder = httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "https://kanban.test/", nil))
	if recorder.Header().Get(echo.HeaderStrictTransportSecurity) == "" {
		t.Fatal("HTTPS 下应发送 HSTS")
	}
}

// TestAPIBodyLimit 验证 /api 请求体超过上限时被拒绝，上传分片接口不受此限制。
func TestAPIBodyLimit(t *testing.T) {
	e := echo.New()
	api := e.Group("/api", APIBodyLimit())
	handler := func(c echo.Context) error {
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(c.Request().Body); err != nil {
			return err
		}
		return c.NoContent(http.StatusOK)
	}
	api.POST("/card/description", handler)
	api.POST("/upload/chunk", handler)

	big := bytes.Repeat([]byte("x"), 9<<20)
	cases := map[string]int{
		"/api/card/description": http.StatusRequestEntityTooLarge,
		"/api/upload/chunk":     http.StatusOK,
	}
	for path, want := range cases {
		recorder := httptest.NewRecorder()
		e.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(big)))
		if recorder.Code != want {
			t.Fatalf("%s 状态码 = %d, want %d", path, recorder.Code, want)
		}
	}
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/card/description", bytes.NewReader(big[:4<<20])))
	if recorder.Code != http.StatusOK {
		t.Fatalf("4MB 描述的状态码 = %d", recorder.Code)
	}
}
