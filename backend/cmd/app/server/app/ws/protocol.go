// Package ws 实现同步用的 WebSocket 连接：握手认证、补齐断线期间的变更，然后持续推送新变更。
//
// 协议：
//  1. 客户端连接 /api/sync/ws 后，先发送 Hello，携带设备令牌和本地已应用的序号（没有本地数据时为 0）。
//  2. 服务端回复全量快照（snapshot）或补齐的变更（events），再发送 caught_up，之后持续发送 events。
//  3. 连接只推送，不接收写入；写入走 HTTP 接口。
//  4. 令牌吊销时服务端以 hub.CloseRevoked 关闭连接，客户端回到登录页；其他原因断开时客户端带着已应用的序号重连。
package ws

import (
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/global/hub"
)

// Hello 是客户端连接后发送的第一条消息。令牌放在消息里而不是地址中，避免出现在访问日志。
type Hello struct {
	Token        string `json:"token"`
	LastRevision int64  `json:"last_revision"`
}

// 服务端消息类型。
const (
	TypeSnapshot = "snapshot"  // 全量快照，Revision 是快照对应的序号
	TypeEvents   = "events"    // 按序号升序的变更
	TypeCaughtUp = "caught_up" // 补齐完成，Revision 是补齐后的序号
)

// Message 是服务端发送的消息。
type Message struct {
	Type     string        `json:"type"`
	Revision int64         `json:"revision,omitempty"`
	Snapshot *dro.Snapshot `json:"snapshot,omitempty"`
	Events   []hub.Event   `json:"events,omitempty"`
}
