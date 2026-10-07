package api

import (
	"kanban/cmd/app/server/app/dto"
	"kanban/cmd/app/server/app/mw"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/service"
	"strconv"

	"github.com/labstack/echo/v4"
)

// List 是列表的接口。打开面板时通过 Content 加载列表，之后按同步推送更新；写入成功后返回带操作配置的列表。
var List = new(listApi)

type listApi struct{}

// Content 返回面板的全部列表及其操作配置，以及读取时的同步序号。
func (a *listApi) Content(c echo.Context) error {
	panelID, err := strconv.ParseInt(c.QueryParam("panel_id"), 10, 64)
	if err != nil {
		return resp.Fail(c, resp.MalformedRequest)
	}
	content, err := service.List.Content(c.Request().Context(), mw.Session(c).User.ID, panelID)
	if err != nil {
		return fail(c, err)
	}
	return success(c, content)
}

// respondList 返回带操作配置的列表。
func (a *listApi) respondList(c echo.Context, list *model.List, err error) error {
	if err != nil {
		return fail(c, err)
	}
	view, err := service.List.View(c.Request().Context(), list)
	if err != nil {
		return fail(c, err)
	}
	return success(c, view)
}

func (a *listApi) Create(c echo.Context) error {
	input := new(dto.CreateList)
	if !bind(c, input) {
		return nil
	}
	list, err := service.List.Create(c.Request().Context(), mw.Session(c).User.ID, input.PanelID, input.Name)
	return a.respondList(c, list, err)
}

func (a *listApi) Rename(c echo.Context) error {
	input := new(dto.Rename)
	if !bind(c, input) {
		return nil
	}
	list, err := service.List.Rename(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Name)
	return a.respondList(c, list, err)
}

func (a *listApi) UpdateSettings(c echo.Context) error {
	input := new(dto.ListSettings)
	if !bind(c, input) {
		return nil
	}
	list, err := service.List.UpdateSettings(c.Request().Context(), mw.Session(c).User.ID, input.ID, service.ListSettings{
		Color: input.Color, ShowAge: input.ShowAge, SortMode: input.SortMode, SortDir: input.SortDir,
		HeadAdd: input.HeadAdd, TailAdd: input.TailAdd, RemindOff: input.RemindOff, DueOff: input.DueOff,
	})
	return a.respondList(c, list, err)
}

func (a *listApi) UpdateRules(c echo.Context) error {
	input := new(dto.ListRules)
	if !bind(c, input) {
		return nil
	}
	list, err := service.List.UpdateRules(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Rules)
	return a.respondList(c, list, err)
}

func (a *listApi) Reorder(c echo.Context) error {
	input := new(dto.Reorder)
	if !bind(c, input) {
		return nil
	}
	list, err := service.List.Reorder(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Index)
	return a.respondList(c, list, err)
}

func (a *listApi) Archive(c echo.Context) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := service.List.Archive(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}

func (a *listApi) Restore(c echo.Context) error {
	input := new(dto.RestoreList)
	if !bind(c, input) {
		return nil
	}
	list, err := service.List.Restore(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.AtStart)
	return a.respondList(c, list, err)
}
