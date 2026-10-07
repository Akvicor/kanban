package ws

import (
	"context"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/service"
	"slices"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const (
	// sendQueueSize 是每条连接待发送消息的上限。慢客户端的队列满时断开，由它重连补齐，不阻塞写入。
	sendQueueSize = 256
	// pendingLimit 是补齐期间暂存的推送变更上限。
	pendingLimit = 4096
	writeTimeout = 10 * time.Second
	pingInterval = 30 * time.Second
)

// client 是一条已认证的同步连接，实现 hub.Client。
//
// 登记到 hub 之后才读取补齐内容，因此读取期间提交的变更不会漏掉：
// 补齐完成前收到的推送先暂存，补齐后只发送序号大于补齐序号的部分，之后直接发送。
type client struct {
	conn *websocket.Conn

	mu      sync.Mutex
	ready   bool
	pending []hub.Event
	out     chan Message

	closeOnce sync.Once
	done      chan struct{} // 开始关闭时关闭
	closed    chan struct{} // 关闭帧发送完成后关闭
}

func newClient(conn *websocket.Conn) *client {
	return &client{
		conn:   conn,
		out:    make(chan Message, sendQueueSize),
		done:   make(chan struct{}),
		closed: make(chan struct{}),
	}
}

// Deliver 实现 hub.Client。
func (c *client) Deliver(events []hub.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.ready {
		c.pending = append(c.pending, events...)
		if len(c.pending) > pendingLimit {
			c.Close(hub.CloseSlow, "同步太慢，请重连")
		}
		return
	}
	c.enqueue(Message{Type: TypeEvents, Events: events})
}

// Close 实现 hub.Client。关闭握手在后台进行，不阻塞调用方。
func (c *client) Close(code int, reason string) {
	c.closeOnce.Do(func() {
		close(c.done)
		go func() {
			_ = c.conn.Close(websocket.StatusCode(code), reason)
			close(c.closed)
		}()
	})
}

// start 发送补齐内容和 caught_up，再发送补齐期间暂存的更新变更，之后切换为直接发送。
func (c *client) start(catchUp *service.CatchUp) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if catchUp.Snapshot != nil {
		c.enqueue(Message{Type: TypeSnapshot, Revision: catchUp.Revision, Snapshot: catchUp.Snapshot})
	} else if len(catchUp.Events) > 0 {
		c.enqueue(Message{Type: TypeEvents, Events: catchUp.Events})
	}
	c.enqueue(Message{Type: TypeCaughtUp, Revision: catchUp.Revision})

	// 不同写入的推送可能先后交错到达，这里按序号排好，并去掉补齐内容中已经包含的部分。
	slices.SortFunc(c.pending, func(a, b hub.Event) int { return int(a.Revision - b.Revision) })
	newer := make([]hub.Event, 0, len(c.pending))
	for _, event := range c.pending {
		if event.Revision > catchUp.Revision {
			newer = append(newer, event)
		}
	}
	if len(newer) > 0 {
		c.enqueue(Message{Type: TypeEvents, Events: newer})
	}
	c.pending = nil
	c.ready = true
}

// enqueue 把消息放进发送队列，队列满时断开连接。调用方必须持有 c.mu。
func (c *client) enqueue(message Message) {
	select {
	case c.out <- message:
	default:
		c.Close(hub.CloseSlow, "同步太慢，请重连")
	}
}

// writeLoop 发送队列中的消息，并定时发送 ping 检测连接是否还在。连接关闭或出错时返回。
func (c *client) writeLoop(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case message := <-c.out:
			writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := wsjson.Write(writeCtx, c.conn, message)
			cancel()
			if err != nil {
				c.Close(hub.CloseSlow, "发送失败")
				return
			}
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Ping(pingCtx)
			cancel()
			if err != nil {
				c.Close(hub.CloseSlow, "连接无响应")
				return
			}
		}
	}
}
