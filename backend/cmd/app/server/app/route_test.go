package app

import (
	"bytes"
	"context"
	"encoding/json"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/service"
	"kanban/cmd/app/server/testutil/dbtest"
	"kanban/cmd/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// response 是测试中解析的统一响应。
type response struct {
	Code     resp.Code       `json:"code"`
	Msg      string          `json:"msg"`
	Data     json.RawMessage `json:"data"`
	Revision int64           `json:"revision"`
}

// call 发送一次接口请求并解析响应。body 为 nil 时发送 GET。
func call(t *testing.T, e *echo.Echo, path, token string, body any) response {
	t.Helper()
	method, reader := http.MethodGet, &bytes.Buffer{}
	if body != nil {
		method = http.MethodPost
		if err := json.NewEncoder(reader).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if token != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, req)
	var result response
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("%s 响应不是 JSON: %s", path, recorder.Body.String())
	}
	return result
}

// login 通过接口登录并返回令牌。
func login(t *testing.T, e *echo.Echo, username, password string) string {
	t.Helper()
	result := call(t, e, "/api/auth/login", "", map[string]string{"username": username, "password": password, "device_name": "test"})
	if result.Code != resp.Succeeded {
		t.Fatalf("登录 %s 失败: %+v", username, result)
	}
	var data struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatal(err)
	}
	return data.Token
}

// TestRouteAuthorization 验证登录要求、管理员权限，以及登出后令牌失效。
func TestRouteAuthorization(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		config.Global.Server.WebPath = "build"
		e := echo.New()
		setupRoutes(e)

		const password = "password-123"
		if _, err := service.User.EnsureInitialAdmin(context.Background(), "admin", password); err != nil {
			t.Fatal(err)
		}
		adminToken := login(t, e, "admin", password)

		if result := call(t, e, "/api/user/me", "", nil); result.Code != resp.NotLoggedIn {
			t.Fatalf("未登录访问 /api/user/me: %+v", result)
		}
		if result := call(t, e, "/api/user/me", adminToken, nil); result.Code != resp.Succeeded {
			t.Fatalf("登录后访问 /api/user/me: %+v", result)
		}

		created := call(t, e, "/api/admin/user/create", adminToken, map[string]string{"username": "alice", "password": password})
		if created.Code != resp.Succeeded {
			t.Fatalf("管理员创建用户: %+v", created)
		}
		aliceToken := login(t, e, "alice", password)
		if result := call(t, e, "/api/admin/user/list", aliceToken, nil); result.Code != resp.AdminRequired {
			t.Fatalf("普通用户访问管理接口: %+v", result)
		}

		if result := call(t, e, "/api/auth/logout", aliceToken, map[string]string{}); result.Code != resp.Succeeded {
			t.Fatalf("登出: %+v", result)
		}
		if result := call(t, e, "/api/user/me", aliceToken, nil); result.Code != resp.SessionExpired {
			t.Fatalf("登出后令牌仍可用: %+v", result)
		}
	})
}
