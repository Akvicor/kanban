package model

import (
	"time"

	"github.com/Akvicor/gmsg/gmodel"
)

// NotifyChannel 是用户的一条 gmsg 通知渠道。发送时只使用这一条渠道自己的 API、Token 和 Sign。
// Token 和 Sign 只保存在服务端，推送、响应和变更日志中只表明是否已设置。
type NotifyChannel struct {
	ID        int64       `gorm:"column:id;primaryKey"`
	UserID    int64       `gorm:"column:user_id;not null;index"`
	Name      string      `gorm:"column:name;type:text;not null"`
	API       string      `gorm:"column:api;type:text;not null"`
	Token     string      `gorm:"column:token;type:text;not null"`
	Sign      string      `gorm:"column:sign;type:text;not null"`
	Format    gmodel.Type `gorm:"column:format;size:16;not null"` // 内容格式：text 或 markdown
	CreatedAt time.Time   `gorm:"column:created_at;not null"`
}

// TableName 返回表名。
func (*NotifyChannel) TableName() string {
	return "notify_channels"
}

// CardNotifyChannel 是卡片选择的通知渠道。提醒和截止共用这组渠道。
type CardNotifyChannel struct {
	ID        int64 `gorm:"column:id;primaryKey"`
	UserID    int64 `gorm:"column:user_id;not null;index"`
	CardID    int64 `gorm:"column:card_id;not null;uniqueIndex:idx_card_notify_channel"`
	ChannelID int64 `gorm:"column:channel_id;not null;uniqueIndex:idx_card_notify_channel;index"`
}

// TableName 返回表名。
func (*CardNotifyChannel) TableName() string {
	return "card_notify_channels"
}

// 通知类型。
const (
	NotifyRemind = "remind" // 到达提醒时间
	NotifyDue    = "due"    // 到达截止时间
)

// NotifyDelivery 是一张卡片的一种通知在一条渠道上的发送记录，每个「卡片 × 类型 × 渠道」一条。
// TargetAt 是这次针对的时间值（提醒时间或截止时间）；卡片的时间改变后，记录按新时间重置。
// SentAt 不为空表示已发送；否则 Attempts 大于 0 表示发送失败、等待重试，NextAttemptAt 是下次发送时间。
type NotifyDelivery struct {
	ID            int64      `gorm:"column:id;primaryKey"`
	UserID        int64      `gorm:"column:user_id;not null;index"`
	CardID        int64      `gorm:"column:card_id;not null;uniqueIndex:idx_notify_delivery"`
	Kind          string     `gorm:"column:kind;size:8;not null;uniqueIndex:idx_notify_delivery"`
	ChannelID     int64      `gorm:"column:channel_id;not null;uniqueIndex:idx_notify_delivery;index"`
	TargetAt      time.Time  `gorm:"column:target_at;not null"`
	SentAt        *time.Time `gorm:"column:sent_at"`
	Attempts      int        `gorm:"column:attempts;not null"`
	NextAttemptAt time.Time  `gorm:"column:next_attempt_at;not null"`
	LastError     string     `gorm:"column:last_error;type:text;not null"`
}

// TableName 返回表名。
func (*NotifyDelivery) TableName() string {
	return "notify_deliveries"
}
