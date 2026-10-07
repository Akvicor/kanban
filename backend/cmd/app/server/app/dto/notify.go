package dto

import "github.com/Akvicor/gmsg/gmodel"

// NotifyChannel 是新建或修改通知渠道的请求。修改时 ID 有值，Token、Sign 为空表示保留原值。
type NotifyChannel struct {
	ID     int64       `json:"id"`
	Name   string      `json:"name"`
	API    string      `json:"api"`
	Token  string      `json:"token"`
	Sign   string      `json:"sign"`
	Format gmodel.Type `json:"format"`
}

// SetCardNotifyChannel 是卡片选择（On 为 true）或取消选择通知渠道的请求。
type SetCardNotifyChannel struct {
	ID        int64 `json:"id"`
	ChannelID int64 `json:"channel_id"`
	On        bool  `json:"on"`
}
