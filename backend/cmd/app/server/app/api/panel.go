package api

import (
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/app/dto"
	"kanban/cmd/app/server/app/mw"
	"kanban/cmd/app/server/service"

	"github.com/labstack/echo/v4"
)

// Panel 是面板的接口：新建、改名、排序、移到其他看板、归档、恢复和主面板。
// 面板数据通过同步获得；写入成功后返回受影响的面板或看板。
var Panel = new(panelApi)

type panelApi struct{}

func (a *panelApi) Create(c echo.Context) error {
	input := new(dto.CreatePanel)
	if !bind(c, input) {
		return nil
	}
	panel, err := service.Panel.Create(c.Request().Context(), mw.Session(c).User.ID, input.BoardID, input.Name)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewPanel(panel))
}

func (a *panelApi) Rename(c echo.Context) error {
	input := new(dto.Rename)
	if !bind(c, input) {
		return nil
	}
	panel, err := service.Panel.Rename(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Name)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewPanel(panel))
}

func (a *panelApi) Reorder(c echo.Context) error {
	input := new(dto.Reorder)
	if !bind(c, input) {
		return nil
	}
	panel, err := service.Panel.Reorder(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Index)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewPanel(panel))
}

func (a *panelApi) Move(c echo.Context) error {
	input := new(dto.PanelToBoard)
	if !bind(c, input) {
		return nil
	}
	panel, err := service.Panel.MoveToBoard(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.BoardID)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewPanel(panel))
}

func (a *panelApi) Archive(c echo.Context) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := service.Panel.Archive(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}

func (a *panelApi) Restore(c echo.Context) error {
	input := new(dto.PanelToBoard)
	if !bind(c, input) {
		return nil
	}
	panel, err := service.Panel.Restore(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.BoardID)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewPanel(panel))
}

// SetMain 设置或取消看板的主面板，返回修改后的看板。
func (a *panelApi) SetMain(c echo.Context) error {
	input := new(dto.SetMainPanel)
	if !bind(c, input) {
		return nil
	}
	board, err := service.Panel.SetMain(c.Request().Context(), mw.Session(c).User.ID, input.BoardID, input.PanelID)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewBoard(board))
}
