package api

import (
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/app/dto"
	"kanban/cmd/app/server/app/mw"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/common/token"
	"kanban/cmd/app/server/service"

	"github.com/labstack/echo/v4"
)

// Auth 是登录和登出接口。
var Auth = new(authApi)

type authApi struct{}

// Login 用用户名和密码登录，为当前设备签发令牌。
func (a *authApi) Login(c echo.Context) error {
	input := new(dto.Login)
	if !bind(c, input) {
		return nil
	}
	plain, session, err := service.Auth.Login(c.Request().Context(), input.Username, input.Password, input.DeviceName)
	if err != nil {
		return fail(c, err)
	}
	return resp.SuccessWithData(c, dro.LoginResult{
		Token: plain,
		Me:    dro.NewMe(session.User, session.Device.ID),
	})
}

// Logout 吊销当前设备的令牌，并清除文件 Cookie。
func (a *authApi) Logout(c echo.Context) error {
	if err := service.Auth.Logout(c.Request().Context(), mw.Session(c)); err != nil {
		return fail(c, err)
	}
	mw.ClearFileCookie(c)
	return success(c, nil)
}

// FileCookie 把当前请求的设备令牌写入文件 Cookie，供页面中直接加载的图片、音视频和下载链接使用。
// 登录后、以及已登录的设备打开页面时调用。
func (a *authApi) FileCookie(c echo.Context) error {
	mw.SetFileCookie(c, token.FromHeader(c.Request().Header.Get(echo.HeaderAuthorization)))
	return success(c, nil)
}
