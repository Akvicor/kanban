package service

import (
	"context"
	"errors"
	"kanban/cmd/app/server/common/timezone"
	"kanban/cmd/app/server/common/types/palette"
	"kanban/cmd/app/server/common/types/role"
	"kanban/cmd/app/server/model"
	"kanban/cmd/app/server/repository"
	"kanban/cmd/app/server/testutil/dbtest"
	"testing"
	"time"
)

const testPassword = "password-123"

// errorKind 返回业务错误的类别，非业务错误返回 0。
func errorKind(err error) Kind {
	var businessError *Error
	if errors.As(err, &businessError) {
		return businessError.Kind
	}
	return 0
}

// mustAdmin 创建初始管理员并返回。
func mustAdmin(t *testing.T) *model.User {
	t.Helper()
	ctx := context.Background()
	if _, err := User.EnsureInitialAdmin(ctx, "admin", testPassword); err != nil {
		t.Fatal(err)
	}
	admin, err := repository.User.FindByUsername(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	return admin
}

// mustUser 由管理员创建一个普通用户并返回。
func mustUser(t *testing.T, username string) *model.User {
	t.Helper()
	user, err := User.Create(context.Background(), username, testPassword)
	if err != nil {
		t.Fatal(err)
	}
	return user
}

// mustLogin 登录并返回令牌明文和会话。
func mustLogin(t *testing.T, username, device string) (string, *Session) {
	t.Helper()
	plain, session, err := Auth.Login(context.Background(), username, testPassword, device)
	if err != nil {
		t.Fatalf("登录 %s 失败: %v", username, err)
	}
	return plain, session
}

// deviceCount 返回用户当前的设备数。
func deviceCount(t *testing.T, userID int64) int {
	t.Helper()
	devices, err := repository.Device.ListByUser(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	return len(devices)
}

func TestEnsureInitialAdmin(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		if _, err := User.EnsureInitialAdmin(ctx, "", ""); errorKind(err) != KindBadRequest {
			t.Fatalf("库中无用户且未提供用户名密码时 err = %v", err)
		}
		created, err := User.EnsureInitialAdmin(ctx, "admin", testPassword)
		if err != nil || !created {
			t.Fatalf("创建初始管理员 created=%v err=%v", created, err)
		}
		// 已有用户时不再创建，也不校验参数。
		if created, err = User.EnsureInitialAdmin(ctx, "", ""); err != nil || created {
			t.Fatalf("已有用户时 created=%v err=%v", created, err)
		}
		admin, err := repository.User.FindByUsername(ctx, "admin")
		if err != nil {
			t.Fatal(err)
		}
		if admin.Role != role.Admin {
			t.Fatalf("初始管理员角色 = %s", admin.Role)
		}
	})
}

func TestCreateUserDefaults(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		mustAdmin(t)
		user := mustUser(t, "alice")
		saved, err := User.FindByID(context.Background(), user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if saved.Role != role.User || saved.Nickname != "alice" || saved.Palette != palette.Clean || saved.OpenMainBoardOnHome ||
			saved.Timezone != timezone.Server() || saved.Locale != "" || saved.RemindTemplate != "" ||
			saved.DueTemplate != "" || len(saved.Shortcuts) != 0 {
			t.Fatalf("新用户默认设置不正确: %+v", saved)
		}

		if _, err = User.Create(context.Background(), "alice", testPassword); errorKind(err) != KindConflict {
			t.Fatalf("重复用户名 err = %v", err)
		}
		for _, input := range [][2]string{{"", testPassword}, {"has space", testPassword}, {"bob", "short"}} {
			if _, err = User.Create(context.Background(), input[0], input[1]); errorKind(err) != KindBadRequest {
				t.Fatalf("Create(%q, %q) err = %v", input[0], input[1], err)
			}
		}
	})
}

func TestUpdateProfile(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		mustAdmin(t)
		alice := mustUser(t, "alice")
		mustUser(t, "bob")
		ctx := context.Background()

		saved, err := User.UpdateProfile(ctx, alice.ID, "alice2", " 小爱 ")
		if err != nil || saved.Username != "alice2" || saved.Nickname != "小爱" {
			t.Fatalf("修改后 = %+v, err = %v", saved, err)
		}
		if _, _, err = Auth.Login(ctx, "Alice2", testPassword, "phone"); err != nil {
			t.Fatalf("不区分大小写登录失败: %v", err)
		}
		if _, err = User.UpdateProfile(ctx, alice.ID, "BOB", "小爱"); errorKind(err) != KindConflict {
			t.Fatalf("占用他人用户名（忽略大小写）err = %v", err)
		}
		if _, err = User.UpdateProfile(ctx, alice.ID, "alice2", "   "); errorKind(err) != KindBadRequest {
			t.Fatalf("空昵称 err = %v", err)
		}
	})
}

func TestLoginAndAuthenticate(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")

		plain, session := mustLogin(t, "alice", "  Firefox · Linux  ")
		if session.Device.Name != "Firefox · Linux" {
			t.Fatalf("设备名 = %q", session.Device.Name)
		}
		got, err := Auth.Authenticate(ctx, plain)
		if err != nil || got.User.ID != alice.ID || got.Device.ID != session.Device.ID {
			t.Fatalf("Authenticate() = %+v, %v", got, err)
		}

		for _, input := range [][2]string{{"alice", "wrong-password"}, {"nobody", testPassword}} {
			if _, _, err = Auth.Login(ctx, input[0], input[1], ""); errorKind(err) != KindUnauthorized {
				t.Fatalf("Login(%q, %q) err = %v", input[0], input[1], err)
			}
		}
		if _, err = Auth.Authenticate(ctx, "invalid-token"); errorKind(err) != KindUnauthorized {
			t.Fatalf("无效令牌 err = %v", err)
		}

		if err = Auth.Logout(ctx, session); err != nil {
			t.Fatal(err)
		}
		if _, err = Auth.Authenticate(ctx, plain); errorKind(err) != KindUnauthorized {
			t.Fatalf("登出后令牌仍可用: %v", err)
		}
	})
}

func TestIdleDevicesExpire(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		idlePlain, idle := mustLogin(t, "alice", "旧手机")
		recentPlain, recent := mustLogin(t, "alice", "电脑")
		oldPlain, old := mustLogin(t, "alice", "旧平板")
		now := time.Now().UTC()
		// 旧手机和旧平板闲置超过期限，电脑在期限内。
		for id, at := range map[int64]time.Time{
			idle.Device.ID:   now.Add(-deviceIdleExpiry - time.Hour),
			recent.Device.ID: now.Add(-deviceIdleExpiry + time.Hour),
			old.Device.ID:    now.Add(-deviceIdleExpiry - time.Hour),
		} {
			if err := repository.Device.Touch(ctx, id, at); err != nil {
				t.Fatal(err)
			}
		}

		if _, err := Auth.Authenticate(ctx, idlePlain); errorKind(err) != KindUnauthorized {
			t.Fatalf("闲置超期的令牌 err = %v", err)
		}
		if deviceCount(t, alice.ID) != 2 {
			t.Fatal("认证时闲置超期的设备未被删除")
		}
		if _, err := Auth.Authenticate(ctx, recentPlain); err != nil {
			t.Fatalf("期限内的令牌 err = %v", err)
		}

		// 后台清理只删除闲置超期的设备。
		if err := Device.ExpireIdleDevices(ctx, now); err != nil {
			t.Fatal(err)
		}
		devices, err := repository.Device.ListByUser(ctx, alice.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(devices) != 1 || devices[0].ID != recent.Device.ID {
			t.Fatalf("清理后的设备 = %+v", devices)
		}
		if _, err = Auth.Authenticate(ctx, oldPlain); errorKind(err) != KindUnauthorized {
			t.Fatalf("被清理设备的令牌 err = %v", err)
		}
	})
}

func TestDisableAndEnable(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		admin := mustAdmin(t)
		alice := mustUser(t, "alice")
		plain, _ := mustLogin(t, "alice", "phone")

		if err := User.Disable(ctx, admin.ID, alice.ID); err != nil {
			t.Fatal(err)
		}
		if deviceCount(t, alice.ID) != 0 {
			t.Fatal("停用后设备令牌未吊销")
		}
		if _, err := Auth.Authenticate(ctx, plain); errorKind(err) != KindUnauthorized {
			t.Fatalf("停用后旧令牌仍可用: %v", err)
		}
		if _, _, err := Auth.Login(ctx, "alice", testPassword, ""); errorKind(err) != KindUnauthorized {
			t.Fatalf("停用后仍可登录: %v", err)
		}

		if err := User.Enable(ctx, admin.ID, alice.ID); err != nil {
			t.Fatal(err)
		}
		mustLogin(t, "alice", "phone")

		if err := User.Disable(ctx, admin.ID, admin.ID); errorKind(err) != KindForbidden {
			t.Fatalf("管理员停用自己 err = %v", err)
		}
	})
}

func TestPasswordChangesRevokeDevices(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		admin := mustAdmin(t)
		alice := mustUser(t, "alice")
		_, laptop := mustLogin(t, "alice", "laptop")
		phoneToken, _ := mustLogin(t, "alice", "phone")

		// 修改自己的密码：当前设备保留，其他设备吊销。
		if err := Setting.ChangePassword(ctx, laptop, "wrong-password", "new-password-1"); errorKind(err) != KindBadRequest {
			t.Fatalf("当前密码错误时 err = %v", err)
		}
		if err := Setting.ChangePassword(ctx, laptop, testPassword, "new-password-1"); err != nil {
			t.Fatal(err)
		}
		devices, err := repository.Device.ListByUser(ctx, alice.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(devices) != 1 || devices[0].ID != laptop.Device.ID {
			t.Fatalf("改密后剩余设备 = %+v", devices)
		}
		if _, err = Auth.Authenticate(ctx, phoneToken); errorKind(err) != KindUnauthorized {
			t.Fatalf("改密后其他设备仍可用: %v", err)
		}
		if _, _, err = Auth.Login(ctx, "alice", "new-password-1", ""); err != nil {
			t.Fatalf("新密码无法登录: %v", err)
		}

		// 管理员重置密码：全部设备吊销。
		if err = User.ResetPassword(ctx, admin.ID, alice.ID, "reset-password-1"); err != nil {
			t.Fatal(err)
		}
		if deviceCount(t, alice.ID) != 0 {
			t.Fatal("重置密码后仍有设备")
		}
		if _, _, err = Auth.Login(ctx, "alice", "reset-password-1", ""); err != nil {
			t.Fatalf("重置后的密码无法登录: %v", err)
		}
	})
}

func TestDeleteUserRemovesData(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		admin := mustAdmin(t)
		alice := mustUser(t, "alice")
		bob := mustUser(t, "bob")
		mustLogin(t, "alice", "laptop")
		mustLogin(t, "bob", "laptop")

		if err := User.Delete(ctx, admin.ID, alice.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := User.FindByID(ctx, alice.ID); errorKind(err) != KindNotFound {
			t.Fatalf("删除后仍能找到用户: %v", err)
		}
		if deviceCount(t, alice.ID) != 0 {
			t.Fatal("删除用户后设备记录残留")
		}
		if deviceCount(t, bob.ID) != 1 {
			t.Fatal("删除用户影响了其他用户的设备")
		}
		if err := User.Delete(ctx, admin.ID, alice.ID); errorKind(err) != KindNotFound {
			t.Fatalf("重复删除 err = %v", err)
		}
	})
}

func TestDevicesAreIsolatedByUser(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		mustAdmin(t)
		alice := mustUser(t, "alice")
		bob := mustUser(t, "bob")
		_, bobSession := mustLogin(t, "bob", "laptop")

		// alice 不能踢掉 bob 的设备，同步快照里也看不到它。
		if err := Device.Revoke(ctx, alice.ID, bobSession.Device.ID); errorKind(err) != KindNotFound {
			t.Fatalf("踢掉其他用户的设备 err = %v", err)
		}
		catchUp, err := Sync.CatchUp(ctx, alice.ID, 0)
		if err != nil || len(catchUp.Snapshot.Devices) != 0 {
			t.Fatalf("alice 的设备 = %+v, %v", catchUp, err)
		}
		if deviceCount(t, bob.ID) != 1 {
			t.Fatal("bob 的设备被误删")
		}
		if err = Device.Revoke(ctx, bob.ID, bobSession.Device.ID); err != nil {
			t.Fatal(err)
		}
	})
}
