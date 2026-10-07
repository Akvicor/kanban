package ws

import (
	"context"
	"errors"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/service"
	"time"

	"github.com/Akvicor/glog"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/labstack/echo/v4"
)

// helloTimeout 是连接后等待 Hello 的时限。
const helloTimeout = 10 * time.Second

// Handle 接受 WebSocket 连接并处理到连接结束。只接受同源页面发起的连接。
func Handle(c echo.Context) error {
	conn, err := websocket.Accept(c.Response(), c.Request(), nil)
	if err != nil {
		// Accept 已经写出了错误响应。
		return nil
	}
	serve(c.Request().Context(), conn)
	return nil
}

func serve(ctx context.Context, conn *websocket.Conn) {
	defer func() { _ = conn.CloseNow() }()

	var hello Hello
	helloCtx, cancel := context.WithTimeout(ctx, helloTimeout)
	err := wsjson.Read(helloCtx, conn, &hello)
	cancel()
	if err != nil || hello.LastRevision < 0 {
		_ = conn.Close(hub.CloseBadHello, "握手消息无效")
		return
	}
	session, err := service.Auth.Authenticate(ctx, hello.Token)
	if err != nil {
		var businessError *service.Error
		if errors.As(err, &businessError) {
			_ = conn.Close(hub.CloseRevoked, businessError.Msg)
			return
		}
		glog.Error("同步连接认证失败: %v", err)
		_ = conn.Close(hub.CloseServerErr, "服务器错误")
		return
	}

	// 客户端在握手后不再发送消息，CloseRead 负责处理 ping、pong 和关闭帧；对方断开时 ctx 被取消。
	ctx = conn.CloseRead(ctx)
	client := newClient(conn)
	unregister := hub.Default.Register(session.User.ID, session.Device.ID, client)
	defer unregister()
	go client.writeLoop(ctx)

	catchUp, err := service.Sync.CatchUp(ctx, session.User.ID, hello.LastRevision)
	if err != nil {
		if ctx.Err() == nil {
			glog.Error("读取同步数据失败: %v", err)
		}
		client.Close(hub.CloseServerErr, "服务器错误")
	} else {
		client.start(catchUp)
	}

	select {
	case <-ctx.Done():
	case <-client.done:
		// 等关闭帧发出后再结束，避免客户端只看到异常断开而拿不到关闭码。
		<-client.closed
	}
}
