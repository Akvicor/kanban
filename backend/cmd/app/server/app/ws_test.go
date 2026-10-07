package app

import (
	"context"
	"errors"
	"kanban/cmd/app/server/app/ws"
	"kanban/cmd/app/server/common/resp"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/service"
	"kanban/cmd/app/server/testutil/dbtest"
	"kanban/cmd/config"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/labstack/echo/v4"
)

// syncClient 是测试用的同步连接。
type syncClient struct {
	t    *testing.T
	conn *websocket.Conn
}

func dialSync(t *testing.T, server *httptest.Server, token string, lastRevision int64) *syncClient {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/sync/ws", nil)
	if err != nil {
		t.Fatalf("连接同步失败: %v", err)
	}
	t.Cleanup(func() { _ = conn.CloseNow() })
	if err = wsjson.Write(ctx, conn, ws.Hello{Token: token, LastRevision: lastRevision}); err != nil {
		t.Fatal(err)
	}
	return &syncClient{t: t, conn: conn}
}

// next 读取下一条消息，失败时返回错误（例如连接被关闭）。
func (c *syncClient) next() (ws.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var message ws.Message
	err := wsjson.Read(ctx, c.conn, &message)
	return message, err
}

func (c *syncClient) mustNext(wantType string) ws.Message {
	c.t.Helper()
	message, err := c.next()
	if err != nil {
		c.t.Fatalf("读取 %s 失败: %v", wantType, err)
	}
	if message.Type != wantType {
		c.t.Fatalf("收到 %s，期望 %s: %+v", message.Type, wantType, message)
	}
	return message
}

// TestSyncOverWebSocket 验证：连接后收到快照；一台设备写入后另一台实时收到变更，响应带同步序号；
// 断线重连按序号补齐；令牌吊销时连接以 CloseRevoked 关闭；无效令牌被拒绝。
func TestSyncOverWebSocket(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		config.Global.Server.WebPath = "build"
		e := echo.New()
		setupRoutes(e)
		server := httptest.NewServer(e)
		t.Cleanup(server.Close)

		const password = "password-123"
		if _, err := service.User.EnsureInitialAdmin(context.Background(), "alice", password); err != nil {
			t.Fatal(err)
		}
		laptopToken := login(t, e, "alice", password) // 序号 1
		phoneToken := login(t, e, "alice", password)  // 序号 2

		phone := dialSync(t, server, phoneToken, 0)
		snapshot := phone.mustNext(ws.TypeSnapshot)
		if snapshot.Revision != 2 || len(snapshot.Snapshot.Devices) != 2 {
			t.Fatalf("快照 = %+v", snapshot)
		}
		phone.mustNext(ws.TypeCaughtUp)

		updated := call(t, e, "/api/user/settings/update", laptopToken, map[string]string{"palette": "paper"})
		if updated.Code != resp.Succeeded || updated.Revision != 3 {
			t.Fatalf("修改设置的响应 = %+v", updated)
		}
		events := phone.mustNext(ws.TypeEvents)
		if len(events.Events) != 1 || events.Events[0].Revision != 3 || events.Events[0].Type != service.EntitySettings {
			t.Fatalf("实时推送 = %+v", events)
		}

		// 断线期间的变更在重连时按序号补齐。
		_ = phone.conn.Close(websocket.StatusNormalClosure, "")
		call(t, e, "/api/user/settings/update", laptopToken, map[string]string{"palette": "dark"}) // 序号 4
		phone = dialSync(t, server, phoneToken, 3)
		replay := phone.mustNext(ws.TypeEvents)
		if len(replay.Events) != 1 || replay.Events[0].Revision != 4 {
			t.Fatalf("补齐 = %+v", replay)
		}
		if caughtUp := phone.mustNext(ws.TypeCaughtUp); caughtUp.Revision != 4 {
			t.Fatalf("caught_up = %+v", caughtUp)
		}

		// laptop 修改密码后，phone 的令牌被吊销，连接以 CloseRevoked 关闭。
		changed := call(t, e, "/api/user/password/update", laptopToken, map[string]string{
			"current_password": password, "new_password": "new-password-1",
		})
		if changed.Code != resp.Succeeded {
			t.Fatalf("修改密码 = %+v", changed)
		}
		for {
			_, err := phone.next()
			if err == nil {
				continue // 关闭前可能先收到设备删除的推送
			}
			if status := websocket.CloseStatus(err); status != hub.CloseRevoked {
				t.Fatalf("连接关闭码 = %d, err = %v", status, err)
			}
			break
		}

		// 已吊销的令牌不能建立同步连接。
		revoked := dialSync(t, server, phoneToken, 0)
		_, err := revoked.next()
		if status := websocket.CloseStatus(err); status != hub.CloseRevoked {
			t.Fatalf("吊销令牌连接的关闭码 = %d, err = %v", status, err)
		}
		if err == nil || errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("吊销令牌的连接没有被关闭: %v", err)
		}
	})
}
