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

// FileAuth 是文件接口（/api/file）的认证：有 Authorization 请求头时按设备令牌认证，没有时按文件 Cookie 中的文件令牌认证。
// 页面中的图片、音视频、PDF 和下载链接由浏览器直接请求，无法带请求头，因此使用 Cookie。
// 文件令牌与设备令牌分开，只能用于文件接口，可以单独注销（见 service.Auth.IssueFileToken）。
// 文件接口只读。失败时直接返回 HTTP 状态码，便于这些元素识别失败。
func FileAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		var (
			session *service.Session
			err     error
		)
		if plain := token.FromHeader(c.Request().Header.Get(echo.HeaderAuthorization)); plain != "" {
			session, err = service.Auth.Authenticate(ctx, plain)
		} else {
			fileToken := ""
			if cookie, cookieErr := c.Cookie(FileCookieName); cookieErr == nil {
				fileToken = cookie.Value
			}
			session, err = service.Auth.AuthenticateFile(ctx, fileToken)
		}
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
	// FileCookieName 是文件接口使用的 Cookie，值为文件令牌（不是设备令牌）。
	FileCookieName = "kanban_file_token"
	// FileCookiePath 限定 Cookie 只发往文件接口。
	FileCookiePath = "/api/file"
	// fileCookieMaxAge 是 Cookie 的有效期。文件令牌在注销、重新签发、设备闲置 90 天或被吊销后失效，Cookie 随之失效；
	// 长期保存使页面重新打开时，图片不必等到前端重新写入 Cookie 才能加载。
	fileCookieMaxAge = 400 * 24 * 60 * 60
)

// SetFileCookie 把文件令牌写入文件 Cookie：HttpOnly，SameSite=Strict，只发往文件接口，HTTPS 下加 Secure。
func SetFileCookie(c echo.Context, plain string) {
	c.SetCookie(&http.Cookie{
		Name: FileCookieName, Value: plain, Path: FileCookiePath, MaxAge: fileCookieMaxAge,
		HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: c.Scheme() == "https",
	})
}

// ClearFileCookie 清除文件 Cookie，登出和注销文件令牌时调用。
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
