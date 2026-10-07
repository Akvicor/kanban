package api

import (
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/app/dto"
	"kanban/cmd/app/server/app/mw"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/service"

	"github.com/labstack/echo/v4"
)

// NotifyChannel 是通知渠道的接口。写入成功后返回渠道信息（不含 Token 和 Sign 明文）。
var NotifyChannel = new(notifyChannelApi)

type notifyChannelApi struct{}

func (a *notifyChannelApi) respond(c echo.Context, channel *model.NotifyChannel, err error) error {
	if err != nil {
		return fail(c, err)
	}
	return success(c, dro.NewNotifyChannel(channel))
}

func channelInput(input *dto.NotifyChannel) service.ChannelInput {
	return service.ChannelInput{Name: input.Name, API: input.API, Token: input.Token, Sign: input.Sign, Format: input.Format}
}

func (a *notifyChannelApi) Create(c echo.Context) error {
	input := new(dto.NotifyChannel)
	if !bind(c, input) {
		return nil
	}
	channel, err := service.NotifyChannel.Create(c.Request().Context(), mw.Session(c).User.ID, channelInput(input))
	return a.respond(c, channel, err)
}

func (a *notifyChannelApi) Update(c echo.Context) error {
	input := new(dto.NotifyChannel)
	if !bind(c, input) {
		return nil
	}
	channel, err := service.NotifyChannel.Update(c.Request().Context(), mw.Session(c).User.ID, input.ID, channelInput(input))
	return a.respond(c, channel, err)
}

func (a *notifyChannelApi) Delete(c echo.Context) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := service.NotifyChannel.Delete(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}

// Test 用渠道发送一条测试消息，失败时返回错误信息。
func (a *notifyChannelApi) Test(c echo.Context) error {
	input := new(dto.ID)
	if !bind(c, input) {
		return nil
	}
	if err := service.NotifyChannel.Test(c.Request().Context(), mw.Session(c).User.ID, input.ID); err != nil {
		return fail(c, err)
	}
	return success(c, nil)
}

// SetCardChannel 为卡片选择或取消选择通知渠道，返回卡片。
func (a *notifyChannelApi) SetCardChannel(c echo.Context) error {
	input := new(dto.SetCardNotifyChannel)
	if !bind(c, input) {
		return nil
	}
	card, err := service.NotifyChannel.SetCardChannel(c.Request().Context(), mw.Session(c).User.ID, input.ID, input.ChannelID, input.On)
	return Card.respondCard(c, card, err)
}
