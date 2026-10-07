// Package hub 记录每个用户在线的同步连接，把已提交的变更推送给该用户的连接，并在令牌吊销时断开连接。
// 推送只在本进程内进行；连接本身的收发由 app/ws 实现。
package hub

import (
	"encoding/json"
	"sync"
)

// Event 是推送给客户端的一条变更，与 change_log 中的记录一一对应。
// Data 是实体变更后的完整数据，删除时为 null。
type Event struct {
	Revision int64           `json:"revision"`
	Op       string          `json:"op"`
	Type     string          `json:"type"`
	ID       int64           `json:"id"`
	Data     json.RawMessage `json:"data"`
}

// 变更操作。
const (
	OpUpsert = "upsert"
	OpDelete = "delete"
)

// 服务端主动断开连接时使用的关闭码，取值在 WebSocket 应用自定义范围 4000–4999 内。
const (
	CloseRevoked   = 4001 // 设备令牌已吊销，客户端回到登录页，不再重连
	CloseBadHello  = 4002 // 握手消息无效或超时
	CloseSlow      = 4003 // 发送队列已满，客户端带着已应用的序号重连
	CloseServerErr = 4004 // 服务端读取变更失败，客户端稍后重连
)

// Client 是一条同步连接。Deliver 和 Close 都不能阻塞。
type Client interface {
	// Deliver 按序号升序交给连接发送。
	Deliver(events []Event)
	// Close 以指定关闭码断开连接。
	Close(code int, reason string)
}

type entry struct {
	deviceID int64
	client   Client
}

// Hub 是在线连接的登记表。
type Hub struct {
	mu      sync.RWMutex
	clients map[int64]map[*entry]struct{} // 用户 ID → 该用户的连接
}

// New 创建一个空的登记表。
func New() *Hub {
	return &Hub{clients: map[int64]map[*entry]struct{}{}}
}

// Register 登记一条连接，返回注销函数。连接关闭后必须调用注销函数。
func (h *Hub) Register(userID, deviceID int64, client Client) (unregister func()) {
	e := &entry{deviceID: deviceID, client: client}
	h.mu.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = map[*entry]struct{}{}
	}
	h.clients[userID][e] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			delete(h.clients[userID], e)
			if len(h.clients[userID]) == 0 {
				delete(h.clients, userID)
			}
		})
	}
}

// Publish 把一个用户的变更推送给该用户的全部连接。
func (h *Hub) Publish(userID int64, events []Event) {
	if len(events) == 0 {
		return
	}
	for _, e := range h.snapshot(userID) {
		e.client.Deliver(events)
	}
}

// DisconnectDevice 以「已吊销」断开某台设备的连接。
func (h *Hub) DisconnectDevice(userID, deviceID int64) {
	for _, e := range h.snapshot(userID) {
		if e.deviceID == deviceID {
			e.client.Close(CloseRevoked, "登录已失效")
		}
	}
}

// DisconnectUser 以「已吊销」断开某个用户的全部连接。
func (h *Hub) DisconnectUser(userID int64) {
	for _, e := range h.snapshot(userID) {
		e.client.Close(CloseRevoked, "登录已失效")
	}
}

// snapshot 复制用户当前的连接列表，使推送和断开不在持锁时调用连接的方法。
func (h *Hub) snapshot(userID int64) []*entry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	list := make([]*entry, 0, len(h.clients[userID]))
	for e := range h.clients[userID] {
		list = append(list, e)
	}
	return list
}

// Default 是服务进程使用的登记表。
var Default = New()
