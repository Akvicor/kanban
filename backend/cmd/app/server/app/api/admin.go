package api

import (
	"context"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/app/dto"
	"kanban/cmd/app/server/app/mw"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/service"

	"github.com/labstack/echo/v4"
)

// Admin 是管理员管理账号的接口。只返回账号信息，不返回任何用户的看板和卡片。
var Admin = new(adminApi)

type adminApi struct{}

// ListUsers 返回全部账号。
func (a *adminApi) ListUsers(c echo.Context) error {
	users, err := service.User.List(c.Request().Context())
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewAdminUsers(users))
}

// CreateUser 创建普通用户。
func (a *adminApi) CreateUser(c echo.Context) error {
	input := new(dto.AdminCreateUser)
	if !bind(c, input) {
		return nil
	}
	user, err := service.User.Create(c.Request().Context(), input.Username, input.Password)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewAdminUsers([]*model.User{user})[0])
}

// DisableUser 停用账号并吊销其全部设备。
func (a *adminApi) DisableUser(c echo.Context) error {
	return a.withTarget(c, service.User.Disable)
}

// EnableUser 启用已停用的账号。
func (a *adminApi) EnableUser(c echo.Context) error {
	return a.withTarget(c, service.User.Enable)
}

// DeleteUser 删除账号及其全部数据。
func (a *adminApi) DeleteUser(c echo.Context) error {
	return a.withTarget(c, service.User.Delete)
}

// ResetPassword 重置账号密码并吊销其全部设备。
func (a *adminApi) ResetPassword(c echo.Context) error {
	input := new(dto.AdminResetPassword)
	if !bind(c, input) {
		return nil
	}
	if err := service.User.ResetPassword(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Password); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}

// withTarget 解析目标账号 ID，以当前管理员身份执行 operate。
func (a *adminApi) withTarget(c echo.Context, operate func(ctx context.Context, actorID, targetID int64) error) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := operate(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}
