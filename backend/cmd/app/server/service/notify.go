package service

import (
	"context"
	"errors"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/common/gmsgsend"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/common/types/locale"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"strings"
	"unicode/utf8"

	"github.com/Akvicor/gmsg"
	"github.com/Akvicor/gmsg/gmodel"
	"gorm.io/gorm"
)

// 通知相关的同步实体类型。渠道随快照下发；发送记录随卡片加载。
const (
	EntityNotifyChannel  = "notify_channel"
	EntityNotifyDelivery = "notify_delivery"
)

const (
	channelNameMaxLength   = 50
	channelAPIMaxLength    = 2048
	channelSecretMaxLength = 500
)

// NotifyChannel 是通知渠道的服务：新建、修改、删除、测试发送，以及卡片选择渠道。
// 渠道的 Token 和 Sign 只保存在服务端，推送、响应和变更日志中只表明是否已设置。
var NotifyChannel = new(notifyChannelService)

type notifyChannelService struct{}

// ChannelInput 是新建或修改渠道的输入。修改时 Token、Sign 为空表示保留原值。
type ChannelInput struct {
	Name   string
	API    string
	Token  string
	Sign   string
	Format gmodel.Type
}

// validate 校验并规范化输入。creating 为 true 时 Token 和 Sign 必填。
func (input ChannelInput) validate(creating bool) (ChannelInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.API = strings.TrimSpace(input.API)
	input.Token = strings.TrimSpace(input.Token)
	input.Sign = strings.TrimSpace(input.Sign)
	if input.Name == "" || utf8.RuneCountInString(input.Name) > channelNameMaxLength {
		return input, badRequest(resp.ChannelNameInvalid, "渠道名字为 1 到 50 个字符")
	}
	// API 允许任意 http、https 地址（包括内网）：gmsg 服务可以部署在内网，用户由管理员创建。
	// 地址中不能带查询参数和片段，实际请求的路径固定为 <API>/api/send。
	if gmsgsend.CheckAPI(input.API) != nil || len(input.API) > channelAPIMaxLength {
		return input, badRequest(resp.ChannelAPIInvalid, "API 必须是 http:// 或 https:// 开头、不带 ? 和 # 的地址")
	}
	if creating && (input.Token == "" || input.Sign == "") {
		return input, badRequest(resp.ChannelSecretEmpty, "Token 和 Sign 不能为空")
	}
	if utf8.RuneCountInString(input.Token) > channelSecretMaxLength || utf8.RuneCountInString(input.Sign) > channelSecretMaxLength {
		return input, badRequest(resp.ChannelSecretTooLong, "Token 和 Sign 不能超过 500 个字符")
	}
	if input.Format != gmodel.TypeText && input.Format != gmodel.TypeMarkdown {
		return input, badRequest(resp.ChannelFormatInvalid, "内容格式只能是文本或 Markdown")
	}
	return input, nil
}

func findChannel(ctx context.Context, userID, id int64) (*model.NotifyChannel, error) {
	channel, err := repository.Notify.FindChannel(ctx, userID, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound(resp.ChannelNotFound, "通知渠道不存在")
	}
	return channel, err
}

func recordChannel(ctx context.Context, channel *model.NotifyChannel) error {
	return recordChange(ctx, channel.UserID, hub.OpUpsert, EntityNotifyChannel, channel.ID, dro.NewNotifyChannel(channel))
}

func recordDelivery(ctx context.Context, delivery *model.NotifyDelivery) error {
	return recordChange(ctx, delivery.UserID, hub.OpUpsert, EntityNotifyDelivery, delivery.ID, dro.NewNotifyDelivery(delivery))
}

// wakeNotifier 在事务提交后让后台发送任务立即检查一次。必须在 write 中调用。
func wakeNotifier(ctx context.Context) error {
	return afterCommit(ctx, Notifier.Wake)
}

// Create 新建通知渠道。
func (s *notifyChannelService) Create(ctx context.Context, userID int64, input ChannelInput) (*model.NotifyChannel, error) {
	input, err := input.validate(true)
	if err != nil {
		return nil, err
	}
	channel := &model.NotifyChannel{
		UserID: userID, Name: input.Name, API: input.API, Token: input.Token, Sign: input.Sign, Format: input.Format, CreatedAt: nowUTC(),
	}
	err = write(ctx, func(ctx context.Context) error {
		if err := repository.Notify.CreateChannel(ctx, channel); err != nil {
			return err
		}
		return recordChannel(ctx, channel)
	})
	return channel, err
}

// Update 修改通知渠道。Token、Sign 为空时保留原值。
func (s *notifyChannelService) Update(ctx context.Context, userID, id int64, input ChannelInput) (*model.NotifyChannel, error) {
	input, err := input.validate(false)
	if err != nil {
		return nil, err
	}
	var channel *model.NotifyChannel
	err = write(ctx, func(ctx context.Context) error {
		if channel, err = findChannel(ctx, userID, id); err != nil {
			return err
		}
		channel.Name, channel.API, channel.Format = input.Name, input.API, input.Format
		if input.Token != "" {
			channel.Token = input.Token
		}
		if input.Sign != "" {
			channel.Sign = input.Sign
		}
		values := map[string]any{"name": channel.Name, "api": channel.API, "format": channel.Format, "token": channel.Token, "sign": channel.Sign}
		if err = repository.Notify.UpdateChannel(ctx, userID, id, values); err != nil {
			return err
		}
		if err = recordChannel(ctx, channel); err != nil {
			return err
		}
		return wakeNotifier(ctx)
	})
	return channel, err
}

// Delete 删除通知渠道：从所有卡片的渠道选择中移除，删除这条渠道的发送记录。
func (s *notifyChannelService) Delete(ctx context.Context, userID, id int64) error {
	return write(ctx, func(ctx context.Context) error {
		if err := repository.User.Lock(ctx, userID); err != nil {
			return err
		}
		if _, err := findChannel(ctx, userID, id); err != nil {
			return err
		}
		cardIDs, err := repository.Notify.CardIDsWithChannel(ctx, userID, id)
		if err != nil {
			return err
		}
		if err = repository.Notify.RemoveChannelFromCards(ctx, userID, id); err != nil {
			return err
		}
		for _, cardID := range cardIDs {
			card, err := repository.Card.FindByID(ctx, userID, cardID)
			if err != nil {
				return err
			}
			if err = recordCard(ctx, card); err != nil {
				return err
			}
		}
		deliveries, err := repository.Notify.ListDeliveriesByChannel(ctx, userID, id)
		if err != nil {
			return err
		}
		if err = deleteDeliveries(ctx, userID, deliveries); err != nil {
			return err
		}
		if err = repository.Notify.DeleteChannel(ctx, userID, id); err != nil {
			return err
		}
		return recordChange(ctx, userID, hub.OpDelete, EntityNotifyChannel, id, nil)
	})
}

// deleteDeliveries 删除发送记录并记录变更。
func deleteDeliveries(ctx context.Context, userID int64, deliveries []*model.NotifyDelivery) error {
	ids := make([]int64, len(deliveries))
	for i, delivery := range deliveries {
		ids[i] = delivery.ID
	}
	if err := repository.Notify.DeleteDeliveries(ctx, userID, ids); err != nil {
		return err
	}
	for _, id := range ids {
		if err := recordChange(ctx, userID, hub.OpDelete, EntityNotifyDelivery, id, nil); err != nil {
			return err
		}
	}
	return nil
}

// Test 用这条渠道发送一条测试消息，返回发送结果。不记录发送状态；标题和正文跟随用户语言。
func (s *notifyChannelService) Test(ctx context.Context, userID, id int64) error {
	channel, err := findChannel(ctx, userID, id)
	if err != nil {
		return err
	}
	title, body := "测试", "这是一条来自看板的测试通知。"
	if user, err := User.FindByID(ctx, userID); err == nil && locale.Type(uiLang(ctx, user)) == locale.En {
		title, body = "Test", "This is a test notification from Kanban."
	}
	if channel.Format == gmodel.TypeMarkdown {
		body = gmsg.Telegram.Escape(body)
	}
	err = gmsgsend.Send(ctx, notifyClient, channelOf(channel), gmsgsend.Message{Title: title, Body: body, At: nowUTC()})
	if err != nil {
		return badRequest(resp.ChannelTestFailed, "测试发送失败："+err.Error())
	}
	return nil
}

func channelOf(channel *model.NotifyChannel) gmsgsend.Channel {
	return gmsgsend.Channel{API: channel.API, Token: channel.Token, Sign: channel.Sign, Format: channel.Format}
}

// SetCardChannel 为卡片选择（on 为 true）或取消选择一条渠道。返回卡片。
func (s *notifyChannelService) SetCardChannel(ctx context.Context, userID, cardID, channelID int64, on bool) (*model.Card, error) {
	var card *model.Card
	err := write(ctx, func(ctx context.Context) error {
		var err error
		if card, _, err = usableCard(ctx, userID, cardID); err != nil {
			return err
		}
		if _, err = findChannel(ctx, userID, channelID); err != nil {
			return err
		}
		current, err := repository.Notify.ChannelIDs(ctx, userID, []int64{cardID})
		if err != nil {
			return err
		}
		selected := false
		for _, id := range current[cardID] {
			selected = selected || id == channelID
		}
		switch {
		case on && !selected:
			err = repository.Notify.AddCardChannel(ctx, &model.CardNotifyChannel{UserID: userID, CardID: cardID, ChannelID: channelID})
		case !on && selected:
			err = repository.Notify.RemoveCardChannel(ctx, userID, cardID, channelID)
		default:
			return nil
		}
		if err != nil {
			return err
		}
		if err = recordCard(ctx, card); err != nil {
			return err
		}
		return wakeNotifier(ctx)
	})
	return card, err
}

// copyCardChannels 把原卡片选择的渠道复制给副本。副本的发送状态从头开始。
func copyCardChannels(ctx context.Context, userID, fromCardID, toCardID int64) error {
	channels, err := repository.Notify.ChannelIDs(ctx, userID, []int64{fromCardID})
	if err != nil {
		return err
	}
	for _, channelID := range channels[fromCardID] {
		if err = repository.Notify.AddCardChannel(ctx, &model.CardNotifyChannel{UserID: userID, CardID: toCardID, ChannelID: channelID}); err != nil {
			return err
		}
	}
	return nil
}
