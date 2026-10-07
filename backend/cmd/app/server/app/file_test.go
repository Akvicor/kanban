package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"kanban/cmd/app/server/app/mw"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/common/types/listsort"
	"kanban/cmd/app/server/global/storage"
	"kanban/cmd/app/server/repository"
	"kanban/cmd/app/server/service"
	"kanban/cmd/app/server/testutil/dbtest"
	"kanban/cmd/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

// get 发送 GET 请求，返回原始响应。
func get(e *echo.Echo, path string, headers map[string]string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, req)
	return recorder
}

// fileCookie 从响应中取出文件 Cookie。
func fileCookie(t *testing.T, recorder *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == mw.FileCookieName {
			return cookie
		}
	}
	t.Fatal("响应中没有文件 Cookie")
	return nil
}

func postRaw(t *testing.T, e *echo.Echo, path, token string, body []byte) response {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, "application/octet-stream")
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, req)
	var result response
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("%s 响应不是 JSON: %s", path, recorder.Body.String())
	}
	return result
}

// TestFileDownloadWithCookie 验证文件接口：Cookie 认证、Range、下载文件名，以及登出后 Cookie 失效。
func TestFileDownloadWithCookie(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		config.Global.Server.WebPath = "build"
		store, err := storage.Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		storage.Set(store)
		defer storage.Set(nil)
		e := echo.New()
		setupRoutes(e)
		ctx := context.Background()

		const password = "password-123"
		if _, err = service.User.EnsureInitialAdmin(ctx, "admin", password); err != nil {
			t.Fatal(err)
		}
		admin, err := repository.User.FindByUsername(ctx, "admin")
		if err != nil {
			t.Fatal(err)
		}
		token := login(t, e, "admin", password)
		board, err := service.Board.Create(ctx, admin.ID, nil, "看板", nil)
		if err != nil {
			t.Fatal(err)
		}
		panels, _ := repository.Panel.ListActiveInBoard(ctx, admin.ID, board.ID)
		list, err := service.List.Create(ctx, admin.ID, panels[0].ID, "待办")
		if err != nil {
			t.Fatal(err)
		}
		card, err := service.Card.Create(ctx, admin.ID, list.ID, "卡片", listsort.Tail)
		if err != nil {
			t.Fatal(err)
		}

		// 通过接口上传：prepare 后一次发送全部内容。
		content := []byte("<html><script>alert(1)</script></html>")
		hash := sha256.Sum256(content)
		sha := hex.EncodeToString(hash[:])
		prepared := call(t, e, "/api/upload/prepare", token, map[string]any{"sha256": sha, "size": len(content)})
		var state struct {
			SessionID int64 `json:"session_id"`
			Done      bool  `json:"done"`
		}
		_ = json.Unmarshal(prepared.Data, &state)
		chunk := postRaw(t, e, fmt.Sprintf("/api/upload/chunk?session_id=%d&offset=0", state.SessionID), token, content)
		if _ = json.Unmarshal(chunk.Data, &state); chunk.Code != resp.Succeeded || !state.Done {
			t.Fatalf("上传分片: %+v", chunk)
		}
		created := call(t, e, "/api/attachment/create_file", token, map[string]any{"card_id": card.ID, "sha256": sha, "name": "页面 Page.html"})
		var attachment struct {
			ID int64 `json:"id"`
		}
		if _ = json.Unmarshal(created.Data, &attachment); created.Code != resp.Succeeded || attachment.ID == 0 {
			t.Fatalf("新建附件: %+v", created)
		}
		path := fmt.Sprintf("/api/file/attachment/%d", attachment.ID)

		if recorder := get(e, path, nil); recorder.Code != http.StatusUnauthorized {
			t.Fatalf("未认证下载状态 = %d", recorder.Code)
		}

		issued := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/auth/file_cookie", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
		e.ServeHTTP(issued, req)
		cookie := fileCookie(t, issued)
		if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != mw.FileCookiePath || cookie.Value != token {
			t.Fatalf("文件 Cookie = %+v", cookie)
		}

		full := get(e, path, map[string]string{"Accept-Encoding": "gzip"}, cookie)
		header := full.Header()
		if full.Code != http.StatusOK || !bytes.Equal(full.Body.Bytes(), content) {
			t.Fatalf("下载状态 = %d, 内容 = %q", full.Code, full.Body.String())
		}
		// HTML 一律作为下载，不在页面中执行；文件名保留大小写并按 RFC 5987 编码。
		if disposition := header.Get(echo.HeaderContentDisposition); !strings.HasPrefix(disposition, "attachment;") ||
			!strings.Contains(disposition, "filename*=utf-8''%E9%A1%B5%E9%9D%A2%20Page.html") {
			t.Fatalf("Content-Disposition = %q", disposition)
		}
		if header.Get("X-Content-Type-Options") != "nosniff" || header.Get(echo.HeaderContentEncoding) != "" {
			t.Fatalf("响应头 = %v", header)
		}

		// 缓存每次都要向服务端验证；内容未变时返回 304。
		if header.Get("Cache-Control") != "private, no-cache" {
			t.Fatalf("Cache-Control = %q", header.Get("Cache-Control"))
		}
		if revalidated := get(e, path, map[string]string{"If-None-Match": header.Get("ETag")}, cookie); revalidated.Code != http.StatusNotModified {
			t.Fatalf("验证缓存状态 = %d", revalidated.Code)
		}

		partial := get(e, path, map[string]string{"Range": "bytes=0-5", "Accept-Encoding": "gzip"}, cookie)
		if partial.Code != http.StatusPartialContent || partial.Body.String() != "<html>" {
			t.Fatalf("Range 状态 = %d, 内容 = %q", partial.Code, partial.Body.String())
		}

		logout := httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodPost, "/api/auth/logout", strings.NewReader("{}"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
		e.ServeHTTP(logout, req)
		if cleared := fileCookie(t, logout); cleared.MaxAge >= 0 {
			t.Fatalf("登出后 Cookie 没有清除: %+v", cleared)
		}
		if recorder := get(e, path, nil, cookie); recorder.Code != http.StatusUnauthorized {
			t.Fatalf("令牌吊销后用旧 Cookie 下载状态 = %d", recorder.Code)
		}
	})
}
