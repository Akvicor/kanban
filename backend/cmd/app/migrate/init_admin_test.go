package migrate

import (
	"context"
	"kanban/cmd/app/server/repository"
	"kanban/cmd/app/server/testutil/dbtest"
	"testing"
)

// TestInitAdmin 验证初始管理员只在库中没有用户时按环境变量创建，环境变量缺失时报错。
func TestInitAdmin(t *testing.T) {
	dbtest.Run(t, func(t *testing.T) {
		ctx := context.Background()
		t.Setenv(EnvAdminUsername, "")
		t.Setenv(EnvAdminPassword, "")
		if err := initAdmin(ctx); err == nil {
			t.Fatal("库中无用户且未设置环境变量时未报错")
		}

		t.Setenv(EnvAdminUsername, "root")
		t.Setenv(EnvAdminPassword, "password-123")
		if err := initAdmin(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := repository.User.FindByUsername(ctx, "root"); err != nil {
			t.Fatalf("初始管理员未创建: %v", err)
		}

		// 已有用户后，环境变量改变或缺失都不再影响。
		t.Setenv(EnvAdminUsername, "")
		if err := initAdmin(ctx); err != nil {
			t.Fatalf("已有用户时报错: %v", err)
		}
		count, err := repository.User.Count(ctx)
		if err != nil || count != 1 {
			t.Fatalf("用户数 = %d, %v", count, err)
		}
	})
}
