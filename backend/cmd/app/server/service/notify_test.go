package service

import (
	"context"
	"encoding/json"
	"io"
	"kanban/cmd/app/server/common/types/listsort"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"kanban/cmd/app/server/testutil/dbtest"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Akvicor/gmsg/gmodel"
)

// fakeGmsg 是替身 gmsg 服务：记录收到的消息；Token 在 failing 中时返回错误码。
type fakeGmsg struct {
	server   *httptest.Server
	mu       sync.Mutex
	received []gmsgRequest
	failing  map[string]bool
}

type gmsgRequest struct {
	Token string
	Sign  string `json:"sign"`
	Type  string `json:"type"`
	Title string `json:"title"`
	Msg   string `json:"msg"`
}

func newFakeGmsg(t *testing.T) *fakeGmsg {
	fake := &fakeGmsg{failing: map[string]bool{}}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request gmsgRequest
		_ = json.NewDecoder(r.Body).Decode(&request)
		request.Token = r.Header.Get("X-Access-Token")
		fake.mu.Lock()
		fake.received = append(fake.received, request)
		failing := fake.failing[request.Token]
		fake.mu.Unlock()
		if failing {
			_, _ = io.WriteString(w, `{"code":1,"msg":"渠道暂时不可用"}`)
			return
		}
		_, _ = io.WriteString(w, `{"code":0,"data":7}`)
	}))
	t.Cleanup(fake.server.Close)
	return fake
}

// take 返回并清空收到的消息。
func (f *fakeGmsg) take() []gmsgRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	received := f.received
	f.received = nil
	return received
}

func (f *fakeGmsg) fail(token string, failing bool) {
	f.mu.Lock()
	f.failing[token] = failing
	f.mu.Unlock()
}

type notifyEnv struct {
	*cardEnv
	gmsg *fakeGmsg
	now  time.Time
}

func newNotifyEnv(t *testing.T) *notifyEnv {
	return &notifyEnv{cardEnv: newCardEnv(t, "待办"), gmsg: newFakeGmsg(t), now: time.Now().UTC().Truncate(time.Second)}
}

func (e *notifyEnv) channel(name, token string, format gmodel.Type) *model.NotifyChannel {
	e.t.Helper()
	channel, err := NotifyChannel.Create(e.ctx, e.user.ID, ChannelInput{Name: name, API: e.gmsg.server.URL, Token: token, Sign: "sign-" + token, Format: format})
	if err != nil {
		e.t.Fatal(err)
	}
	return channel
}

func (e *notifyEnv) choose(card *model.Card, channels ...*model.NotifyChannel) {
	e.t.Helper()
	for _, channel := range channels {
		if _, err := NotifyChannel.SetCardChannel(e.ctx, e.user.ID, card.ID, channel.ID, true); err != nil {
			e.t.Fatal(err)
		}
	}
}

func (e *notifyEnv) remindAt(card *model.Card, at time.Time, notify bool) {
	e.t.Helper()
	if _, err := Card.SetDates(e.ctx, e.user.ID, card.ID, CardDates{RemindAt: &at, RemindNotify: notify}); err != nil {
		e.t.Fatal(err)
	}
}

// run 在 now 后 offset 时执行一次检查，返回这次发出的消息。
func (e *notifyEnv) run(offset time.Duration) []gmsgRequest {
	e.t.Helper()
	if err := Notifier.RunOnce(e.ctx, e.now.Add(offset)); err != nil {
		e.t.Fatal(err)
	}
	return e.gmsg.take()
}

func (e *notifyEnv) deliveries(card *model.Card) []*model.NotifyDelivery {
	e.t.Helper()
	deliveries, err := repository.Notify.ListDeliveries(e.ctx, e.user.ID, []int64{card.ID})
	if err != nil {
		e.t.Fatal(err)
	}
	return deliveries
}

func TestNotifyChannelKeepsSecrets(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newNotifyEnv(t)
		channel := e.channel("手机", "token-secret", gmodel.TypeMarkdown)
		updated, err := NotifyChannel.Update(e.ctx, e.user.ID, channel.ID, ChannelInput{Name: "手机 Telegram", API: e.gmsg.server.URL, Format: gmodel.TypeText})
		if err != nil {
			t.Fatal(err)
		}
		if updated.Token != "token-secret" || updated.Sign != "sign-token-secret" || updated.Format != gmodel.TypeText {
			t.Fatalf("留空时应保留原 Token 和 Sign：%+v", updated)
		}
		logs, _ := repository.ChangeLog.ListAfter(e.ctx, e.user.ID, 0)
		for _, log := range logs {
			if log.Type == EntityNotifyChannel && (strings.Contains(log.Data, "token-secret") || !strings.Contains(log.Data, `"token_set":true`)) {
				t.Fatalf("变更日志 = %s", log.Data)
			}
		}
		invalid := []ChannelInput{
			{Name: "", API: "https://gmsg", Token: "t", Sign: "s", Format: gmodel.TypeText},
			{Name: "x", API: "ftp://gmsg", Token: "t", Sign: "s", Format: gmodel.TypeText},
			{Name: "x", API: "https://gmsg/x?", Token: "t", Sign: "s", Format: gmodel.TypeText},
			{Name: "x", API: "https://gmsg/x#", Token: "t", Sign: "s", Format: gmodel.TypeText},
			{Name: "x", API: "https://gmsg", Token: "", Sign: "s", Format: gmodel.TypeText},
			{Name: "x", API: "https://gmsg", Token: "t", Sign: "s", Format: gmodel.TypeHTML},
		}
		for _, input := range invalid {
			if _, err = NotifyChannel.Create(e.ctx, e.user.ID, input); errorKind(err) != KindBadRequest {
				t.Fatalf("Create(%+v) err = %v", input, err)
			}
		}
		// 内网地址的 gmsg 可以使用；测试发送使用渠道自己的 Token。
		if err = NotifyChannel.Test(e.ctx, e.user.ID, channel.ID); err != nil {
			t.Fatal(err)
		}
		if got := e.gmsg.take(); len(got) != 1 || got[0].Token != "token-secret" || got[0].Title != "测试" {
			t.Fatalf("测试发送 = %+v", got)
		}
		e.gmsg.fail("token-secret", true)
		if err = NotifyChannel.Test(e.ctx, e.user.ID, channel.ID); errorKind(err) != KindBadRequest || !strings.Contains(err.Error(), "渠道暂时不可用") {
			t.Fatalf("测试发送失败 err = %v", err)
		}
	})
}

func TestNotifySendsOncePerTimeValue(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newNotifyEnv(t)
		markdown := e.channel("Markdown", "md", gmodel.TypeMarkdown)
		text := e.channel("文本", "text", gmodel.TypeText)
		card := e.create("待办", "发布 v1.2 (beta)", listsort.Tail)
		e.choose(card, markdown, text)
		// 停机期间到达的提醒：时间早于第一次检查，恢复后补发。
		e.remindAt(card, e.now.Add(-time.Hour), true)

		sent := e.run(0)
		if len(sent) != 2 {
			t.Fatalf("应向两条渠道各发一次：%+v", sent)
		}
		for _, request := range sent {
			if request.Title != "提醒" {
				t.Fatalf("标题 = %q", request.Title)
			}
			switch request.Token {
			case "md":
				if request.Type != "markdown" || !strings.HasPrefix(request.Msg, `发布 v1\.2 \(beta\)`) {
					t.Fatalf("Markdown 渠道 = %+v", request)
				}
			case "text":
				if request.Type != "text" || !strings.HasPrefix(request.Msg, "发布 v1.2 (beta)\n") {
					t.Fatalf("文本渠道 = %+v", request)
				}
			}
		}
		if deliveries := e.deliveries(card); len(deliveries) != 2 || deliveries[0].SentAt == nil || deliveries[1].SentAt == nil {
			t.Fatalf("发送记录 = %+v", deliveries)
		}
		// 同一时间值不重复发送。
		if again := e.run(time.Minute); len(again) != 0 {
			t.Fatalf("重复发送：%+v", again)
		}
		// 修改时间后按新时间重新发送；新时间已经过去时按已到达处理。
		e.remindAt(card, e.now.Add(-time.Minute), true)
		if resent := e.run(2 * time.Minute); len(resent) != 2 {
			t.Fatalf("修改时间后 = %+v", resent)
		}
		// 时间改到未来：旧记录删除，到点前不发送，到点后发送。
		e.remindAt(card, e.now.Add(time.Hour), true)
		if early := e.run(3 * time.Minute); len(early) != 0 || len(e.deliveries(card)) != 0 {
			t.Fatalf("未到时间 = %+v, 记录 = %d", early, len(e.deliveries(card)))
		}
		if onTime := e.run(time.Hour); len(onTime) != 2 {
			t.Fatalf("到点 = %+v", onTime)
		}
	})
}

func TestNotifyRetriesFailedChannelOnly(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newNotifyEnv(t)
		good := e.channel("正常", "good", gmodel.TypeText)
		bad := e.channel("故障", "bad", gmodel.TypeText)
		e.gmsg.fail("bad", true)
		card := e.create("待办", "卡片", listsort.Tail)
		e.choose(card, good, bad)
		e.remindAt(card, e.now, true)

		if sent := e.run(0); len(sent) != 2 {
			t.Fatalf("第一次 = %+v", sent)
		}
		failed, _ := repository.Notify.FindDelivery(e.ctx, e.user.ID, card.ID, model.NotifyRemind, bad.ID)
		if failed.SentAt != nil || failed.Attempts != 1 || !failed.NextAttemptAt.Equal(e.now.Add(time.Minute)) || !strings.Contains(failed.LastError, "渠道暂时不可用") {
			t.Fatalf("失败记录 = %+v", failed)
		}
		// 未到重试时间不发送；到点只重试失败的渠道，间隔翻倍。
		if early := e.run(30 * time.Second); len(early) != 0 {
			t.Fatalf("未到重试时间 = %+v", early)
		}
		if retried := e.run(time.Minute); len(retried) != 1 || retried[0].Token != "bad" {
			t.Fatalf("第一次重试 = %+v", retried)
		}
		failed, _ = repository.Notify.FindDelivery(e.ctx, e.user.ID, card.ID, model.NotifyRemind, bad.ID)
		if failed.Attempts != 2 || !failed.NextAttemptAt.Equal(e.now.Add(3*time.Minute)) {
			t.Fatalf("第二次失败 = %+v", failed)
		}
		e.gmsg.fail("bad", false)
		if retried := e.run(3 * time.Minute); len(retried) != 1 || retried[0].Token != "bad" {
			t.Fatalf("恢复后重试 = %+v", retried)
		}
		if failed, _ = repository.Notify.FindDelivery(e.ctx, e.user.ID, card.ID, model.NotifyRemind, bad.ID); failed.SentAt == nil {
			t.Fatal("恢复后应标记为已发送")
		}
		if retryDelay(10) != time.Hour {
			t.Fatalf("重试间隔上限 = %v", retryDelay(10))
		}
	})
}

func TestNotifyStopConditions(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newNotifyEnv(t)
		channel := e.channel("渠道", "c", gmodel.TypeText)
		e.gmsg.fail("c", true)
		card := e.create("待办", "卡片", listsort.Tail)
		e.choose(card, channel)
		e.remindAt(card, e.now, true)
		e.run(0) // 失败，等待重试

		// 关闭卡片的通知开关：未发送的记录删除，不再发送。
		e.gmsg.fail("c", false)
		e.remindAt(card, e.now, false)
		if sent := e.run(time.Hour); len(sent) != 0 || len(e.deliveries(card)) != 0 {
			t.Fatalf("关闭开关后 = %+v", sent)
		}
		// 列表关闭提醒通知时不发送。
		e.remindAt(card, e.now, true)
		list := e.lists["待办"]
		if err := repository.List.Update(e.ctx, e.user.ID, list.ID, map[string]any{"remind_off": true}); err != nil {
			t.Fatal(err)
		}
		if sent := e.run(time.Hour); len(sent) != 0 {
			t.Fatalf("列表关闭通知后 = %+v", sent)
		}
		if err := repository.List.Update(e.ctx, e.user.ID, list.ID, map[string]any{"remind_off": false}); err != nil {
			t.Fatal(err)
		}
		// 归档的卡片不发送；恢复后已到达且未发送的通知立即发送。
		if err := Card.Archive(e.ctx, e.user.ID, card.ID); err != nil {
			t.Fatal(err)
		}
		if sent := e.run(time.Hour); len(sent) != 0 {
			t.Fatalf("归档后 = %+v", sent)
		}
		if _, err := Card.Restore(e.ctx, e.user.ID, card.ID, list.ID, listsort.Head); err != nil {
			t.Fatal(err)
		}
		if sent := e.run(time.Hour); len(sent) != 1 {
			t.Fatalf("恢复后 = %+v", sent)
		}
		// 已发送的记录保留：关闭再打开开关，同一时间值不重复发送。
		e.remindAt(card, e.now, false)
		e.run(time.Hour)
		e.remindAt(card, e.now, true)
		if sent := e.run(time.Hour); len(sent) != 0 {
			t.Fatalf("重新打开开关后重复发送：%+v", sent)
		}

		// 停用用户的卡片不发送，启用后补发。
		other := e.create("待办", "另一张", listsort.Tail)
		e.choose(other, channel)
		admin, _ := repository.User.FindByUsername(e.ctx, "admin")
		if err := User.Disable(e.ctx, admin.ID, e.user.ID); err != nil {
			t.Fatal(err)
		}
		e.remindAt(other, e.now, true)
		if sent := e.run(time.Hour); len(sent) != 0 {
			t.Fatalf("停用后 = %+v", sent)
		}
		if err := User.Enable(e.ctx, admin.ID, e.user.ID); err != nil {
			t.Fatal(err)
		}
		if sent := e.run(time.Hour); len(sent) != 1 {
			t.Fatalf("启用后 = %+v", sent)
		}

		// 从卡片上移除渠道：未发送的记录删除，不再发送。
		third := e.create("待办", "第三张", listsort.Tail)
		e.choose(third, channel)
		e.gmsg.fail("c", true)
		e.remindAt(third, e.now, true)
		e.run(time.Hour)
		if _, err := NotifyChannel.SetCardChannel(e.ctx, e.user.ID, third.ID, channel.ID, false); err != nil {
			t.Fatal(err)
		}
		if sent := e.run(2 * time.Hour); len(sent) != 0 || len(e.deliveries(third)) != 0 {
			t.Fatalf("移除渠道后 = %+v", sent)
		}
	})
}

func TestNotifyStopsAfterFolderArchiveAndListMove(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newNotifyEnv(t)
		channel := e.channel("渠道", "c", gmodel.TypeText)
		e.gmsg.fail("c", true)
		card := e.create("待办", "卡片", listsort.Tail)
		e.choose(card, channel)
		e.remindAt(card, e.now, true)
		e.run(0)

		// 再建一列并关闭提醒，跨列移入后不再发送，未发送的记录被清理。
		quiet, err := List.Create(e.ctx, e.user.ID, e.panel.ID, "安静")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = List.UpdateSettings(e.ctx, e.user.ID, quiet.ID, ListSettings{
			ShowAge: true, SortMode: listsort.Manual, SortDir: listsort.Asc, HeadAdd: listsort.Head, TailAdd: listsort.Tail, RemindOff: true,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err = Card.Move(e.ctx, e.user.ID, card.ID, quiet.ID, nil); err != nil {
			t.Fatal(err)
		}
		if sent := e.run(time.Hour); len(sent) != 0 || len(e.deliveries(card)) != 0 {
			t.Fatalf("移入关闭通知的列后 = %+v", sent)
		}

		other := e.create("待办", "另一张", listsort.Tail)
		e.choose(other, channel)
		e.remindAt(other, e.now, true)
		e.run(0)
		folder, err := Folder.Create(e.ctx, e.user.ID, nil, "工作", nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Board.Move(e.ctx, e.user.ID, e.panel.BoardID, &folder.ID, nil); err != nil {
			t.Fatal(err)
		}
		if err = Folder.Archive(e.ctx, e.user.ID, folder.ID); err != nil {
			t.Fatal(err)
		}
		if sent := e.run(time.Hour); len(sent) != 0 || len(e.deliveries(other)) != 0 {
			t.Fatalf("归档文件夹后 = %+v", sent)
		}
	})
}

func TestNotifyChannelCascades(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newNotifyEnv(t)
		channel := e.channel("渠道", "c", gmodel.TypeText)
		card := e.create("待办", "卡片", listsort.Tail)
		e.choose(card, channel)
		due := e.now
		if _, err := Card.SetDates(e.ctx, e.user.ID, card.ID, CardDates{DueAt: &due, DueNotify: true}); err != nil {
			t.Fatal(err)
		}
		if sent := e.run(0); len(sent) != 1 || sent[0].Title != "截止" {
			t.Fatalf("截止通知 = %+v", sent)
		}

		// 复制卡片带上所选渠道，发送状态从头开始。
		copied, err := Card.Copy(e.ctx, e.user.ID, card.ID, nil)
		if err != nil {
			t.Fatal(err)
		}
		view, _ := cardView(e.ctx, copied)
		if len(view.NotifyChannelIDs) != 1 || len(e.deliveries(copied)) != 0 {
			t.Fatalf("副本 = %+v", view)
		}
		if sent := e.run(time.Minute); len(sent) != 1 {
			t.Fatalf("副本的截止通知 = %+v", sent)
		}

		// 删除渠道：从卡片上移除，发送记录删除。
		if err = NotifyChannel.Delete(e.ctx, e.user.ID, channel.ID); err != nil {
			t.Fatal(err)
		}
		if view, _ = cardView(e.ctx, card); len(view.NotifyChannelIDs) != 0 || len(e.deliveries(card)) != 0 {
			t.Fatalf("删除渠道后 = %+v", view)
		}

		// 删除用户时一并删除通知数据。
		other := e.channel("另一条", "o", gmodel.TypeText)
		e.choose(card, other)
		admin, _ := repository.User.FindByUsername(e.ctx, "admin")
		if err = User.Delete(e.ctx, admin.ID, e.user.ID); err != nil {
			t.Fatal(err)
		}
		if channels, _ := repository.Notify.ListChannels(context.Background(), e.user.ID); len(channels) != 0 {
			t.Fatal("删除用户后渠道仍在")
		}
	})
}

func TestNotifyBodyFields(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		e := newNotifyEnv(t)
		card := e.create("待办", "整理预算", listsort.Tail)
		if _, err := Card.SetLabel(e.ctx, e.user.ID, card.ID, e.label("工作"), true); err != nil {
			t.Fatal(err)
		}
		if _, err := Card.SetLabel(e.ctx, e.user.ID, card.ID, e.label("紧急"), true); err != nil {
			t.Fatal(err)
		}
		if _, err := Task.Create(e.ctx, e.user.ID, card.ID, nil, []string{"一", "二"}); err != nil {
			t.Fatal(err)
		}
		at := time.Date(2026, 10, 3, 1, 30, 0, 0, time.UTC)
		if _, err := Card.SetDates(e.ctx, e.user.ID, card.ID, CardDates{RemindAt: &at}); err != nil {
			t.Fatal(err)
		}
		if _, err := Setting.Update(e.ctx, e.user.ID, SettingChanges{Timezone: ptr("Asia/Shanghai"), RemindTemplate: ptr("{{board}}/{{panel}}/{{list}} {{labels}} {{tasks}} {{priority}}|{{remind_at}}")}); err != nil {
			t.Fatal(err)
		}
		_, body, err := notifyMessage(e.ctx, e.user.ID, card.ID, model.NotifyRemind, false)
		if err != nil {
			t.Fatal(err)
		}
		if want := "整理预算\n工作/默认/待办 工作，紧急 0/2 |2026-10-03 09:30"; body != want {
			t.Fatalf("正文 = %q, want %q", body, want)
		}
	})
}
