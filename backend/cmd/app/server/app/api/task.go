package api

import (
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/app/dto"
	"kanban/cmd/app/server/app/mw"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/service"

	"github.com/labstack/echo/v4"
)

// Task 是卡片中任务和关联的接口。写入成功后返回受影响的任务或关联。
var Task = new(taskApi)

type taskApi struct{}

func respondTask(c echo.Context, task *model.Task, err error) error {
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewTask(task))
}

// Create 新建一个或多个任务，返回新建的任务。
func (a *taskApi) Create(c echo.Context) error {
	input := new(dto.CreateTasks)
	if !bind(c, input) {
		return nil
	}
	tasks, err := service.Task.Create(c.Request().Context(), mw.Session(c).User.ID, input.CardID, input.ParentID, input.Titles)
	if err != nil {
		return fail(c, err)
	}
	return success(c, taskViews(tasks))
}

// taskViews 把一组任务转换为响应数据。
func taskViews(tasks []*model.Task) []dro.Task {
	views := make([]dro.Task, 0, len(tasks))
	for _, task := range tasks {
		views = append(views, dro.NewTask(task))
	}
	return views
}

func (a *taskApi) Rename(c echo.Context) error {
	input := new(dto.RenameTask)
	if !bind(c, input) {
		return nil
	}
	task, err := service.Task.Rename(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Title)
	return respondTask(c, task, err)
}

// SetDone 勾选或取消勾选任务，返回状态被改动的任务（勾选时含随之勾选的下级任务）。
func (a *taskApi) SetDone(c echo.Context) error {
	input := new(dto.SetTaskDone)
	if !bind(c, input) {
		return nil
	}
	tasks, err := service.Task.SetDone(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Done)
	if err != nil {
		return fail(c, err)
	}
	return success(c, taskViews(tasks))
}

func (a *taskApi) Move(c echo.Context) error {
	input := new(dto.MoveTask)
	if !bind(c, input) {
		return nil
	}
	task, err := service.Task.Move(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.ParentID, input.Index)
	return respondTask(c, task, err)
}

func (a *taskApi) Delete(c echo.Context) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := service.Task.Delete(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}

// AddLink 给卡片加一条关联，返回新建的关联。
func (a *taskApi) AddLink(c echo.Context) error {
	input := new(dto.AddCardLink)
	if !bind(c, input) {
		return nil
	}
	link, err := service.CardLink.Add(c.Request().Context(), mw.Session(c).User.ID, input.CardID, input.BoardID, input.PanelID)
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewCardLink(link))
}

func (a *taskApi) DeleteLink(c echo.Context) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := service.CardLink.Delete(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}
