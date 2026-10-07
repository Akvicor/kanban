package dro

import (
	"kanban/cmd/app/server/model"
	"time"

	"github.com/Akvicor/gmsg/gmodel"
)

// NotifyChannel 是一条通知渠道。Token 和 Sign 不回显，只表明是否已设置。
type NotifyChannel struct {
	ID        int64       `json:"id"`
	Name      string      `json:"name"`
	API       string      `json:"api"`
	Format    gmodel.Type `json:"format"`
	TokenSet  bool        `json:"token_set"`
	SignSet   bool        `json:"sign_set"`
	CreatedAt time.Time   `json:"created_at"`
}

// NewNotifyChannel 从渠道模型生成渠道信息。
func NewNotifyChannel(channel *model.NotifyChannel) NotifyChannel {
	return NotifyChannel{
		ID: channel.ID, Name: channel.Name, API: channel.API, Format: channel.Format,
		TokenSet: channel.Token != "", SignSet: channel.Sign != "", CreatedAt: channel.CreatedAt,
	}
}

// NewNotifyChannels 生成渠道信息列表。
func NewNotifyChannels(channels []*model.NotifyChannel) []NotifyChannel {
	list := make([]NotifyChannel, 0, len(channels))
	for _, channel := range channels {
		list = append(list, NewNotifyChannel(channel))
	}
	return list
}

// NotifyDelivery 是一张卡片的一种通知（remind 或 due）在一条渠道上的发送记录。
// TargetAt 是针对的时间值；SentAt 不为空表示已发送；否则 Attempts 大于 0 表示发送失败，
// NextAttemptAt 是下次重试时间，LastError 是最后一次错误。
type NotifyDelivery struct {
	ID            int64      `json:"id"`
	CardID        int64      `json:"card_id"`
	Kind          string     `json:"kind"`
	ChannelID     int64      `json:"channel_id"`
	TargetAt      time.Time  `json:"target_at"`
	SentAt        *time.Time `json:"sent_at"`
	Attempts      int        `json:"attempts"`
	NextAttemptAt time.Time  `json:"next_attempt_at"`
	LastError     string     `json:"last_error"`
}

// NewNotifyDelivery 从发送记录模型生成发送记录信息。
func NewNotifyDelivery(delivery *model.NotifyDelivery) NotifyDelivery {
	return NotifyDelivery{
		ID: delivery.ID, CardID: delivery.CardID, Kind: delivery.Kind, ChannelID: delivery.ChannelID,
		TargetAt: delivery.TargetAt, SentAt: delivery.SentAt, Attempts: delivery.Attempts,
		NextAttemptAt: delivery.NextAttemptAt, LastError: delivery.LastError,
	}
}
