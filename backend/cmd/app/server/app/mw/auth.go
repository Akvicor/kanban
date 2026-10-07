package mw

import (
	"errors"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/common/token"
	"kanban/cmd/app/server/common/types/role"
	"kanban/cmd/app/server/service"
	"net/http"

	"github.com/Akvicor/glog"
	"github.com/labstack/echo/v4"
)

// sessionKey 是 echo.Context 中保存当前会话的键。
const sessionKey = "kanban.session"

// Auth 校验 Authorization 请求头中的设备令牌，通过后把会话放进上下文。
// 接口中的用户只从这里取，不从请求体取，以保证数据按登录用户隔离。
func Auth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		plain := token.FromHeader(c.Request().Header.Get(echo.HeaderAuthorization))
		session, err := service.Auth.Authenticate(c.Request().Context(), plain)
		if err != nil {
			var businessError *service.Error
			if errors.As(err, &businessError) {
				return resp.Fail(c, businessError.Code)
			}
			glog.Error("校验登录令牌失败: %v", err)
			return resp.Fail(c, resp.Failed)
		}
		setSession(c, session)
		return next(c)
	}
}

// FileAuth 是文件接口（/api/file）的认证：优先取 Authorization 请求头，没有时取文件 Cookie。
// 页面中的图片、音视频、PDF 和下载链接由浏览器直接请求，无法带请求头，因此使用 Cookie。
// 文件接口只读。失败时直接返回 HTTP 状态码，便于这些元素识别失败。
func FileAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		plain := token.FromHeader(c.Request().Header.Get(echo.HeaderAuthorization))
		if plain == "" {
			if cookie, err := c.Cookie(FileCookieName); err == nil {
				plain = cookie.Value
			}
		}
		session, err := service.Auth.Authenticate(c.Request().Context(), plain)
		if err != nil {
			var businessError *service.Error
			if errors.As(err, &businessError) {
				return c.NoContent(http.StatusUnauthorized)
			}
			glog.Error("校验登录令牌失败: %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		setSession(c, session)
		return next(c)
	}
}

func setSession(c echo.Context, session *service.Session) {
	c.Set(sessionKey, session)
	c.SetRequest(c.Request().WithContext(service.WithRevisionNote(c.Request().Context(), session.User.ID)))
}

const (
	// FileCookieName 是文件接口使用的 Cookie，值为设备令牌。
	FileCookieName = "kanban_file_token"
	// FileCookiePath 限定 Cookie 只发往文件接口。
	FileCookiePath = "/api/file"
	// fileCookieMaxAge 是 Cookie 的有效期。令牌在设备闲置 90 天或被吊销后失效，Cookie 随之失效；
	// 长期保存使页面重新打开时，图片不必等到前端重新写入 Cookie 才能加载。
	fileCookieMaxAge = 400 * 24 * 60 * 60
)

// SetFileCookie 写入文件 Cookie：HttpOnly，SameSite=Strict，只发往文件接口，HTTPS 下加 Secure。
func SetFileCookie(c echo.Context, plain string) {
	c.SetCookie(&http.Cookie{
		Name: FileCookieName, Value: plain, Path: FileCookiePath, MaxAge: fileCookieMaxAge,
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: c.Scheme() == "https",
	})
}

// ClearFileCookie 清除文件 Cookie，登出时调用。
func ClearFileCookie(c echo.Context) {
	c.SetCookie(&http.Cookie{
		Name: FileCookieName, Value: "", Path: FileCookiePath, MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: c.Scheme() == "https",
	})
}

// Admin 只允许管理员继续访问，需要放在 Auth 之后。
func Admin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if Session(c).User.Role != role.Admin {
			return resp.Fail(c, resp.AdminRequired)
		}
		return next(c)
	}
}

// Session 返回 Auth 放入的当前会话。只能在 Auth 之后的处理函数中调用。
func Session(c echo.Context) *service.Session {
	return c.Get(sessionKey).(*service.Session)
}
