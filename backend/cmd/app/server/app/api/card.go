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

// Card 是卡片的接口。打开面板时随面板内容加载卡片；卡片归档和列表归档中的卡片按需加载。
// 写入成功后返回带标签的卡片。
var Card = new(cardApi)

type cardApi struct{}

// respondCard 返回带标签的卡片。
func (a *cardApi) respondCard(c echo.Context, card *model.Card, err error) error {
	if err != nil {
		return fail(c, err)
	}
	view, err := service.Card.View(c.Request().Context(), card)
	if err != nil {
		return fail(c, err)
	}
	return success(c, view)
}

// queryID 读取查询参数中的 ID，格式错误时直接写出响应并返回 false。
func queryID(c echo.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.QueryParam(name), 10, 64)
	if err != nil {
		_ = resp.Fail(c, resp.MalformedRequest)
		return 0, false
	}
	return id, true
}

// Archived 返回面板卡片归档中的卡片。
func (a *cardApi) Archived(c echo.Context) error {
	panelID, ok := queryID(c, "panel_id")
	if !ok {
		return nil
	}
	bundle, err := service.Card.ArchivedInPanel(c.Request().Context(), mw.Session(c).User.ID, panelID)
	if err != nil {
		return fail(c, err)
	}
	return success(c, bundle)
}

// InArchivedList 返回列表归档中某个列表的卡片。
func (a *cardApi) InArchivedList(c echo.Context) error {
	listID, ok := queryID(c, "list_id")
	if !ok {
		return nil
	}
	bundle, err := service.Card.InArchivedList(c.Request().Context(), mw.Session(c).User.ID, listID)
	if err != nil {
		return fail(c, err)
	}
	return success(c, bundle)
}

func (a *cardApi) Create(c echo.Context) error {
	input := new(dto.CreateCard)
	if !bind(c, input) {
		return nil
	}
	card, err := service.Card.Create(c.Request().Context(), mw.Session(c).User.ID, input.ListID, input.Title, input.Button)
	return a.respondCard(c, card, err)
}

func (a *cardApi) UpdateTitle(c echo.Context) error {
	input := new(dto.UpdateCardText)
	if !bind(c, input) {
		return nil
	}
	card, err := service.Card.UpdateTitle(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Text, input.BaseRevision)
	return a.respondCard(c, card, err)
}

func (a *cardApi) UpdateDescription(c echo.Context) error {
	input := new(dto.UpdateCardText)
	if !bind(c, input) {
		return nil
	}
	card, err := service.Card.UpdateDescription(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Text, input.BaseRevision)
	return a.respondCard(c, card, err)
}

func (a *cardApi) SetPriority(c echo.Context) error {
	input := new(dto.SetCardPriority)
	if !bind(c, input) {
		return nil
	}
	card, err := service.Card.SetPriority(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.PriorityLevelID)
	return a.respondCard(c, card, err)
}

func (a *cardApi) SetDates(c echo.Context) error {
	input := new(dto.SetCardDates)
	if !bind(c, input) {
		return nil
	}
	card, err := service.Card.SetDates(c.Request().Context(), mw.Session(c).User.ID, input.ID, service.CardDates{
		RemindAt: input.RemindAt, DueAt: input.DueAt, RemindNotify: input.RemindNotify, DueNotify: input.DueNotify,
	})
	return a.respondCard(c, card, err)
}

func (a *cardApi) SetLabel(c echo.Context) error {
	input := new(dto.SetCardLabel)
	if !bind(c, input) {
		return nil
	}
	card, err := service.Card.SetLabel(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.LabelID, input.On)
	return a.respondCard(c, card, err)
}

func (a *cardApi) Timer(c echo.Context) error {
	input := new(dto.CardTimer)
	if !bind(c, input) {
		return nil
	}
	card, err := service.Card.Timer(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.Action, input.Seconds)
	return a.respondCard(c, card, err)
}

func (a *cardApi) Move(c echo.Context) error {
	input := new(dto.MoveCard)
	if !bind(c, input) {
		return nil
	}
	card, err := service.Card.Move(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.ListID, input.Index)
	return a.respondCard(c, card, err)
}

func (a *cardApi) Archive(c echo.Context) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := service.Card.Archive(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}

// ArchiveAllInList 把列表中的全部卡片放入卡片归档。
func (a *cardApi) ArchiveAllInList(c echo.Context) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := service.Card.ArchiveAllInList(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}

func (a *cardApi) Restore(c echo.Context) error {
	input := new(dto.RestoreCard)
	if !bind(c, input) {
		return nil
	}
	card, err := service.Card.Restore(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.ListID, input.End)
	return a.respondCard(c, card, err)
}

func (a *cardApi) Copy(c echo.Context) error {
	input := new(dto.CopyCard)
	if !bind(c, input) {
		return nil
	}
	card, err := service.Card.Copy(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.ListID)
	return a.respondCard(c, card, err)
}
