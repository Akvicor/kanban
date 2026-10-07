package api

import (
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/app/dto"
	"kanban/cmd/app/server/app/mw"
	"kanban/cmd/app/server/service"

	"github.com/labstack/echo/v4"
)

// User 是当前用户自己的信息、个人设置、密码和设备接口。设备列表通过同步获得。
var User = new(userApi)

type userApi struct{}

// Me 返回当前用户的账号、个人设置和当前设备。
func (a *userApi) Me(c echo.Context) error {
	session := mw.Session(c)
	return success(c, dro.NewMe(session.User, session.Device.ID))
}

// UpdateProfile 修改自己的用户名和昵称，返回修改后的账号信息。
func (a *userApi) UpdateProfile(c echo.Context) error {
	input := new(dto.UpdateProfile)
	if !bind(c, input) {
		return nil
	}
	user, err := service.User.UpdateProfile(c.Request().Context(), mw.Session(c).User.ID, input.Username, input.Nickname)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewAccount(user))
}

// UpdateSettings 修改个人设置，返回修改后的完整设置。
func (a *userApi) UpdateSettings(c echo.Context) error {
	input := new(dto.UpdateSettings)
	if !bind(c, input) {
		return nil
	}
	user, err := service.Setting.Update(c.Request().Context(), mw.Session(c).User.ID, service.SettingChanges{
		Timezone:            input.Timezone,
		Palette:             input.Palette,
		Locale:              input.Locale,
		PanModifier:         input.PanModifier,
		OpenMainBoardOnHome: input.OpenMainBoardOnHome,
		RemindTemplate:      input.RemindTemplate,
		DueTemplate:         input.DueTemplate,
		Shortcuts:           input.Shortcuts,
		EditorMode:          input.EditorMode,
	})
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewSettings(user))
}

// ChangePassword 修改自己的密码，当前设备保持登录，其他设备的令牌被吊销。
func (a *userApi) ChangePassword(c echo.Context) error {
	input := new(dto.ChangePassword)
	if !bind(c, input) {
		return nil
	}
	if err := service.Setting.ChangePassword(c.Request().Context(), mw.Session(c), input.CurrentPassword, input.NewPassword); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}

// RevokeDevice 踢掉自己的某台设备。踢掉当前设备等于登出。
func (a *userApi) RevokeDevice(c echo.Context) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := service.Device.Revoke(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}
