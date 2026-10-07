package service

import (
	"context"
	"encoding/json"
	"kanban/cmd/app/server/common/types/palette"
	"kanban/cmd/app/server/global/hub"
	"kanban/cmd/app/server/repository"
	"kanban/cmd/app/server/testutil/dbtest"
	"sync"
	"testing"
)

// recorder 是登记到 hub.Default 的测试连接，记录收到的推送和关闭码。
type recorder struct {
	mu     sync.Mutex
	events []hub.Event
	closed int
}

func (r *recorder) Deliver(events []hub.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, events...)
}

func (r *recorder) Close(code int, _ string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = code
}

func listen(t *testing.T, userID, deviceID int64) *recorder {
	t.Helper()
	r := &recorder{}
	t.Cleanup(hub.Default.Register(userID, deviceID, r))
	return r
}

func TestWritesRecordAndPublishChanges(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		mustAdmin(t)
		alice := mustUser(t, "alice")
		_, laptop := mustLogin(t, "alice", "laptop") // 序号 1：设备 laptop
		_, phone := mustLogin(t, "alice", "phone")   // 序号 2：设备 phone
		laptopConn := listen(t, alice.ID, laptop.Device.ID)
		phoneConn := listen(t, alice.ID, phone.Device.ID)

		ctx := WithRevisionNote(context.Background(), alice.ID)
		if _, err := Setting.Update(ctx, alice.ID, SettingChanges{Palette: ptr(palette.Dark)}); err != nil {
			t.Fatal(err)
		}
		if Revision(ctx) != 3 {
			t.Fatalf("响应序号 = %d，应为 3", Revision(ctx))
		}
		if len(laptopConn.events) != 1 || laptopConn.events[0].Type != EntitySettings || laptopConn.events[0].Revision != 3 {
			t.Fatalf("laptop 收到 %+v", laptopConn.events)
		}
		var settings struct {
			Palette string `json:"palette"`
		}
		if err := json.Unmarshal(laptopConn.events[0].Data, &settings); err != nil || settings.Palette != "dark" {
			t.Fatalf("推送的设置 = %s, %v", laptopConn.events[0].Data, err)
		}

		// 改密码：吊销 phone，记录删除变更，并在提交后断开 phone 的连接。
		if err := Setting.ChangePassword(context.Background(), laptop, testPassword, "new-password-1"); err != nil {
			t.Fatal(err)
		}
		last := laptopConn.events[len(laptopConn.events)-1]
		if last.Op != hub.OpDelete || last.Type != EntityDevice || last.ID != phone.Device.ID || last.Revision != 4 {
			t.Fatalf("改密后的推送 = %+v", last)
		}
		if phoneConn.closed != hub.CloseRevoked || laptopConn.closed != 0 {
			t.Fatalf("关闭码: phone=%d laptop=%d", phoneConn.closed, laptopConn.closed)
		}

		// 失败的写入不推送、不占用序号。
		before := len(laptopConn.events)
		if _, err := Setting.Update(context.Background(), alice.ID, SettingChanges{Timezone: ptr("Mars/Base")}); err == nil {
			t.Fatal("无效时区未报错")
		}
		if len(laptopConn.events) != before {
			t.Fatal("失败的写入产生了推送")
		}
		user, err := User.FindByID(context.Background(), alice.ID)
		if err != nil || user.Revision != 4 {
			t.Fatalf("当前序号 = %d, %v", user.Revision, err)
		}
	})
}

func TestConcurrentWritesGetContiguousRevisions(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		mustAdmin(t)
		alice := mustUser(t, "alice")
		const writers = 20
		var wg sync.WaitGroup
		errs := make(chan error, writers)
		for i := range writers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				value := i%2 == 0
				_, err := Setting.Update(context.Background(), alice.ID, SettingChanges{OpenMainBoardOnHome: &value})
				errs <- err
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatalf("并发写入失败: %v", err)
			}
		}
		logs, err := repository.ChangeLog.ListAfter(context.Background(), alice.ID, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(logs) != writers {
			t.Fatalf("变更记录 %d 条，应为 %d", len(logs), writers)
		}
		for i, log := range logs {
			if log.Revision != int64(i+1) {
				t.Fatalf("第 %d 条序号为 %d，序号不连续", i+1, log.Revision)
			}
		}
	})
}

func TestCatchUp(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		mustLogin(t, "alice", "laptop") // 序号 1
		for _, p := range []palette.Type{palette.Dark, palette.Paper, palette.System} {
			if _, err := Setting.Update(ctx, alice.ID, SettingChanges{Palette: &p}); err != nil {
				t.Fatal(err)
			}
		} // 序号 2–4

		check := func(last int64, wantSnapshot bool, wantEvents int) {
			t.Helper()
			result, err := Sync.CatchUp(ctx, alice.ID, last)
			if err != nil {
				t.Fatal(err)
			}
			if result.Revision != 4 || (result.Snapshot != nil) != wantSnapshot || len(result.Events) != wantEvents {
				t.Fatalf("CatchUp(%d) = 序号 %d，快照 %v，变更 %d 条", last, result.Revision, result.Snapshot != nil, len(result.Events))
			}
			if wantSnapshot && (result.Snapshot.Settings.Palette != palette.System || len(result.Snapshot.Devices) != 1) {
				t.Fatalf("快照内容 = %+v", result.Snapshot)
			}
			if wantEvents > 0 && result.Events[0].Revision != last+1 {
				t.Fatalf("第一条变更序号 = %d", result.Events[0].Revision)
			}
		}
		check(0, true, 0)  // 没有本地数据
		check(4, false, 0) // 已是最新
		check(1, false, 3) // 落后 3 条
		check(9, true, 0)  // 比服务端还新

		// 早期记录被清理后，落后的客户端收到快照。
		if err := repository.ChangeLog.DeleteUpTo(ctx, alice.ID, 2); err != nil {
			t.Fatal(err)
		}
		check(1, true, 0)
		check(2, false, 2)
	})
}

func TestTrimChangeLog(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		for range 3 {
			if _, err := Setting.Update(ctx, alice.ID, SettingChanges{OpenMainBoardOnHome: ptr(true)}); err != nil {
				t.Fatal(err)
			}
		}
		// 未到清理点时不删除；到达清理点时删除保留范围之前的记录。
		if err := trimChangeLog(ctx, alice.ID, changeLogRetention+1); err != nil {
			t.Fatal(err)
		}
		if logs, _ := repository.ChangeLog.ListAfter(ctx, alice.ID, 0); len(logs) != 3 {
			t.Fatalf("未到清理点时剩余 %d 条", len(logs))
		}
		if err := trimChangeLog(ctx, alice.ID, changeLogRetention+changeLogTrimEvery); err != nil {
			t.Fatal(err)
		}
		if logs, _ := repository.ChangeLog.ListAfter(ctx, alice.ID, 0); len(logs) != 0 {
			t.Fatalf("到达清理点后剩余 %d 条", len(logs))
		}
	})
}

func TestDeleteUserRemovesChangeLogAndDisconnects(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		admin := mustAdmin(t)
		alice := mustUser(t, "alice")
		_, session := mustLogin(t, "alice", "laptop")
		conn := listen(t, alice.ID, session.Device.ID)

		if err := User.Delete(ctx, admin.ID, alice.ID); err != nil {
			t.Fatal(err)
		}
		if conn.closed != hub.CloseRevoked {
			t.Fatalf("删除用户后连接关闭码 = %d", conn.closed)
		}
		if logs, _ := repository.ChangeLog.ListAfter(ctx, alice.ID, 0); len(logs) != 0 {
			t.Fatalf("删除用户后变更记录残留 %d 条", len(logs))
		}
	})
}
