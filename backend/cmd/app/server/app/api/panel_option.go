package api

import (
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/app/dto"
	"kanban/cmd/app/server/app/mw"
	"kanban/cmd/app/server/service"

	"github.com/labstack/echo/v4"
)

// PanelOption 是面板中标签和优先级挡位的接口：新建、修改、排序、删除。
var PanelOption = new(panelOptionApi)

type panelOptionApi struct{}

func (a *panelOptionApi) CreateLabel(c echo.Context) error {
	input := new(dto.CreateOption)
	if !bind(c, input) {
		return nil
	}
	label, err := service.Label.Create(c.Request().Context(), mw.Session(c).User.ID, input.PanelID, input.Name, input.Color)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewLabel(label))
}

func (a *panelOptionApi) UpdateLabel(c echo.Context) error {
	input := new(dto.UpdateOption)
	if !bind(c, input) {
		return nil
	}
	label, err := service.Label.Update(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Name, input.Color)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewLabel(label))
}

func (a *panelOptionApi) ReorderLabel(c echo.Context) error {
	input := new(dto.Reorder)
	if !bind(c, input) {
		return nil
	}
	label, err := service.Label.Reorder(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Index)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewLabel(label))
}

func (a *panelOptionApi) DeleteLabel(c echo.Context) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := service.Label.Delete(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}

func (a *panelOptionApi) CreatePriority(c echo.Context) error {
	input := new(dto.CreateOption)
	if !bind(c, input) {
		return nil
	}
	level, err := service.PriorityLevel.Create(c.Request().Context(), mw.Session(c).User.ID, input.PanelID, input.Name, input.Color)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewPriorityLevel(level))
}

func (a *panelOptionApi) UpdatePriority(c echo.Context) error {
	input := new(dto.UpdateOption)
	if !bind(c, input) {
		return nil
	}
	level, err := service.PriorityLevel.Update(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Name, input.Color)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewPriorityLevel(level))
}

func (a *panelOptionApi) ReorderPriority(c echo.Context) error {
	input := new(dto.Reorder)
	if !bind(c, input) {
		return nil
	}
	level, err := service.PriorityLevel.Reorder(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Index)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewPriorityLevel(level))
}

func (a *panelOptionApi) DeletePriority(c echo.Context) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := service.PriorityLevel.Delete(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}
