package migrate

import (
	"context"
	"fmt"
	"kanban/cmd/app/server/service"
	"os"

	"github.com/Akvicor/glog"
)

// 初始管理员的用户名和密码来自这两个环境变量，只在库中还没有任何用户时使用。
const (
	EnvAdminUsername = "KANBAN_ADMIN_USERNAME"
	EnvAdminPassword = "KANBAN_ADMIN_PASSWORD"
)

// initAdmin 在库中没有用户时创建初始管理员。没有用户且环境变量未设置时返回错误，避免启动一个无法登录的服务。
func initAdmin(ctx context.Context) error {
	created, err := service.User.EnsureInitialAdmin(ctx, os.Getenv(EnvAdminUsername), os.Getenv(EnvAdminPassword))
	if err != nil {
		return fmt.Errorf("创建初始管理员失败（请通过 %s 和 %s 提供用户名和密码）: %w", EnvAdminUsername, EnvAdminPassword, err)
	}
	if created {
		glog.Info("已创建初始管理员 %s", os.Getenv(EnvAdminUsername))
	}
	return nil
}
